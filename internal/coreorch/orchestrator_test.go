package coreorch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"orchv3/internal/taskmanager"
)

func TestRunProposalsOnceProcessesOnlyReadyTasksSequentially(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{
			readyTask("issue-1", "ENG-1", "First"),
			{
				ID:         "issue-2",
				Identifier: "ENG-2",
				Title:      "Skip me",
				State:      taskmanager.WorkflowState{ID: "state-code"},
			},
			readyTask("issue-3", "ENG-3", "Second"),
		},
	}
	runner := &recordingProposalRunner{
		urls: []string{
			"https://github.com/example/repo/pull/1",
			"https://github.com/example/repo/pull/2",
		},
	}
	var logs bytes.Buffer
	orch := testOrchestrator(taskManager, runner, &logs)

	if err := orch.RunProposalsOnce(context.Background()); err != nil {
		t.Fatalf("RunProposalsOnce() returned error: %v", err)
	}

	if len(runner.inputs) != 2 {
		t.Fatalf("runner inputs len = %d, want 2", len(runner.inputs))
	}
	if !strings.Contains(runner.inputs[0], "Identifier: ENG-1") || !strings.Contains(runner.inputs[1], "Identifier: ENG-3") {
		t.Fatalf("runner inputs = %#v", runner.inputs)
	}
	if got := strings.Join(taskManager.addPRTaskIDs, ","); got != "issue-1,issue-3" {
		t.Fatalf("AddPR task order = %q", got)
	}
	if got := strings.Join(taskManager.moveTaskIDs, ","); got != "issue-1,issue-1,issue-3,issue-3" {
		t.Fatalf("MoveTask order = %q", got)
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-proposing-progress,state-proposal-review,state-proposing-progress,state-proposal-review" {
		t.Fatalf("MoveTask states = %q", got)
	}
	if got := strings.Join(taskManager.calls, ","); got != "move:issue-1:state-proposing-progress,add-pr:issue-1,move:issue-1:state-proposal-review,move:issue-3:state-proposing-progress,add-pr:issue-3,move:issue-3:state-proposal-review" {
		t.Fatalf("mutation calls = %q", got)
	}

	events := decodeEvents(t, logs.String())
	assertLogContains(t, events, "skip task=issue-2 identifier=ENG-2 state=state-code")
	assertLogContains(t, events, "processed proposal task=issue-3 identifier=ENG-3 pr=https://github.com/example/repo/pull/2")
}

func TestRunProposalsOnceWithNoReadyTasksDoesNotMutateTasks(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{{
			ID:         "issue-1",
			Identifier: "ENG-1",
			State:      taskmanager.WorkflowState{ID: "state-code"},
		}},
	}
	runner := &recordingProposalRunner{}
	var logs bytes.Buffer
	orch := testOrchestrator(taskManager, runner, &logs)

	if err := orch.RunProposalsOnce(context.Background()); err != nil {
		t.Fatalf("RunProposalsOnce() returned error: %v", err)
	}
	if len(runner.inputs) != 0 {
		t.Fatalf("runner inputs len = %d, want 0", len(runner.inputs))
	}
	if len(taskManager.addPRTaskIDs) != 0 || len(taskManager.moveTaskIDs) != 0 {
		t.Fatalf("mutations = addPR %#v move %#v, want none", taskManager.addPRTaskIDs, taskManager.moveTaskIDs)
	}

	events := decodeEvents(t, logs.String())
	assertLogContains(t, events, "no ready-to-propose tasks found")
}

func TestBuildProposalInputIncludesTaskPayloadDeterministically(t *testing.T) {
	task := readyTask("issue-1", "ENG-1", "Add proposal flow")
	task.Description = "Implement the proposal stage."
	task.Comments = []taskmanager.Comment{
		{
			Body:      "Keep it minimal.",
			CreatedAt: time.Date(2026, 4, 25, 9, 30, 0, 0, time.FixedZone("MSK", 3*60*60)),
			User:      taskmanager.User{DisplayName: "Alex"},
		},
		{
			Body: "Check CLI mode.",
			User: taskmanager.User{Name: "Dana"},
		},
	}

	got := BuildProposalInput(task)
	want := `Linear task:
ID: issue-1
Identifier: ENG-1
Title: Add proposal flow

Description:
Implement the proposal stage.

Comments:
1. Alex at 2026-04-25T06:30:00Z: Keep it minimal.
2. Dana: Check CLI mode.`
	if got != want {
		t.Fatalf("BuildProposalInput() =\n%s\nwant:\n%s", got, want)
	}
}

func TestBuildProposalInputHandlesMissingDescriptionAndComments(t *testing.T) {
	task := readyTask("issue-1", "ENG-1", "Add proposal flow")

	got := BuildProposalInput(task)
	if !strings.Contains(got, "No description provided.") {
		t.Fatalf("input missing no-description marker:\n%s", got)
	}
	if !strings.Contains(got, "No comments available.") {
		t.Fatalf("input missing no-comments marker:\n%s", got)
	}
}

func TestRunProposalsOnceRequiresProposingInProgressState(t *testing.T) {
	taskManager := &recordingTaskManager{}
	runner := &recordingProposalRunner{}
	orch := testOrchestrator(taskManager, runner, nil)
	orch.Config.ProposingInProgressStateID = " "

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "proposing-in-progress state id") {
		t.Fatalf("error = %q, want proposing-in-progress state context", err.Error())
	}
}

func TestRunProposalsOnceInProgressMoveFailureDoesNotRunProposal(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyTask("issue-1", "ENG-1", "First")},
		moveErrByStateID: map[string]error{
			"state-proposing-progress": errors.New("linear move failed"),
		},
	}
	runner := &recordingProposalRunner{}
	orch := testOrchestrator(taskManager, runner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "task=issue-1 identifier=ENG-1") || !strings.Contains(err.Error(), "move to proposing in-progress state state-proposing-progress") {
		t.Fatalf("error = %q, want task and in-progress move context", err.Error())
	}
	if len(runner.inputs) != 0 {
		t.Fatalf("runner inputs len = %d, want 0", len(runner.inputs))
	}
	if len(taskManager.addPRTaskIDs) != 0 {
		t.Fatalf("AddPR calls = %d, want 0", len(taskManager.addPRTaskIDs))
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-proposing-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceRunnerFailureLeavesTaskInProgress(t *testing.T) {
	taskManager := &recordingTaskManager{tasks: []taskmanager.Task{readyTask("issue-1", "ENG-1", "First")}}
	runner := &recordingProposalRunner{err: errors.New("codex failed")}
	orch := testOrchestrator(taskManager, runner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "task=issue-1 identifier=ENG-1") || !strings.Contains(err.Error(), "run proposal") {
		t.Fatalf("error = %q, want task and operation context", err.Error())
	}
	if len(taskManager.addPRTaskIDs) != 0 {
		t.Fatalf("AddPR calls = %d, want 0", len(taskManager.addPRTaskIDs))
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-proposing-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceAddPRFailureDoesNotMoveTaskToReview(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks:    []taskmanager.Task{readyTask("issue-1", "ENG-1", "First")},
		addPRErr: errors.New("linear attach failed"),
	}
	runner := &recordingProposalRunner{urls: []string{"https://github.com/example/repo/pull/1"}}
	orch := testOrchestrator(taskManager, runner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "attach proposal pr") {
		t.Fatalf("error = %q, want add PR context", err.Error())
	}
	if len(taskManager.addPRTaskIDs) != 1 {
		t.Fatalf("AddPR calls = %d, want 1", len(taskManager.addPRTaskIDs))
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-proposing-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceMoveTaskFailureReturnsPartialSuccessContext(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyTask("issue-1", "ENG-1", "First")},
		moveErrByStateID: map[string]error{
			"state-proposal-review": errors.New("linear move failed"),
		},
	}
	runner := &recordingProposalRunner{urls: []string{"https://github.com/example/repo/pull/1"}}
	orch := testOrchestrator(taskManager, runner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "move to proposal review state state-proposal-review after attaching pr https://github.com/example/repo/pull/1") {
		t.Fatalf("error = %q, want move context", err.Error())
	}
	if len(taskManager.addPRTaskIDs) != 1 || len(taskManager.moveTaskIDs) != 2 {
		t.Fatalf("mutations = addPR %#v move %#v", taskManager.addPRTaskIDs, taskManager.moveTaskIDs)
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-proposing-progress,state-proposal-review" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

type recordingTaskManager struct {
	tasks            []taskmanager.Task
	getTasksErr      error
	addPRErr         error
	moveErr          error
	moveErrByStateID map[string]error
	addPRTaskIDs     []string
	addPRURLs        []string
	moveTaskIDs      []string
	moveStateIDs     []string
	calls            []string
}

func (manager *recordingTaskManager) GetTasks(ctx context.Context) ([]taskmanager.Task, error) {
	return manager.tasks, manager.getTasksErr
}

func (manager *recordingTaskManager) AddPR(ctx context.Context, taskID string, prURL string) error {
	manager.addPRTaskIDs = append(manager.addPRTaskIDs, taskID)
	manager.addPRURLs = append(manager.addPRURLs, prURL)
	manager.calls = append(manager.calls, "add-pr:"+taskID)
	return manager.addPRErr
}

func (manager *recordingTaskManager) MoveTask(ctx context.Context, taskID string, stateID string) error {
	manager.moveTaskIDs = append(manager.moveTaskIDs, taskID)
	manager.moveStateIDs = append(manager.moveStateIDs, stateID)
	manager.calls = append(manager.calls, "move:"+taskID+":"+stateID)
	if manager.moveErrByStateID != nil {
		if err := manager.moveErrByStateID[stateID]; err != nil {
			return err
		}
	}
	return manager.moveErr
}

type recordingProposalRunner struct {
	inputs []string
	urls   []string
	err    error
}

func (runner *recordingProposalRunner) Run(ctx context.Context, taskDescription string) (string, error) {
	runner.inputs = append(runner.inputs, taskDescription)
	if runner.err != nil {
		return "", runner.err
	}
	if len(runner.urls) < len(runner.inputs) {
		return "https://github.com/example/repo/pull/default", nil
	}

	return runner.urls[len(runner.inputs)-1], nil
}

type logEvent struct {
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func decodeEvents(t *testing.T, output string) []logEvent {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	events := make([]logEvent, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event logEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode log event %q: %v", line, err)
		}
		events = append(events, event)
	}

	return events
}

func assertLogContains(t *testing.T, events []logEvent, message string) {
	t.Helper()

	for _, event := range events {
		if strings.Contains(event.Message, message) {
			return
		}
	}

	t.Fatalf("missing log message containing %q in %#v", message, events)
}

func testOrchestrator(taskManager TaskManager, runner ProposalRunner, logs *bytes.Buffer) Orchestrator {
	var logWriter ioWriter
	if logs != nil {
		logWriter = logs
	}

	return Orchestrator{
		Config: Config{
			ReadyToProposeStateID:      "state-propose",
			ProposingInProgressStateID: "state-proposing-progress",
			NeedProposalReviewStateID:  "state-proposal-review",
		},
		TaskManager:    taskManager,
		ProposalRunner: runner,
		LogWriter:      logWriter,
	}
}

func readyTask(id string, identifier string, title string) taskmanager.Task {
	return taskmanager.Task{
		ID:         id,
		Identifier: identifier,
		Title:      title,
		State:      taskmanager.WorkflowState{ID: "state-propose", Name: "Ready to Propose"},
	}
}

type ioWriter interface {
	Write(p []byte) (int, error)
}
