package archiverunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/gitmanager"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
)

const defaultTempPattern = "orchv3-archive-*"

type Runner struct {
	Config  config.ProposalRunnerConfig
	Command commandrunner.Runner
	Agent   AgentExecutor
	Git     GitManager
	Service string
	Stdout  io.Writer
	Stderr  io.Writer

	MkdirTemp func(dir string, pattern string) (string, error)
	RemoveAll func(path string) error
}

type GitManager interface {
	Clone(ctx context.Context) (gitmanager.Workspace, error)
	Close(workspace gitmanager.Workspace) error
	ResolvePullRequestBranch(ctx context.Context, cloneDir string, prURL string) (string, error)
	Checkout(ctx context.Context, cloneDir string, branchName string) error
	StatusShort(ctx context.Context, cloneDir string) (string, error)
	CommitAllAndPush(ctx context.Context, cloneDir string, branchName string, message string, setUpstream bool) error
}

type ArchiveInput struct {
	TaskID      string
	Identifier  string
	Title       string
	AgentPrompt string
	PRURL       string
	BranchName  string
}

func New(cfg config.ProposalRunnerConfig) *Runner {
	return &Runner{
		Config:    cfg,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		MkdirTemp: os.MkdirTemp,
		RemoveAll: os.RemoveAll,
	}
}

func (input ArchiveInput) validate() error {
	if strings.TrimSpace(input.Title) == "" {
		return errors.New("archive input title must not be empty")
	}
	if strings.TrimSpace(input.AgentPrompt) == "" {
		return errors.New("archive input agent prompt must not be empty")
	}
	if strings.TrimSpace(input.PRURL) == "" && strings.TrimSpace(input.BranchName) == "" {
		return errors.New("archive input branch source must include pr url or branch name")
	}

	return nil
}

func (runner *Runner) Run(ctx context.Context, input ArchiveInput) (err error) {
	if err := input.validate(); err != nil {
		return err
	}
	if err := validateConfig(runner.Config); err != nil {
		return fmt.Errorf("validate archive runner config: %w", err)
	}

	command := runner.commandRunner()
	git := runner.gitManager(command)
	stdout := writerOrDiscard(runner.Stdout)
	logger := steplog.NewWithService(stdout, runner.Service)
	defer func() {
		if err != nil {
			logger.Errorf("archiverunner", "workflow failed: %v", err)
		}
	}()

	workspace, err := git.Clone(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := git.Close(workspace); cleanupErr != nil {
			if err == nil {
				err = cleanupErr
			}
		}
	}()

	branchName, err := runner.resolveBranch(ctx, git, workspace.CloneDir, input)
	if err != nil {
		return err
	}
	if err := git.Checkout(ctx, workspace.CloneDir, branchName); err != nil {
		return err
	}

	if _, err := runner.agentExecutor(command).Run(ctx, AgentExecutionInput{
		TaskDescription: strings.TrimSpace(input.AgentPrompt),
		CloneDir:        workspace.CloneDir,
		TempDir:         workspace.TempDir,
		Stdout:          stdout,
		Stderr:          writerOrDiscard(runner.Stderr),
	}); err != nil {
		return fmt.Errorf("agent archive: %w", err)
	}

	status, err := git.StatusShort(ctx, workspace.CloneDir)
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return errors.New("git status: no changes produced by agent")
	}
	logger.Infof("git", "status:\n%s", strings.TrimRight(status, "\n"))

	commitMessage := BuildCommitMessage(input.Identifier, input.Title)
	if err := git.CommitAllAndPush(ctx, workspace.CloneDir, branchName, commitMessage, false); err != nil {
		return err
	}

	return nil
}

func BuildCommitMessage(identifier string, title string) string {
	displayName := proposalrunner.BuildDisplayName(identifier, title)
	return proposalrunner.BuildPRTitle("Archive:", displayName)
}

func (runner *Runner) resolveBranch(ctx context.Context, git GitManager, cloneDir string, input ArchiveInput) (string, error) {
	if branch := strings.TrimSpace(input.BranchName); branch != "" {
		return branch, nil
	}

	return git.ResolvePullRequestBranch(ctx, cloneDir, input.PRURL)
}

func validateConfig(cfg config.ProposalRunnerConfig) error {
	if strings.TrimSpace(cfg.RepositoryURL) == "" {
		return errors.New("PROPOSAL_REPOSITORY_URL must not be empty")
	}

	requiredValues := map[string]string{
		"PROPOSAL_REMOTE_NAME": cfg.RemoteName,
		"PROPOSAL_GIT_PATH":    cfg.GitPath,
		"PROPOSAL_CODEX_PATH":  cfg.CodexPath,
		"PROPOSAL_GH_PATH":     cfg.GHPath,
	}
	for key, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", key)
		}
	}

	return nil
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

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
