package proposalrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/gitmanager"
	"orchv3/internal/steplog"
)

const (
	defaultTempPattern = "orchv3-proposal-*"
	maxSlugLength      = 48
)

type Runner struct {
	Config   config.ProposalRunnerConfig
	Command  commandrunner.Runner
	Agent    AgentExecutor
	Git      GitManager
	Producer agentmeta.Producer
	Service  string
	Stdout   io.Writer
	Stderr   io.Writer

	MkdirTemp func(dir string, pattern string) (string, error)
	RemoveAll func(path string) error
	Now       func() time.Time
}

type GitManager interface {
	Clone(ctx context.Context) (gitmanager.Workspace, error)
	Close(workspace gitmanager.Workspace) error
	StatusShort(ctx context.Context, cloneDir string) (string, error)
	CheckoutNewBranch(ctx context.Context, cloneDir string, branchName string) error
	CommitAllAndPush(ctx context.Context, cloneDir string, branchName string, message string, setUpstream bool) error
	CreatePullRequest(ctx context.Context, cloneDir string, request gitmanager.PullRequest) (string, error)
	CommentPullRequest(ctx context.Context, cloneDir string, prURL string, body string) error
}

type ProposalInput struct {
	Title       string
	Identifier  string
	AgentPrompt string
}

func (input ProposalInput) validate() error {
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("proposal input title must not be empty")
	}
	if strings.TrimSpace(input.AgentPrompt) == "" {
		return errors.New("proposal input agent prompt must not be empty")
	}

	return nil
}

func (runner *Runner) newLogger(w io.Writer) steplog.Logger {
	return steplog.NewWithService(w, runner.Service)
}

func New(cfg config.ProposalRunnerConfig) *Runner {
	return &Runner{
		Config:    cfg,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		MkdirTemp: os.MkdirTemp,
		RemoveAll: os.RemoveAll,
		Now:       time.Now,
	}
}

func (runner *Runner) Run(ctx context.Context, input ProposalInput) (prURL string, err error) {
	if err := input.validate(); err != nil {
		return "", err
	}

	if err := runner.Config.Validate(); err != nil {
		return "", fmt.Errorf("validate proposal runner config: %w", err)
	}

	displayName := BuildDisplayName(input.Identifier, input.Title)
	agentPrompt := strings.TrimSpace(input.AgentPrompt)

	command := runner.commandRunner()
	git := runner.gitManager(command)
	stdout := writerOrDiscard(runner.Stdout)
	logger := runner.newLogger(stdout)
	defer func() {
		if err != nil {
			logger.Errorf("proposalrunner", "workflow failed: %v", err)
		}
	}()

	workspace, err := git.Clone(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		if cleanupErr := git.Close(workspace); cleanupErr != nil {
			if err == nil {
				err = cleanupErr
			}
		}
	}()

	agentResult, err := runner.agentExecutor(command).Run(ctx, AgentExecutionInput{
		TaskDescription: agentPrompt,
		CloneDir:        workspace.CloneDir,
		TempDir:         workspace.TempDir,
		Stdout:          stdout,
		Stderr:          writerOrDiscard(runner.Stderr),
	})
	if err != nil {
		return "", fmt.Errorf("agent proposal: %w", err)
	}

	status, err := git.StatusShort(ctx, workspace.CloneDir)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(status) == "" {
		return "", errors.New("git status: no changes produced by agent")
	}
	logger.Infof("git", "status:\n%s", strings.TrimRight(status, "\n"))

	branchName := BuildBranchName(runner.Config.BranchPrefix, displayName, runner.now())
	prTitle := BuildPRTitle(runner.Config.PRTitlePrefix, displayName)
	prBody := BuildPRBody(agentPrompt)

	if err := git.CheckoutNewBranch(ctx, workspace.CloneDir, branchName); err != nil {
		return "", err
	}
	commitMessage := prTitle
	if runner.Producer != (agentmeta.Producer{}) {
		commitMessage = agentmeta.AppendTrailer(prTitle, runner.Producer)
	}
	if err := git.CommitAllAndPush(ctx, workspace.CloneDir, branchName, commitMessage, true); err != nil {
		return "", err
	}

	prURL, err = git.CreatePullRequest(ctx, workspace.CloneDir, gitmanager.PullRequest{
		BaseBranch: runner.Config.BaseBranch,
		HeadBranch: branchName,
		Title:      prTitle,
		Body:       prBody,
	})
	if err != nil {
		return "", err
	}
	logger.Infof("github", "created PR %s", prURL)

	if err := git.CommentPullRequest(ctx, workspace.CloneDir, prURL, agentResult.FinalMessage); err != nil {
		return "", err
	}

	return prURL, nil
}

func BuildBranchName(prefix string, displayName string, now time.Time) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "codex/proposal"
	}

	timestamp := now.UTC().Format("20060102150405")
	return prefix + "/" + timestamp + "-" + slugify(displayName)
}

func BuildPRTitle(prefix string, displayName string) string {
	prefix = strings.TrimSpace(prefix)
	displayName = strings.TrimSpace(displayName)
	runes := []rune(displayName)
	if len(runes) > 72 {
		displayName = strings.TrimSpace(string(runes[:72]))
	}

	if prefix == "" {
		return displayName
	}

	return strings.TrimSpace(prefix + " " + displayName)
}

func BuildPRBody(agentPrompt string) string {
	return fmt.Sprintf("Generated by orchv3 proposal runner.\n\nTask:\n%s\n", strings.TrimSpace(agentPrompt))
}

// BuildDisplayName composes the human-readable name used to derive PR title,
// branch slug and commit message from a proposal task. It normalizes the title
// (collapsing newlines into spaces and trimming whitespace), then prefixes the
// optional Linear identifier when present.
func BuildDisplayName(identifier string, title string) string {
	title = normalizeTitle(title)
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return title
	}

	return identifier + ": " + title
}

func normalizeTitle(value string) string {
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return strings.TrimSpace(value)
}

func (runner *Runner) commandRunner() commandrunner.Runner {
	if runner.Command != nil {
		return runner.Command
	}

	return commandrunner.ExecRunner{LogWriter: writerOrDiscard(runner.Stdout)}
}

func (runner *Runner) agentExecutor(command commandrunner.Runner) AgentExecutor {
	if runner.Agent != nil {
		return runner.Agent
	}

	return CodexCLIExecutor{
		Config:  runner.Config,
		Command: command,
		Service: runner.Service,
	}
}

func (runner *Runner) gitManager(command commandrunner.Runner) GitManager {
	if runner.Git != nil {
		return runner.Git
	}

	manager := gitmanager.NewFromProposalConfig(runner.Config)
	manager.Config.TempPattern = defaultTempPattern
	manager.Command = command
	manager.Service = runner.Service
	manager.Stdout = writerOrDiscard(runner.Stdout)
	manager.Stderr = writerOrDiscard(runner.Stderr)
	manager.MkdirTemp = runner.MkdirTemp
	manager.RemoveAll = runner.RemoveAll
	return manager
}

func (runner *Runner) now() time.Time {
	if runner.Now != nil {
		return runner.Now()
	}

	return time.Now()
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	value = re.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		value = "task"
	}
	if len(value) > maxSlugLength {
		value = strings.Trim(value[:maxSlugLength], "-")
	}

	return value
}
