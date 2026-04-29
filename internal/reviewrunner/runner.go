package reviewrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/reviewrunner/prcommenter"
	"orchv3/internal/reviewrunner/reviewparse"
	"orchv3/internal/steplog"
)

const defaultTempPattern = "orchv3-review-*"

// Runner orchestrates one cross-agent PR review: clone, prompt, parse, publish.
type Runner struct {
	Config      config.ReviewRunnerConfig
	ProposalCfg config.ProposalRunnerConfig
	Command     commandrunner.Runner
	Executors   map[string]AgentExecutor
	Commenter   prcommenter.PRCommenter
	Service     string
	Stdout      io.Writer
	Stderr      io.Writer
	MkdirTemp   func(dir, pattern string) (string, error)
	RemoveAll   func(path string) error
}

// ReviewInput is the per-PR input handed to (*Runner).Run.
type ReviewInput struct {
	Stage      agentmeta.Stage
	Identifier string
	Title      string
	BranchName string
	PRNumber   int
	RepoOwner  string
	RepoName   string
	PRURL      string
}

// Result reports the outcome of a single review run.
type Result struct {
	Skipped bool
}

func (in ReviewInput) validate() error {
	if in.Stage == "" {
		return errors.New("review input stage must not be empty")
	}
	if strings.TrimSpace(in.BranchName) == "" && strings.TrimSpace(in.PRURL) == "" {
		return errors.New("review input branch source must include branch name or PR URL")
	}
	if in.PRNumber <= 0 {
		return errors.New("review input PR number must be positive")
	}
	if strings.TrimSpace(in.RepoOwner) == "" || strings.TrimSpace(in.RepoName) == "" {
		return errors.New("review input repo owner and name must not be empty")
	}
	return nil
}

// Run executes the review workflow end-to-end for a single PR.
func (r *Runner) Run(ctx context.Context, in ReviewInput) (result Result, err error) {
	if err := in.validate(); err != nil {
		return Result{}, err
	}
	if r.Command == nil {
		return Result{}, errors.New("command runner must not be nil")
	}
	if r.Commenter == nil {
		return Result{}, errors.New("commenter must not be nil")
	}
	if r.Executors == nil {
		return Result{}, errors.New("executors map must not be nil")
	}

	stdout := writerOrDiscard(r.Stdout)
	stderr := writerOrDiscard(r.Stderr)
	logger := steplog.NewWithService(stdout, r.Service)
	logger.Infof("review", "start stage=%s pr=%d branch=%s", in.Stage, in.PRNumber, in.BranchName)
	defer func() {
		if err != nil {
			logger.Errorf("review", "workflow failed: %v", err)
		}
	}()

	branch, err := r.resolveBranch(ctx, in, stdout, stderr)
	if err != nil {
		return Result{}, err
	}
	in.BranchName = branch

	tempDir, err := r.mkdirTemp("", defaultTempPattern)
	if err != nil {
		return Result{}, fmt.Errorf("create temp dir: %w", err)
	}
	logger.Infof("temp", "created %s", tempDir)
	defer func() {
		if !r.ProposalCfg.CleanupTemp {
			logger.Infof("temp", "preserving %s", tempDir)
			return
		}
		if rmErr := r.removeAll(tempDir); rmErr != nil {
			logger.Errorf("temp", "cleanup failed for %s: %v", tempDir, rmErr)
			if err == nil {
				err = fmt.Errorf("cleanup temp dir %s: %w", tempDir, rmErr)
			}
			return
		}
		logger.Infof("temp", "removed %s", tempDir)
	}()

	cloneDir := filepath.Join(tempDir, "repo")
	if err := r.gitClone(ctx, in.BranchName, cloneDir, stdout, stderr); err != nil {
		return Result{}, fmt.Errorf("git clone: %w", err)
	}

	headSHA, message, err := r.readHead(ctx, cloneDir, stdout, stderr)
	if err != nil {
		return Result{}, err
	}

	var producer agentmeta.Producer
	parsed, parseErr := agentmeta.ParseTrailer(message)
	switch {
	case parseErr == nil:
		producer = parsed
	case errors.Is(parseErr, agentmeta.ErrTrailerNotFound):
		logger.Infof("review", "producer trailer absent on HEAD %s", headSHA)
	default:
		return Result{}, fmt.Errorf("parse producer trailer: %w", parseErr)
	}

	reviewer, err := SelectReviewer(r.Config, producer)
	if err != nil {
		return Result{}, fmt.Errorf("select reviewer: %w", err)
	}
	executor, ok := r.Executors[reviewer.Slot]
	if !ok {
		return Result{}, fmt.Errorf("no agent executor registered for slot %q", reviewer.Slot)
	}

	changePath, _ := r.detectChangePath(ctx, cloneDir, stdout, stderr)
	diff, _ := r.gitDiff(ctx, cloneDir, stdout, stderr)

	targets, err := CollectTargets(TargetInput{
		Stage:      in.Stage,
		CloneDir:   cloneDir,
		MaxBytes:   r.Config.MaxContextBytes,
		ChangePath: changePath,
		Diff:       diff,
	})
	if err != nil {
		return Result{}, fmt.Errorf("collect targets: %w", err)
	}

	profile, err := ProfileFor(in.Stage)
	if err != nil {
		return Result{}, err
	}
	categories := categorySlice(profile.Categories)

	prompt, err := RenderPrompt(PromptInput{
		Stage:         in.Stage,
		ProducerBy:    producer.By,
		ProducerModel: producer.Model,
		ReviewerBy:    reviewer.Slot,
		ReviewerModel: reviewer.Model,
		Categories:    categories,
		Targets:       targets,
	}, r.Config.PromptDir)
	if err != nil {
		return Result{}, fmt.Errorf("render prompt: %w", err)
	}

	review, err := r.executeWithRepair(ctx, executor, prompt, in.Stage, cloneDir, tempDir, stdout, stderr)
	if err != nil {
		return Result{}, err
	}

	extras := []string{}
	if reviewer.ProducerUnknown {
		extras = append(extras, "Producer trailer absent — reviewer chosen by REVIEW_ROLE_SECONDARY.")
	}
	for _, t := range targets {
		if t.Truncated {
			extras = append(extras, fmt.Sprintf("Target truncated: %s", t.Path))
		}
	}

	res, err := r.Commenter.PostReview(ctx, prcommenter.PostReviewInput{
		RepoOwner:         in.RepoOwner,
		RepoName:          in.RepoName,
		PRNumber:          in.PRNumber,
		HeadSHA:           headSHA,
		Stage:             in.Stage,
		ReviewerSlot:      reviewer.Slot,
		Review:            review,
		WalkthroughExtras: extras,
	})
	if err != nil {
		return Result{}, fmt.Errorf("post review: %w", err)
	}
	if res.Skipped {
		logger.Infof("review", "skipped_idempotent stage=%s sha=%s", in.Stage, headSHA)
	} else {
		logger.Infof("review", "publish ok stage=%s findings=%d", in.Stage, len(review.Findings))
	}
	return Result{Skipped: res.Skipped}, nil
}

// executeWithRepair runs the executor once; if the response fails to parse,
// retries once with a repair prompt that quotes the parse error verbatim.
func (r *Runner) executeWithRepair(ctx context.Context, exec AgentExecutor, prompt string, stage agentmeta.Stage, cloneDir, tempDir string, stdout, stderr io.Writer) (reviewparse.Review, error) {
	result, err := exec.Run(ctx, AgentExecutionInput{
		Prompt:   prompt,
		CloneDir: cloneDir,
		TempDir:  tempDir,
		Stdout:   stdout,
		Stderr:   stderr,
	})
	if err != nil {
		return reviewparse.Review{}, fmt.Errorf("agent review: %w", err)
	}
	review, parseErr := reviewparse.Parse([]byte(result.FinalMessage), stage)
	if parseErr == nil {
		return review, nil
	}
	if r.Config.ParseRepairRetries < 1 {
		return reviewparse.Review{}, fmt.Errorf("parse review JSON: %w", parseErr)
	}

	repairPrompt := fmt.Sprintf(
		"Твой предыдущий ответ невалиден. Ошибка: %v.\nВерни строго JSON по схеме без лишнего текста.\n\n--- предыдущий ответ ---\n%s",
		parseErr, result.FinalMessage,
	)
	result2, err := exec.Run(ctx, AgentExecutionInput{
		Prompt:   repairPrompt,
		CloneDir: cloneDir,
		TempDir:  tempDir,
		Stdout:   stdout,
		Stderr:   stderr,
	})
	if err != nil {
		return reviewparse.Review{}, fmt.Errorf("agent review repair: %w", err)
	}
	review, parseErr = reviewparse.Parse([]byte(result2.FinalMessage), stage)
	if parseErr != nil {
		return reviewparse.Review{}, fmt.Errorf("parse review JSON after repair: %w", parseErr)
	}
	return review, nil
}

// resolveBranch returns input.BranchName when set, otherwise asks `gh` for the
// PR's head ref. Mirrors the pattern in apply/archive runners but executes
// before clone so `git clone --branch` has a name to use.
func (r *Runner) resolveBranch(ctx context.Context, in ReviewInput, stdout, stderr io.Writer) (string, error) {
	if branch := strings.TrimSpace(in.BranchName); branch != "" {
		return branch, nil
	}
	prURL := strings.TrimSpace(in.PRURL)
	if prURL == "" {
		return "", errors.New("review input has neither branch name nor PR URL")
	}
	var branchOutput bytes.Buffer
	args := []string{"pr", "view", prURL, "--json", "headRefName", "--jq", ".headRefName"}
	if err := r.Command.Run(ctx, commandrunner.Command{
		Name:   r.ProposalCfg.GHPath,
		Args:   args,
		Stdout: &branchOutput,
		Stderr: stderr,
	}); err != nil {
		return "", fmt.Errorf("github resolve pr branch %s: %w", prURL, err)
	}
	branch := strings.TrimSpace(branchOutput.String())
	if branch == "" {
		return "", fmt.Errorf("github resolve pr branch %s: empty headRefName", prURL)
	}
	return branch, nil
}

func (r *Runner) gitClone(ctx context.Context, branch, cloneDir string, stdout, stderr io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(cloneDir), 0o755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	return r.Command.Run(ctx, commandrunner.Command{
		Name:   r.ProposalCfg.GitPath,
		Args:   []string{"clone", "--branch", branch, r.ProposalCfg.RepositoryURL, cloneDir},
		Dir:    filepath.Dir(cloneDir),
		Stdout: stdout,
		Stderr: stderr,
	})
}

func (r *Runner) readHead(ctx context.Context, cloneDir string, stdout, stderr io.Writer) (string, string, error) {
	var shaOut bytes.Buffer
	if err := r.Command.Run(ctx, commandrunner.Command{
		Name:   r.ProposalCfg.GitPath,
		Args:   []string{"rev-parse", "HEAD"},
		Dir:    cloneDir,
		Stdout: &shaOut,
		Stderr: stderr,
	}); err != nil {
		return "", "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	var msgOut bytes.Buffer
	if err := r.Command.Run(ctx, commandrunner.Command{
		Name:   r.ProposalCfg.GitPath,
		Args:   []string{"log", "-1", "--format=%B"},
		Dir:    cloneDir,
		Stdout: &msgOut,
		Stderr: stderr,
	}); err != nil {
		return "", "", fmt.Errorf("git log -1: %w", err)
	}
	return strings.TrimSpace(shaOut.String()), msgOut.String(), nil
}

// baseRef returns the remote-tracking ref of the configured base branch.
// `git clone --branch <X>` fetches all remote refs but only creates a local
// branch for X, so `main` may not resolve locally. Use `origin/main` instead.
func (r *Runner) baseRef() string {
	base := r.ProposalCfg.BaseBranch
	if base == "" {
		base = "main"
	}
	remote := r.ProposalCfg.RemoteName
	if remote == "" {
		remote = "origin"
	}
	return remote + "/" + base
}

func (r *Runner) gitDiff(ctx context.Context, cloneDir string, stdout, stderr io.Writer) (string, error) {
	var out bytes.Buffer
	err := r.Command.Run(ctx, commandrunner.Command{
		Name:   r.ProposalCfg.GitPath,
		Args:   []string{"diff", r.baseRef() + "...HEAD"},
		Dir:    cloneDir,
		Stdout: &out,
		Stderr: stderr,
	})
	return out.String(), err
}

func (r *Runner) detectChangePath(ctx context.Context, cloneDir string, stdout, stderr io.Writer) (string, error) {
	var out bytes.Buffer
	err := r.Command.Run(ctx, commandrunner.Command{
		Name:   r.ProposalCfg.GitPath,
		Args:   []string{"diff", "--name-only", r.baseRef() + "...HEAD"},
		Dir:    cloneDir,
		Stdout: &out,
		Stderr: stderr,
	})
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "openspec/changes/") {
			continue
		}
		parts := strings.SplitN(line, "/", 4)
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/"), nil
		}
	}
	return "", nil
}

func (r *Runner) mkdirTemp(dir, pattern string) (string, error) {
	if r.MkdirTemp != nil {
		return r.MkdirTemp(dir, pattern)
	}
	return os.MkdirTemp(dir, pattern)
}

func (r *Runner) removeAll(path string) error {
	if r.RemoveAll != nil {
		return r.RemoveAll(path)
	}
	return os.RemoveAll(path)
}

func categorySlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}
