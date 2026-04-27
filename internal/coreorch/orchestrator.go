package coreorch

import (
	"context"
	"fmt"
	"io"
	"strings"

	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
	"orchv3/internal/taskmanager"
)

const module = "coreorch"

type TaskManager interface {
	GetTasks(ctx context.Context) ([]taskmanager.Task, error)
	AddPR(ctx context.Context, taskID string, prURL string) error
	MoveTask(ctx context.Context, taskID string, stateID string) error
}

type ProposalRunner interface {
	Run(ctx context.Context, input proposalrunner.ProposalInput) (string, error)
}

type Config struct {
	ReadyToProposeStateID      string
	ProposingInProgressStateID string
	NeedProposalReviewStateID  string
}

type Orchestrator struct {
	Config         Config
	TaskManager    TaskManager
	ProposalRunner ProposalRunner
	Service        string
	LogWriter      io.Writer
}

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

	readyCount := 0
	for _, task := range tasks {
		if task.State.ID != orch.Config.ReadyToProposeStateID {
			logger.Infof(
				module,
				"skip task=%s identifier=%s state=%s",
				task.ID,
				task.Identifier,
				task.State.ID,
			)
			continue
		}

		readyCount++
		if err := orch.processTask(ctx, logger, task); err != nil {
			return err
		}
	}

	if readyCount == 0 {
		logger.Infof(module, "no ready-to-propose tasks found")
	}

	return nil
}

func (orch *Orchestrator) processTask(ctx context.Context, logger steplog.Logger, task taskmanager.Task) error {
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

	if err := orch.TaskManager.MoveTask(ctx, task.ID, orch.Config.NeedProposalReviewStateID); err != nil {
		logger.Errorf(module, "move proposal task %s pr=%s state=%s: %v", taskRef, prURL, orch.Config.NeedProposalReviewStateID, err)
		return fmt.Errorf("process proposal %s: move to proposal review state %s after attaching pr %s: %w", taskRef, orch.Config.NeedProposalReviewStateID, prURL, err)
	}

	logger.Infof(module, "processed proposal task=%s identifier=%s pr=%s", task.ID, task.Identifier, prURL)
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
	if strings.TrimSpace(orch.Config.ReadyToProposeStateID) == "" {
		return fmt.Errorf("ready-to-propose state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.ProposingInProgressStateID) == "" {
		return fmt.Errorf("proposing-in-progress state id must not be empty")
	}
	if strings.TrimSpace(orch.Config.NeedProposalReviewStateID) == "" {
		return fmt.Errorf("need-proposal-review state id must not be empty")
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

	return writer
}
