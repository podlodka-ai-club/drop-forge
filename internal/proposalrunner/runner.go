package proposalrunner

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
	"time"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

const (
	defaultTempPattern = "orchv3-proposal-*"
	maxSlugLength      = 48
	lastMessageFile    = "codex-last-message.txt"
)

type Runner struct {
	Config  config.ProposalRunnerConfig
	Command commandrunner.Runner
	Service string
	Stdout  io.Writer
	Stderr  io.Writer

	MkdirTemp func(dir string, pattern string) (string, error)
	RemoveAll func(path string) error
	Now       func() time.Time
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

func (runner *Runner) Run(ctx context.Context, taskDescription string) (prURL string, err error) {
	taskDescription = strings.TrimSpace(taskDescription)
	if taskDescription == "" {
		return "", errors.New("proposal task description must not be empty")
	}

	if err := runner.Config.Validate(); err != nil {
		return "", fmt.Errorf("validate proposal runner config: %w", err)
	}

	command := runner.commandRunner()
	stdout := writerOrDiscard(runner.Stdout)
	stderr := writerOrDiscard(runner.Stderr)
	logger := runner.newLogger(stdout)
	defer func() {
		if err != nil {
			logger.Errorf("proposalrunner", "workflow failed: %v", err)
		}
	}()

	tempDir, err := runner.mkdirTemp("", defaultTempPattern)
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
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
	if err := runner.runLoggedCommand(ctx, command, commandrunner.Command{
		Name: runner.Config.GitPath,
		Args: []string{"clone", runner.Config.RepositoryURL, cloneDir},
		Dir:  tempDir,
	}, "git", stdout, stderr); err != nil {
		return "", fmt.Errorf("git clone: %w", err)
	}

	prompt := BuildCodexPrompt(taskDescription)
	lastMessagePath := filepath.Join(tempDir, lastMessageFile)
	logger.Infof("codex", "prompt:\n%s", prompt)
	if err := runner.runLoggedCommand(ctx, command, commandrunner.Command{
		Name:  runner.Config.CodexPath,
		Args:  CodexArgs(cloneDir, lastMessagePath),
		Dir:   cloneDir,
		Stdin: strings.NewReader(prompt),
	}, "codex", stdout, stderr); err != nil {
		return "", fmt.Errorf("codex proposal: %w", err)
	}

	lastMessage, err := ReadLastCodexMessage(lastMessagePath)
	if err != nil {
		return "", fmt.Errorf("read final codex message: %w", err)
	}

	status, err := runner.gitStatus(ctx, command, cloneDir, stdout, stderr)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(status) == "" {
		return "", errors.New("git status: no changes produced by codex")
	}
	logger.Infof("git", "status:\n%s", strings.TrimRight(status, "\n"))

	branchName := BuildBranchName(runner.Config.BranchPrefix, taskDescription, runner.now())
	prTitle := BuildPRTitle(runner.Config.PRTitlePrefix, taskDescription)
	prBody := BuildPRBody(taskDescription)

	gitCommands := []commandrunner.Command{
		{
			Name: runner.Config.GitPath,
			Args: []string{"checkout", "-b", branchName},
			Dir:  cloneDir,
		},
		{
			Name: runner.Config.GitPath,
			Args: []string{"add", "-A"},
			Dir:  cloneDir,
		},
		{
			Name: runner.Config.GitPath,
			Args: []string{"commit", "-m", prTitle},
			Dir:  cloneDir,
		},
		{
			Name: runner.Config.GitPath,
			Args: []string{"push", "-u", runner.Config.RemoteName, branchName},
			Dir:  cloneDir,
		},
	}

	for _, gitCommand := range gitCommands {
		logger.Infof("git", "%s", strings.Join(append([]string{gitCommand.Name}, gitCommand.Args...), " "))
		if err := runner.runLoggedCommand(ctx, command, gitCommand, "git", stdout, stderr); err != nil {
			return "", fmt.Errorf("git %s: %w", gitCommand.Args[0], err)
		}
	}

	prURL, err = runner.createPullRequest(ctx, command, cloneDir, branchName, prTitle, prBody, stdout, stderr)
	if err != nil {
		return "", err
	}
	logger.Infof("github", "created PR %s", prURL)

	if err := runner.commentLastCodexMessage(ctx, command, cloneDir, prURL, lastMessage, stdout, stderr); err != nil {
		return "", err
	}

	return prURL, nil
}

func BuildCodexPrompt(taskDescription string) string {
	return fmt.Sprintf(`Use the openspec-propose skill to create a complete OpenSpec proposal for the task below.

Task description:
%s
`, strings.TrimSpace(taskDescription))
}

func CodexArgs(cloneDir string, lastMessagePath string) []string {
	args := []string{"exec", "--json", "--sandbox", "danger-full-access"}
	if strings.TrimSpace(lastMessagePath) != "" {
		args = append(args, "--output-last-message", lastMessagePath)
	}
	args = append(args, "--cd", cloneDir, "-")

	return args
}

func BuildBranchName(prefix string, taskDescription string, now time.Time) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "codex/proposal"
	}

	timestamp := now.UTC().Format("20060102150405")
	return prefix + "/" + timestamp + "-" + slugify(taskDescription)
}

func BuildPRTitle(prefix string, taskDescription string) string {
	prefix = strings.TrimSpace(prefix)
	description := firstLine(taskDescription)
	descriptionRunes := []rune(description)
	if len(descriptionRunes) > 72 {
		description = strings.TrimSpace(string(descriptionRunes[:72]))
	}

	if prefix == "" {
		return description
	}

	return strings.TrimSpace(prefix + " " + description)
}

func BuildPRBody(taskDescription string) string {
	return fmt.Sprintf("Generated by orchv3 proposal runner.\n\nTask:\n%s\n", strings.TrimSpace(taskDescription))
}

func (runner *Runner) gitStatus(ctx context.Context, command commandrunner.Runner, cloneDir string, stdout io.Writer, stderr io.Writer) (string, error) {
	var status bytes.Buffer
	err := runner.runLoggedCommand(ctx, command, commandrunner.Command{
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

func (runner *Runner) createPullRequest(ctx context.Context, command commandrunner.Runner, cloneDir string, branchName string, title string, body string, stdout io.Writer, stderr io.Writer) (string, error) {
	var prOutput bytes.Buffer
	args := []string{
		"pr", "create",
		"--base", runner.Config.BaseBranch,
		"--head", branchName,
		"--title", title,
		"--body", body,
	}

	runner.newLogger(stdout).Infof("github", "%s %s", runner.Config.GHPath, strings.Join(args, " "))
	if err := runner.runLoggedCommand(ctx, command, commandrunner.Command{
		Name:   runner.Config.GHPath,
		Args:   args,
		Dir:    cloneDir,
		Stdout: &prOutput,
	}, "github", stdout, stderr); err != nil {
		return "", fmt.Errorf("github pr create: %w", err)
	}

	prURL, err := parsePRURL(prOutput.String())
	if err != nil {
		return "", fmt.Errorf("github pr create: %w", err)
	}

	return prURL, nil
}

func ReadLastCodexMessage(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func (runner *Runner) commentLastCodexMessage(ctx context.Context, command commandrunner.Runner, cloneDir string, prURL string, lastMessage string, stdout io.Writer, stderr io.Writer) error {
	logger := runner.newLogger(stdout)
	if strings.TrimSpace(lastMessage) == "" {
		logger.Infof("github", "skipped PR comment: final Codex response is empty")
		return nil
	}

	args := []string{"pr", "comment", prURL, "--body", lastMessage}
	logger.Infof("github", "publishing final Codex response as PR comment")
	logger.Infof("github", "%s %s", runner.Config.GHPath, strings.Join(args, " "))

	if err := runner.runLoggedCommand(ctx, command, commandrunner.Command{
		Name: runner.Config.GHPath,
		Args: args,
		Dir:  cloneDir,
	}, "github", stdout, stderr); err != nil {
		return fmt.Errorf("github final response comment: %w", err)
	}

	logger.Infof("github", "created PR comment from final Codex response")
	return nil
}

func parsePRURL(output string) (string, error) {
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

func (runner *Runner) commandRunner() commandrunner.Runner {
	if runner.Command != nil {
		return runner.Command
	}

	return commandrunner.ExecRunner{LogWriter: writerOrDiscard(runner.Stdout)}
}

func (runner *Runner) runLoggedCommand(ctx context.Context, exec commandrunner.Runner, command commandrunner.Command, module string, stdout io.Writer, stderr io.Writer) error {
	stdoutLog := runner.newLogger(stdout).LineWriter(module)
	stderrLog := runner.newLogger(stderr).LineWriter(module)

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

func firstLine(value string) string {
	for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}

	return "OpenSpec proposal"
}
