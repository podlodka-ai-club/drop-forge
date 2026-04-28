package gitmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

const defaultTempPattern = "orchv3-git-*"

type Config struct {
	RepositoryURL string
	BaseBranch    string
	RemoteName    string
	CleanupTemp   bool
	GitPath       string
	GHPath        string
	TempPattern   string
}

type Manager struct {
	Config  Config
	Command commandrunner.Runner
	Service string
	Stdout  io.Writer
	Stderr  io.Writer

	MkdirTemp func(dir string, pattern string) (string, error)
	RemoveAll func(path string) error
}

type Workspace struct {
	TempDir  string
	CloneDir string
}

type PullRequest struct {
	BaseBranch string
	HeadBranch string
	Title      string
	Body       string
}

func ConfigFromProposal(cfg config.ProposalRunnerConfig) Config {
	return Config{
		RepositoryURL: cfg.RepositoryURL,
		BaseBranch:    cfg.BaseBranch,
		RemoteName:    cfg.RemoteName,
		CleanupTemp:   cfg.CleanupTemp,
		GitPath:       cfg.GitPath,
		GHPath:        cfg.GHPath,
		TempPattern:   defaultTempPattern,
	}
}

func NewFromProposalConfig(cfg config.ProposalRunnerConfig) *Manager {
	return &Manager{
		Config:    ConfigFromProposal(cfg),
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		MkdirTemp: os.MkdirTemp,
		RemoveAll: os.RemoveAll,
	}
}

func (manager *Manager) Clone(ctx context.Context) (Workspace, error) {
	tempDir, err := manager.mkdirTemp("", manager.tempPattern())
	if err != nil {
		return Workspace{}, fmt.Errorf("create temp dir: %w", err)
	}

	stdout := writerOrDiscard(manager.Stdout)
	stderr := writerOrDiscard(manager.Stderr)
	logger := manager.logger(stdout)
	logger.Infof("temp", "created %s", tempDir)

	cloneDir := filepath.Join(tempDir, "repo")
	logger.Infof("git", "cloning %s into %s", manager.Config.RepositoryURL, cloneDir)
	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name: manager.Config.GitPath,
		Args: []string{"clone", manager.Config.RepositoryURL, cloneDir},
		Dir:  tempDir,
	}, "git", stdout, stderr); err != nil {
		return Workspace{}, fmt.Errorf("git clone: %w", err)
	}

	return Workspace{TempDir: tempDir, CloneDir: cloneDir}, nil
}

func (manager *Manager) Close(workspace Workspace) error {
	tempDir := strings.TrimSpace(workspace.TempDir)
	if tempDir == "" {
		return nil
	}

	logger := manager.logger(writerOrDiscard(manager.Stdout))
	if !manager.Config.CleanupTemp {
		logger.Infof("temp", "preserving %s", tempDir)
		return nil
	}

	if err := manager.removeAll(tempDir); err != nil {
		logger.Errorf("temp", "cleanup failed for %s: %v", tempDir, err)
		return fmt.Errorf("cleanup temp dir %s: %w", tempDir, err)
	}

	logger.Infof("temp", "removed %s", tempDir)
	return nil
}

func (manager *Manager) StatusShort(ctx context.Context, cloneDir string) (string, error) {
	var status bytes.Buffer
	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name:   manager.Config.GitPath,
		Args:   []string{"status", "--short"},
		Dir:    cloneDir,
		Stdout: &status,
	}, "git", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}

	return status.String(), nil
}

func (manager *Manager) Checkout(ctx context.Context, cloneDir string, branchName string) error {
	branchName = strings.TrimSpace(branchName)
	manager.logger(writerOrDiscard(manager.Stdout)).Infof("git", "checkout branch=%s", branchName)
	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name: manager.Config.GitPath,
		Args: []string{"checkout", branchName},
		Dir:  cloneDir,
	}, "git", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
		return fmt.Errorf("git checkout %s: %w", branchName, err)
	}

	return nil
}

func (manager *Manager) CheckoutNewBranch(ctx context.Context, cloneDir string, branchName string) error {
	branchName = strings.TrimSpace(branchName)
	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name: manager.Config.GitPath,
		Args: []string{"checkout", "-b", branchName},
		Dir:  cloneDir,
	}, "git", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
		return fmt.Errorf("git checkout %s: %w", branchName, err)
	}

	return nil
}

func (manager *Manager) CommitAllAndPush(ctx context.Context, cloneDir string, branchName string, message string, setUpstream bool) error {
	commands := []commandrunner.Command{
		{Name: manager.Config.GitPath, Args: []string{"add", "-A"}, Dir: cloneDir},
		{Name: manager.Config.GitPath, Args: []string{"commit", "-m", message}, Dir: cloneDir},
	}
	pushArgs := []string{"push", manager.Config.RemoteName, branchName}
	if setUpstream {
		pushArgs = []string{"push", "-u", manager.Config.RemoteName, branchName}
	}
	commands = append(commands, commandrunner.Command{Name: manager.Config.GitPath, Args: pushArgs, Dir: cloneDir})

	logger := manager.logger(writerOrDiscard(manager.Stdout))
	for _, gitCommand := range commands {
		logger.Infof("git", "%s", strings.Join(append([]string{gitCommand.Name}, gitCommand.Args...), " "))
		if err := manager.runLoggedCommand(ctx, gitCommand, "git", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
			return fmt.Errorf("git %s: %w", gitCommand.Args[0], err)
		}
	}

	return nil
}

func (manager *Manager) ResolvePullRequestBranch(ctx context.Context, cloneDir string, prURL string) (string, error) {
	prURL = strings.TrimSpace(prURL)
	var branchOutput bytes.Buffer
	args := []string{"pr", "view", prURL, "--json", "headRefName", "--jq", ".headRefName"}
	manager.logger(writerOrDiscard(manager.Stdout)).Infof("github", "%s %s", manager.Config.GHPath, strings.Join(args, " "))
	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name:   manager.Config.GHPath,
		Args:   args,
		Dir:    cloneDir,
		Stdout: &branchOutput,
	}, "github", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
		return "", fmt.Errorf("github resolve pr branch %s: %w", prURL, err)
	}

	branch := strings.TrimSpace(branchOutput.String())
	if branch == "" {
		return "", fmt.Errorf("github resolve pr branch %s: empty headRefName", prURL)
	}

	return branch, nil
}

func (manager *Manager) CreatePullRequest(ctx context.Context, cloneDir string, request PullRequest) (string, error) {
	baseBranch := strings.TrimSpace(request.BaseBranch)
	if baseBranch == "" {
		baseBranch = manager.Config.BaseBranch
	}
	var prOutput bytes.Buffer
	args := []string{
		"pr", "create",
		"--base", baseBranch,
		"--head", request.HeadBranch,
		"--title", request.Title,
		"--body", request.Body,
	}

	manager.logger(writerOrDiscard(manager.Stdout)).Infof("github", "%s %s", manager.Config.GHPath, strings.Join(args, " "))
	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name:   manager.Config.GHPath,
		Args:   args,
		Dir:    cloneDir,
		Stdout: &prOutput,
	}, "github", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
		return "", fmt.Errorf("github pr create: %w", err)
	}

	prURL, err := ParsePRURL(prOutput.String())
	if err != nil {
		return "", fmt.Errorf("github pr create: %w", err)
	}

	return prURL, nil
}

func (manager *Manager) CommentPullRequest(ctx context.Context, cloneDir string, prURL string, body string) error {
	logger := manager.logger(writerOrDiscard(manager.Stdout))
	if strings.TrimSpace(body) == "" {
		logger.Infof("github", "skipped PR comment: final agent response is empty")
		return nil
	}

	args := []string{"pr", "comment", prURL, "--body", body}
	logger.Infof("github", "publishing final agent response as PR comment")
	logger.Infof("github", "%s %s", manager.Config.GHPath, strings.Join(args, " "))

	if err := manager.runLoggedCommand(ctx, commandrunner.Command{
		Name: manager.Config.GHPath,
		Args: args,
		Dir:  cloneDir,
	}, "github", writerOrDiscard(manager.Stdout), writerOrDiscard(manager.Stderr)); err != nil {
		return fmt.Errorf("github final agent response comment: %w", err)
	}

	logger.Infof("github", "created PR comment from final agent response")
	return nil
}

func ParsePRURL(output string) (string, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return "", errors.New("empty PR URL output")
	}

	var payload struct {
		URL string `json:"url"`
	}
	if strings.HasPrefix(output, "{") {
		if err := json.Unmarshal([]byte(output), &payload); err == nil && payload.URL != "" {
			return payload.URL, nil
		}
	}

	if parsedURL, err := url.ParseRequestURI(output); err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
		return output, nil
	}

	urlPattern := regexp.MustCompile(`https?://[^\s"']+`)
	if match := urlPattern.FindString(output); match != "" {
		return match, nil
	}

	return "", errors.New("PR URL missing in gh output")
}

func (manager *Manager) runLoggedCommand(ctx context.Context, command commandrunner.Command, module string, stdout io.Writer, stderr io.Writer) error {
	stdoutLog := manager.logger(stdout).LineWriter(module)
	stderrLog := manager.logger(stderr).LineWriter(module)

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

	err := manager.commandRunner().Run(ctx, command)
	stdoutLog.Flush()
	stderrLog.Flush()

	return err
}

func (manager *Manager) commandRunner() commandrunner.Runner {
	if manager.Command != nil {
		return manager.Command
	}

	return commandrunner.ExecRunner{LogWriter: writerOrDiscard(manager.Stdout)}
}

func (manager *Manager) mkdirTemp(dir string, pattern string) (string, error) {
	if manager.MkdirTemp != nil {
		return manager.MkdirTemp(dir, pattern)
	}

	return os.MkdirTemp(dir, pattern)
}

func (manager *Manager) removeAll(path string) error {
	if manager.RemoveAll != nil {
		return manager.RemoveAll(path)
	}

	return os.RemoveAll(path)
}

func (manager *Manager) tempPattern() string {
	if strings.TrimSpace(manager.Config.TempPattern) == "" {
		return defaultTempPattern
	}

	return manager.Config.TempPattern
}

func (manager *Manager) logger(writer io.Writer) steplog.Logger {
	return steplog.NewWithService(writerOrDiscard(writer), manager.Service)
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
