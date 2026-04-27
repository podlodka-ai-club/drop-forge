package archiverunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
)

const defaultTempPattern = "orchv3-archive-*"

type Runner struct {
	Config  config.ProposalRunnerConfig
	Command commandrunner.Runner
	Agent   AgentExecutor
	Service string
	Stdout  io.Writer
	Stderr  io.Writer

	MkdirTemp func(dir string, pattern string) (string, error)
	RemoveAll func(path string) error
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
	stdout := writerOrDiscard(runner.Stdout)
	stderr := writerOrDiscard(runner.Stderr)
	logger := steplog.NewWithService(stdout, runner.Service)
	defer func() {
		if err != nil {
			logger.Errorf("archiverunner", "workflow failed: %v", err)
		}
	}()

	tempDir, err := runner.mkdirTemp("", defaultTempPattern)
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	logger.Infof("temp", "created %s", tempDir)
	defer func() {
		if !runner.Config.CleanupTemp {
			logger.Infof("temp", "preserving %s", tempDir)
			return
		}

		if cleanupErr := runner.removeAll(tempDir); cleanupErr != nil {
			logger.Errorf("temp", "cleanup failed for %s: %v", tempDir, cleanupErr)
			if err == nil {
				err = fmt.Errorf("cleanup temp dir %s: %w", tempDir, cleanupErr)
			}
			return
		}

		logger.Infof("temp", "removed %s", tempDir)
	}()

	cloneDir := filepath.Join(tempDir, "repo")
	logger.Infof("git", "cloning %s into %s", runner.Config.RepositoryURL, cloneDir)
	if err := runLoggedCommand(ctx, runner.Service, command, commandrunner.Command{
		Name: runner.Config.GitPath,
		Args: []string{"clone", runner.Config.RepositoryURL, cloneDir},
		Dir:  tempDir,
	}, "git", stdout, stderr); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	branchName, err := runner.resolveBranch(ctx, command, cloneDir, input, stdout, stderr)
	if err != nil {
		return err
	}
	logger.Infof("git", "checkout branch=%s", branchName)
	if err := runLoggedCommand(ctx, runner.Service, command, commandrunner.Command{
		Name: runner.Config.GitPath,
		Args: []string{"checkout", branchName},
		Dir:  cloneDir,
	}, "git", stdout, stderr); err != nil {
		return fmt.Errorf("git checkout %s: %w", branchName, err)
	}

	if _, err := runner.agentExecutor(command).Run(ctx, AgentExecutionInput{
		TaskDescription: strings.TrimSpace(input.AgentPrompt),
		CloneDir:        cloneDir,
		TempDir:         tempDir,
		Stdout:          stdout,
		Stderr:          stderr,
	}); err != nil {
		return fmt.Errorf("agent archive: %w", err)
	}

	status, err := runner.gitStatus(ctx, command, cloneDir, stdout, stderr)
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return errors.New("git status: no changes produced by agent")
	}
	logger.Infof("git", "status:\n%s", strings.TrimRight(status, "\n"))

	commitMessage := BuildCommitMessage(input.Identifier, input.Title)
	for _, gitCommand := range []commandrunner.Command{
		{Name: runner.Config.GitPath, Args: []string{"add", "-A"}, Dir: cloneDir},
		{Name: runner.Config.GitPath, Args: []string{"commit", "-m", commitMessage}, Dir: cloneDir},
		{Name: runner.Config.GitPath, Args: []string{"push", runner.Config.RemoteName, branchName}, Dir: cloneDir},
	} {
		logger.Infof("git", "%s", strings.Join(append([]string{gitCommand.Name}, gitCommand.Args...), " "))
		if err := runLoggedCommand(ctx, runner.Service, command, gitCommand, "git", stdout, stderr); err != nil {
			return fmt.Errorf("git %s: %w", gitCommand.Args[0], err)
		}
	}

	return nil
}

func BuildCommitMessage(identifier string, title string) string {
	displayName := proposalrunner.BuildDisplayName(identifier, title)
	return proposalrunner.BuildPRTitle("Archive:", displayName)
}

func (runner *Runner) resolveBranch(ctx context.Context, command commandrunner.Runner, cloneDir string, input ArchiveInput, stdout io.Writer, stderr io.Writer) (string, error) {
	if branch := strings.TrimSpace(input.BranchName); branch != "" {
		return branch, nil
	}

	prURL := strings.TrimSpace(input.PRURL)
	var branchOutput bytes.Buffer
	args := []string{"pr", "view", prURL, "--json", "headRefName", "--jq", ".headRefName"}
	steplog.NewWithService(writerOrDiscard(stdout), runner.Service).Infof("github", "%s %s", runner.Config.GHPath, strings.Join(args, " "))
	if err := runLoggedCommand(ctx, runner.Service, command, commandrunner.Command{
		Name:   runner.Config.GHPath,
		Args:   args,
		Dir:    cloneDir,
		Stdout: &branchOutput,
	}, "github", stdout, stderr); err != nil {
		return "", fmt.Errorf("github resolve pr branch %s: %w", prURL, err)
	}

	branch := strings.TrimSpace(branchOutput.String())
	if branch == "" {
		return "", fmt.Errorf("github resolve pr branch %s: empty headRefName", prURL)
	}

	return branch, nil
}

func (runner *Runner) gitStatus(ctx context.Context, command commandrunner.Runner, cloneDir string, stdout io.Writer, stderr io.Writer) (string, error) {
	var status bytes.Buffer
	err := runLoggedCommand(ctx, runner.Service, command, commandrunner.Command{
		Name:   runner.Config.GitPath,
		Args:   []string{"status", "--short"},
		Dir:    cloneDir,
		Stdout: &status,
	}, "git", stdout, stderr)
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}

	return status.String(), nil
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

func runLoggedCommand(ctx context.Context, service string, exec commandrunner.Runner, command commandrunner.Command, module string, stdout io.Writer, stderr io.Writer) error {
	stdoutLog := steplog.NewWithService(writerOrDiscard(stdout), service).LineWriter(module)
	stderrLog := steplog.NewWithService(writerOrDiscard(stderr), service).LineWriter(module)

	stdoutWriters := []io.Writer{stdoutLog}
	if command.Stdout != nil {
		stdoutWriters = append(stdoutWriters, command.Stdout)
	}
	command.Stdout = io.MultiWriter(stdoutWriters...)

	stderrWriters := []io.Writer{stderrLog}
	if command.Stderr != nil {
		stderrWriters = append(stderrWriters, command.Stderr)
	}
	command.Stderr = io.MultiWriter(stderrWriters...)

	err := exec.Run(ctx, command)
	stdoutLog.Flush()
	stderrLog.Flush()

	return err
}

func (runner *Runner) mkdirTemp(dir string, pattern string) (string, error) {
	if runner.MkdirTemp != nil {
		return runner.MkdirTemp(dir, pattern)
	}

	return os.MkdirTemp(dir, pattern)
}

func (runner *Runner) removeAll(path string) error {
	if runner.RemoveAll != nil {
		return runner.RemoveAll(path)
	}

	return os.RemoveAll(path)
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
