package coreorch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"orchv3/internal/agentmeta"
	"orchv3/internal/applyrunner"
	"orchv3/internal/archiverunner"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/reviewrunner"
	"orchv3/internal/steplog"
	"orchv3/internal/taskmanager"
)

const module = "coreorch"

type TaskManager interface {
	GetTasks(ctx context.Context) ([]taskmanager.Task, error)
	AddPR(ctx context.Context, taskID string, prURL string) error
	MoveTask(ctx context.Context, taskID string, stateID string) error
	MoveTaskWithContext(ctx context.Context, taskID string, stateID string, statusContext taskmanager.StatusChangeContext) error
}

type ProposalRunner interface {
	Run(ctx context.Context, input proposalrunner.ProposalInput) (string, error)
}

type ApplyRunner interface {
	Run(ctx context.Context, input applyrunner.ApplyInput) error
}

type ArchiveRunner interface {
	Run(ctx context.Context, input archiverunner.ArchiveInput) error
}

type ReviewRunner interface {
	Run(ctx context.Context, input reviewrunner.ReviewInput) (reviewrunner.Result, error)
}

type Config struct {
	ReadyToProposeStateID       string
	ProposingInProgressStateID  string
	NeedProposalReviewStateID   string
	NeedProposalAIReviewStateID string
	ReadyToCodeStateID          string
	CodeInProgressStateID       string
	NeedCodeReviewStateID       string
	NeedCodeAIReviewStateID     string
	ReadyToArchiveStateID       string
	ArchivingInProgressStateID  string
	NeedArchiveReviewStateID    string
	NeedArchiveAIReviewStateID  string
	AIReviewEnabled             bool
}

type Orchestrator struct {
	Config         Config
	TaskManager    TaskManager
	ProposalRunner ProposalRunner
	ApplyRunner    ApplyRunner
	ArchiveRunner  ArchiveRunner
	ReviewRunner   ReviewRunner
	Service        string
	LogWriter      io.Writer
}

type WaitFunc func(ctx context.Context, interval time.Duration) error

func (orch *Orchestrator) RunProposalsOnce(ctx context.Context) error {
	if err := orch.validate(); err != nil {
		return err
	}

	logger := steplog.NewWithService(writerOrDiscard(orch.LogWriter), orch.Service)
	tasks, err := orch.TaskManager.GetTasks(ctx)
	if err != nil {
		logger.Errorf(module, "load managed tasks: %v", err)
		return fmt.Errorf("load managed tasks: %w", err)
	}

	proposalCount := 0
	applyCount := 0
	archiveCount := 0
	var wg sync.WaitGroup
	var errsMu sync.Mutex
	var errs []error
	collectErr := func(err error) {
		if err == nil {
			return
		}
		errsMu.Lock()
		defer errsMu.Unlock()
		errs = append(errs, err)
	}

	for _, task := range tasks {
		task := task
		switch {
		case task.State.ID == orch.Config.ReadyToProposeStateID:
			proposalCount++
			wg.Add(1)
			go func() {
				defer wg.Done()
				collectErr(orch.processProposalTask(ctx, logger, task))
			}()
		case task.State.ID == orch.Config.ReadyToCodeStateID:
			applyCount++
			wg.Add(1)
			go func() {
				defer wg.Done()
				collectErr(orch.processApplyTask(ctx, logger, task))
			}()
		case task.State.ID == orch.Config.ReadyToArchiveStateID:
			archiveCount++
			wg.Add(1)
			go func() {
				defer wg.Done()
				collectErr(orch.processArchiveTask(ctx, logger, task))
			}()
		case orch.Config.NeedProposalAIReviewStateID != "" && task.State.ID == orch.Config.NeedProposalAIReviewStateID:
			wg.Add(1)
			go func() {
				defer wg.Done()
				collectErr(orch.routeReview(ctx, logger, task, agentmeta.StageProposal, orch.Config.NeedProposalReviewStateID, "Need Proposal Review"))
			}()
		case orch.Config.NeedCodeAIReviewStateID != "" && task.State.ID == orch.Config.NeedCodeAIReviewStateID:
			wg.Add(1)
			go func() {
				defer wg.Done()
				collectErr(orch.routeReview(ctx, logger, task, agentmeta.StageApply, orch.Config.NeedCodeReviewStateID, "Need Code Review"))
			}()
		case orch.Config.NeedArchiveAIReviewStateID != "" && task.State.ID == orch.Config.NeedArchiveAIReviewStateID:
			wg.Add(1)
			go func() {
				defer wg.Done()
				collectErr(orch.routeReview(ctx, logger, task, agentmeta.StageArchive, orch.Config.NeedArchiveReviewStateID, "Need Archive Review"))
			}()
		default:
			logger.Infof(
				module,
				"skip task=%s identifier=%s state=%s",
				task.ID,
				task.Identifier,
				task.State.ID,
			)
		}
	}

	if proposalCount == 0 {
		logger.Infof(module, "no ready-to-propose tasks found")
	}
	if applyCount == 0 {
		logger.Infof(module, "no ready-to-code tasks found")
	}
	if archiveCount == 0 {
		logger.Infof(module, "no ready-to-archive tasks found")
	}

	wg.Wait()
	return errors.Join(errs...)
}

func (orch *Orchestrator) RunProposalsLoop(ctx context.Context, interval time.Duration) error {
	return orch.runProposalsLoop(ctx, interval, waitInterval)
}

func (orch *Orchestrator) runProposalsLoop(ctx context.Context, interval time.Duration, wait WaitFunc) error {
	if interval <= 0 {
		return fmt.Errorf("proposal poll interval must be positive, got %s", interval)
	}
	if wait == nil {
		return fmt.Errorf("proposal poll wait func must not be nil")
	}
	if err := orch.validate(); err != nil {
		return err
	}

	logger := steplog.NewWithService(writerOrDiscard(orch.LogWriter), orch.Service)
	for iteration := 1; ; iteration++ {
		if err := ctx.Err(); err != nil {
			logger.Infof(module, "proposal monitor stopped: %v", err)
			return nil
		}

		logger.Infof(module, "orchestration monitor iteration start iteration=%d", iteration)
		if err := orch.RunProposalsOnce(ctx); err != nil {
			logger.Errorf(module, "orchestration monitor iteration error iteration=%d: %v", iteration, err)
		}

		if err := wait(ctx, interval); err != nil {
			if ctx.Err() != nil {
				logger.Infof(module, "proposal monitor stopped: %v", ctx.Err())
				return nil
			}
			return fmt.Errorf("wait orchestration poll interval: %w", err)
		}
	}
}

func (orch *Orchestrator) processProposalTask(ctx context.Context, logger steplog.Logger, task taskmanager.Task) error {
	taskRef := taskReference(task)
	logger.Infof(module, "process proposal task=%s identifier=%s", task.ID, task.Identifier)

	if err := orch.TaskManager.MoveTask(ctx, task.ID, orch.Config.ProposingInProgressStateID); err != nil {
		logger.Errorf(module, "move proposal task %s state=%s: %v", taskRef, orch.Config.ProposingInProgressStateID, err)
		return fmt.Errorf("process proposal %s: move to proposing in-progress state %s: %w", taskRef, orch.Config.ProposingInProgressStateID, err)
	}

	proposalInput := BuildProposalInput(task)
	prURL, err := orch.ProposalRunner.Run(ctx, proposalInput)
	if err != nil {
		logger.Errorf(module, "run proposal %s: %v", taskRef, err)
		return fmt.Errorf("process proposal %s: run proposal: %w", taskRef, err)
	}

	if err := orch.TaskManager.AddPR(ctx, task.ID, prURL); err != nil {
		logger.Errorf(module, "attach proposal pr %s pr=%s: %v", taskRef, prURL, err)
		return fmt.Errorf("process proposal %s: attach proposal pr %s: %w", taskRef, prURL, err)
	}

	target := orch.Config.NeedProposalReviewStateID
	targetName := "Need Proposal Review"
	if orch.Config.AIReviewEnabled {
		target = orch.Config.NeedProposalAIReviewStateID
		targetName = "Need Proposal AI Review"
	}
	statusContext := reviewStatusContext(task, targetName, prURL, "")
	if err := orch.TaskManager.MoveTaskWithContext(ctx, task.ID, target, statusContext); err != nil {
		logger.Errorf(module, "move proposal task %s pr=%s state=%s: %v", taskRef, prURL, target, err)
		return fmt.Errorf("process proposal %s: move to proposal review state %s after attaching pr %s: %w", taskRef, target, prURL, err)
	}

	logger.Infof(module, "processed proposal task=%s identifier=%s pr=%s", task.ID, task.Identifier, prURL)
	return nil
}

func (orch *Orchestrator) processApplyTask(ctx context.Context, logger steplog.Logger, task taskmanager.Task) error {
	taskRef := taskReference(task)
	logger.Infof(module, "process apply task=%s identifier=%s", task.ID, task.Identifier)

	applyInput, err := BuildApplyInput(task)
	if err != nil {
		logger.Errorf(module, "build apply input %s: %v", taskRef, err)
		return fmt.Errorf("process apply %s: build apply input: %w", taskRef, err)
	}

	if err := orch.TaskManager.MoveTask(ctx, task.ID, orch.Config.CodeInProgressStateID); err != nil {
		logger.Errorf(module, "move apply task %s state=%s: %v", taskRef, orch.Config.CodeInProgressStateID, err)
		return fmt.Errorf("process apply %s: move to code in-progress state %s: %w", taskRef, orch.Config.CodeInProgressStateID, err)
	}

	if err := orch.ApplyRunner.Run(ctx, applyInput); err != nil {
		logger.Errorf(module, "run apply %s: %v", taskRef, err)
		return fmt.Errorf("process apply %s: run apply: %w", taskRef, err)
	}

	target := orch.Config.NeedCodeReviewStateID
	targetName := "Need Code Review"
	if orch.Config.AIReviewEnabled {
		target = orch.Config.NeedCodeAIReviewStateID
		targetName = "Need Code AI Review"
	}
	statusContext := reviewStatusContext(task, targetName, applyInput.PRURL, applyInput.BranchName)
	if err := orch.TaskManager.MoveTaskWithContext(ctx, task.ID, target, statusContext); err != nil {
		logger.Errorf(module, "move apply task %s state=%s: %v", taskRef, target, err)
		return fmt.Errorf("process apply %s: move to code review state %s: %w", taskRef, target, err)
	}

	logger.Infof(module, "processed apply task=%s identifier=%s", task.ID, task.Identifier)
	return nil
}

func (orch *Orchestrator) processArchiveTask(ctx context.Context, logger steplog.Logger, task taskmanager.Task) error {
	taskRef := taskReference(task)
	logger.Infof(module, "process archive task=%s identifier=%s", task.ID, task.Identifier)

	archiveInput, err := BuildArchiveInput(task)
	if err != nil {
		logger.Errorf(module, "build archive input %s: %v", taskRef, err)
		return fmt.Errorf("process archive %s: build archive input: %w", taskRef, err)
	}

	if err := orch.TaskManager.MoveTask(ctx, task.ID, orch.Config.ArchivingInProgressStateID); err != nil {
		logger.Errorf(module, "move archive task %s state=%s: %v", taskRef, orch.Config.ArchivingInProgressStateID, err)
		return fmt.Errorf("process archive %s: move to archiving in-progress state %s: %w", taskRef, orch.Config.ArchivingInProgressStateID, err)
	}

	if err := orch.ArchiveRunner.Run(ctx, archiveInput); err != nil {
		logger.Errorf(module, "run archive %s: %v", taskRef, err)
		return fmt.Errorf("process archive %s: run archive: %w", taskRef, err)
	}

	target := orch.Config.NeedArchiveReviewStateID
	targetName := "Need Archive Review"
	if orch.Config.AIReviewEnabled {
		target = orch.Config.NeedArchiveAIReviewStateID
		targetName = "Need Archive AI Review"
	}
	statusContext := reviewStatusContext(task, targetName, archiveInput.PRURL, archiveInput.BranchName)
	if err := orch.TaskManager.MoveTaskWithContext(ctx, task.ID, target, statusContext); err != nil {
		logger.Errorf(module, "move archive task %s state=%s: %v", taskRef, target, err)
		return fmt.Errorf("process archive %s: move to archive review state %s: %w", taskRef, target, err)
	}

	logger.Infof(module, "processed archive task=%s identifier=%s", task.ID, task.Identifier)
	return nil
}

func BuildProposalInput(task taskmanager.Task) proposalrunner.ProposalInput {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = "Untitled task"
	}

	return proposalrunner.ProposalInput{
		Title:       title,
		Identifier:  strings.TrimSpace(task.Identifier),
		AgentPrompt: buildAgentPrompt(task),
	}
}

func BuildApplyInput(task taskmanager.Task) (applyrunner.ApplyInput, error) {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = "Untitled task"
	}

	prURL, branchName := branchSource(task.PullRequests)
	if branchName == "" && prURL == "" {
		return applyrunner.ApplyInput{}, fmt.Errorf("pull request branch source is missing")
	}

	return applyrunner.ApplyInput{
		TaskID:      strings.TrimSpace(task.ID),
		Identifier:  strings.TrimSpace(task.Identifier),
		Title:       title,
		AgentPrompt: buildAgentPrompt(task),
		PRURL:       prURL,
		BranchName:  branchName,
	}, nil
}

func BuildReviewInput(task taskmanager.Task, stage agentmeta.Stage) (reviewrunner.ReviewInput, error) {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = "Untitled task"
	}
	prURL, branchName := branchSource(task.PullRequests)
	if branchName == "" && prURL == "" {
		return reviewrunner.ReviewInput{}, fmt.Errorf("pull request branch source is missing")
	}
	if prURL == "" {
		return reviewrunner.ReviewInput{}, fmt.Errorf("review input requires PR URL to derive owner/repo/number")
	}
	owner, repo, number, err := parseGitHubPR(prURL)
	if err != nil {
		return reviewrunner.ReviewInput{}, fmt.Errorf("parse PR URL %q: %w", prURL, err)
	}
	return reviewrunner.ReviewInput{
		Stage:      stage,
		Identifier: strings.TrimSpace(task.Identifier),
		Title:      title,
		BranchName: branchName,
		PRNumber:   number,
		RepoOwner:  owner,
		RepoName:   repo,
		PRURL:      prURL,
	}, nil
}

func parseGitHubPR(prURL string) (string, string, int, error) {
	u, err := url.Parse(prURL)
	if err != nil {
		return "", "", 0, err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "pull" {
		return "", "", 0, fmt.Errorf("unexpected PR URL path: %s", u.Path)
	}
	num, err := strconv.Atoi(parts[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("parse PR number from %q: %w", parts[3], err)
	}
	return parts[0], parts[1], num, nil
}

func (orch *Orchestrator) processReviewTask(ctx context.Context, logger steplog.Logger, task taskmanager.Task, stage agentmeta.Stage, targetState string, targetStateName string) error {
	taskRef := taskReference(task)
	logger.Infof(module, "process %s ai-review task=%s identifier=%s", stage, task.ID, task.Identifier)
	in, err := BuildReviewInput(task, stage)
	if err != nil {
		logger.Errorf(module, "build review input %s: %v", taskRef, err)
		return fmt.Errorf("process %s ai-review %s: build input: %w", stage, taskRef, err)
	}
	if _, err := orch.ReviewRunner.Run(ctx, in); err != nil {
		logger.Errorf(module, "run %s ai-review %s: %v", stage, taskRef, err)
		return fmt.Errorf("process %s ai-review %s: run review: %w", stage, taskRef, err)
	}
	statusContext := reviewStatusContext(task, targetStateName, in.PRURL, in.BranchName)
	if err := orch.TaskManager.MoveTaskWithContext(ctx, task.ID, targetState, statusContext); err != nil {
		logger.Errorf(module, "move %s review task %s state=%s: %v", stage, taskRef, targetState, err)
		return fmt.Errorf("process %s ai-review %s: move to human review state %s: %w", stage, taskRef, targetState, err)
	}
	logger.Infof(module, "processed %s ai-review task=%s identifier=%s", stage, task.ID, task.Identifier)
	return nil
}

func (orch *Orchestrator) routeReview(ctx context.Context, logger steplog.Logger, task taskmanager.Task, stage agentmeta.Stage, targetState string, targetStateName string) error {
	if !orch.Config.AIReviewEnabled {
		logger.Infof(module, "skip ai-review task=%s identifier=%s state=%s reason=feature_disabled", task.ID, task.Identifier, task.State.ID)
		return nil
	}
	if orch.ReviewRunner == nil {
		return fmt.Errorf("ai-review enabled but ReviewRunner is nil")
	}
	return orch.processReviewTask(ctx, logger, task, stage, targetState, targetStateName)
}

func BuildArchiveInput(task taskmanager.Task) (archiverunner.ArchiveInput, error) {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = "Untitled task"
	}

	prURL, branchName := branchSource(task.PullRequests)
	if branchName == "" && prURL == "" {
		return archiverunner.ArchiveInput{}, fmt.Errorf("pull request branch source is missing")
	}

	return archiverunner.ArchiveInput{
		TaskID:      strings.TrimSpace(task.ID),
		Identifier:  strings.TrimSpace(task.Identifier),
		Title:       title,
		AgentPrompt: buildAgentPrompt(task),
		PRURL:       prURL,
		BranchName:  branchName,
	}, nil
}

func branchSource(pullRequests []taskmanager.PullRequest) (string, string) {
	var prURL string
	var branchName string
	for _, pullRequest := range pullRequests {
		if branchName == "" {
			branchName = strings.TrimSpace(pullRequest.Branch)
		}
		if prURL == "" {
			prURL = strings.TrimSpace(pullRequest.URL)
		}
		if branchName != "" && prURL != "" {
			break
		}
	}

	return prURL, branchName
}

func reviewStatusContext(task taskmanager.Task, targetStateName string, prURL string, branchName string) taskmanager.StatusChangeContext {
	return taskmanager.StatusChangeContext{
		TaskIdentifier:    task.Identifier,
		TaskTitle:         task.Title,
		SourceStateID:     task.State.ID,
		SourceStateName:   task.State.Name,
		TargetStateName:   targetStateName,
		PullRequestURL:    prURL,
		PullRequestBranch: branchName,
	}
}

func buildAgentPrompt(task taskmanager.Task) string {
	var builder strings.Builder

	builder.WriteString("Linear task:\n")
	writeField(&builder, "ID", task.ID)
	writeField(&builder, "Identifier", task.Identifier)
	writeField(&builder, "Title", task.Title)

	builder.WriteString("\nDescription:\n")
	description := strings.TrimSpace(task.Description)
	if description == "" {
		description = "No description provided."
	}
	builder.WriteString(description)
	builder.WriteString("\n")

	builder.WriteString("\nComments:\n")
	if len(task.Comments) == 0 {
		builder.WriteString("No comments available.\n")
		return strings.TrimSpace(builder.String())
	}

	for index, comment := range task.Comments {
		author := strings.TrimSpace(comment.User.DisplayName)
		if author == "" {
			author = strings.TrimSpace(comment.User.Name)
		}
		if author == "" {
			author = "Unknown author"
		}

		body := strings.TrimSpace(comment.Body)
		if body == "" {
			body = "(empty comment)"
		}

		if comment.CreatedAt.IsZero() {
			fmt.Fprintf(&builder, "%d. %s: %s\n", index+1, author, body)
			continue
		}

		fmt.Fprintf(&builder, "%d. %s at %s: %s\n", index+1, author, comment.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"), body)
	}

	return strings.TrimSpace(builder.String())
}

func (orch *Orchestrator) validate() error {
	if orch.TaskManager == nil {
		return fmt.Errorf("task manager must not be nil")
	}
	if orch.ProposalRunner == nil {
		return fmt.Errorf("proposal runner must not be nil")
	}
	if orch.ApplyRunner == nil {
		return fmt.Errorf("apply runner must not be nil")
	}
	if orch.ArchiveRunner == nil {
		return fmt.Errorf("archive runner must not be nil")
	}
	if orch.Config.AIReviewEnabled && orch.ReviewRunner == nil {
		return fmt.Errorf("ai review enabled but review runner is nil")
	}
	if orch.Config.AIReviewEnabled {
		if strings.TrimSpace(orch.Config.NeedProposalAIReviewStateID) == "" {
			return fmt.Errorf("need-proposal-ai-review state id must not be empty when ai review enabled")
		}
		if strings.TrimSpace(orch.Config.NeedCodeAIReviewStateID) == "" {
			return fmt.Errorf("need-code-ai-review state id must not be empty when ai review enabled")
		}
		if strings.TrimSpace(orch.Config.NeedArchiveAIReviewStateID) == "" {
			return fmt.Errorf("need-archive-ai-review state id must not be empty when ai review enabled")
		}
	}
	if strings.TrimSpace(orch.Config.ReadyToProposeStateID) == "" {
		return fmt.Errorf("ready-to-propose state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.ProposingInProgressStateID) == "" {
		return fmt.Errorf("proposing-in-progress state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.NeedProposalReviewStateID) == "" {
		return fmt.Errorf("need-proposal-review state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.ReadyToCodeStateID) == "" {
		return fmt.Errorf("ready-to-code state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.CodeInProgressStateID) == "" {
		return fmt.Errorf("code-in-progress state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.NeedCodeReviewStateID) == "" {
		return fmt.Errorf("need-code-review state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.ReadyToArchiveStateID) == "" {
		return fmt.Errorf("ready-to-archive state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.ArchivingInProgressStateID) == "" {
		return fmt.Errorf("archiving-in-progress state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.NeedArchiveReviewStateID) == "" {
		return fmt.Errorf("need-archive-review state id must not be empty")
	}

	return nil
}

func writeField(builder *strings.Builder, name string, value string) {
	fmt.Fprintf(builder, "%s: %s\n", name, strings.TrimSpace(value))
}

func taskReference(task taskmanager.Task) string {
	identifier := strings.TrimSpace(task.Identifier)
	if identifier == "" {
		return fmt.Sprintf("task=%s", task.ID)
	}

	return fmt.Sprintf("task=%s identifier=%s", task.ID, identifier)
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return &lockedWriter{writer: writer}
}

type lockedWriter struct {
	mu     sync.Mutex
	writer io.Writer
}

func (writer *lockedWriter) Write(p []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.writer.Write(p)
}

func waitInterval(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
