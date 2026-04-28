package coreorch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"orchv3/internal/applyrunner"
	"orchv3/internal/archiverunner"
	"orchv3/internal/proposalrunner"
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
				State:      taskmanager.WorkflowState{ID: "state-archive"},
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
	if runner.inputs[0].Identifier != "ENG-1" || runner.inputs[0].Title != "First" {
		t.Fatalf("runner inputs[0] = %#v", runner.inputs[0])
	}
	if runner.inputs[1].Identifier != "ENG-3" || runner.inputs[1].Title != "Second" {
		t.Fatalf("runner inputs[1] = %#v", runner.inputs[1])
	}
	if !strings.Contains(runner.inputs[0].AgentPrompt, "Identifier: ENG-1") || !strings.Contains(runner.inputs[1].AgentPrompt, "Identifier: ENG-3") {
		t.Fatalf("runner agent prompts = %#v", runner.inputs)
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
	assertLogContains(t, events, "skip task=issue-2 identifier=ENG-2 state=state-archive")
	assertLogContains(t, events, "processed proposal task=issue-3 identifier=ENG-3 pr=https://github.com/example/repo/pull/2")
}

func TestRunProposalsOnceAttachesGitLabMRURLWithoutInspectingIt(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyTask("issue-1", "ENG-1", "GitLab proposal")},
	}
	runner := &recordingProposalRunner{
		urls: []string{"https://gitlab.com/example/repo/-/merge_requests/7"},
	}
	orch := testOrchestrator(taskManager, runner, nil)

	if err := orch.RunProposalsOnce(context.Background()); err != nil {
		t.Fatalf("RunProposalsOnce() returned error: %v", err)
	}

	if got := strings.Join(taskManager.addPRURLs, ","); got != "https://gitlab.com/example/repo/-/merge_requests/7" {
		t.Fatalf("AddPR URLs = %q", got)
	}
}

func TestRunProposalsOnceWithNoReadyTasksDoesNotMutateTasks(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{{
			ID:         "issue-1",
			Identifier: "ENG-1",
			State:      taskmanager.WorkflowState{ID: "state-archive"},
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

func TestRunProposalsOnceRoutesProposalAndApplyTasksSequentially(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{
			readyTask("issue-1", "ENG-1", "Proposal"),
			readyCodeTask("issue-2", "ENG-2", "Apply", taskmanager.PullRequest{URL: "https://github.com/example/repo/pull/2"}),
		},
	}
	proposalRunner := &recordingProposalRunner{urls: []string{"https://github.com/example/repo/pull/1"}}
	applyRunner := &recordingApplyRunner{}
	orch := testOrchestratorWithApply(taskManager, proposalRunner, applyRunner, nil)

	if err := orch.RunProposalsOnce(context.Background()); err != nil {
		t.Fatalf("RunProposalsOnce() returned error: %v", err)
	}

	if len(proposalRunner.inputs) != 1 {
		t.Fatalf("proposal inputs len = %d, want 1", len(proposalRunner.inputs))
	}
	if len(applyRunner.inputs) != 1 {
		t.Fatalf("apply inputs len = %d, want 1", len(applyRunner.inputs))
	}
	if applyRunner.inputs[0].PRURL != "https://github.com/example/repo/pull/2" || applyRunner.inputs[0].Identifier != "ENG-2" {
		t.Fatalf("apply input = %#v", applyRunner.inputs[0])
	}
	if got := strings.Join(taskManager.calls, ","); got != "move:issue-1:state-proposing-progress,add-pr:issue-1,move:issue-1:state-proposal-review,move:issue-2:state-code-progress,move:issue-2:state-code-review" {
		t.Fatalf("mutation calls = %q", got)
	}
}

func TestRunProposalsOnceRoutesProposalApplyAndArchiveTasksSequentially(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{
			readyTask("issue-1", "ENG-1", "Proposal"),
			readyCodeTask("issue-2", "ENG-2", "Apply", taskmanager.PullRequest{URL: "https://github.com/example/repo/pull/2"}),
			readyArchiveTask("issue-3", "ENG-3", "Archive", taskmanager.PullRequest{Branch: "codex/proposal/archive"}),
		},
	}
	proposalRunner := &recordingProposalRunner{urls: []string{"https://github.com/example/repo/pull/1"}}
	applyRunner := &recordingApplyRunner{}
	archiveRunner := &recordingArchiveRunner{}
	orch := testOrchestratorWithArchive(taskManager, proposalRunner, applyRunner, archiveRunner, nil)

	if err := orch.RunProposalsOnce(context.Background()); err != nil {
		t.Fatalf("RunProposalsOnce() returned error: %v", err)
	}

	if len(proposalRunner.inputs) != 1 {
		t.Fatalf("proposal inputs len = %d, want 1", len(proposalRunner.inputs))
	}
	if len(applyRunner.inputs) != 1 {
		t.Fatalf("apply inputs len = %d, want 1", len(applyRunner.inputs))
	}
	if len(archiveRunner.inputs) != 1 {
		t.Fatalf("archive inputs len = %d, want 1", len(archiveRunner.inputs))
	}
	if archiveRunner.inputs[0].BranchName != "codex/proposal/archive" || archiveRunner.inputs[0].Identifier != "ENG-3" {
		t.Fatalf("archive input = %#v", archiveRunner.inputs[0])
	}
	if got := strings.Join(taskManager.calls, ","); got != "move:issue-1:state-proposing-progress,add-pr:issue-1,move:issue-1:state-proposal-review,move:issue-2:state-code-progress,move:issue-2:state-code-review,move:issue-3:state-archiving-progress,move:issue-3:state-archive-review" {
		t.Fatalf("mutation calls = %q", got)
	}
}

func TestRunProposalsOnceRejectsApplyTaskWithoutBranchSource(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyCodeTask("issue-1", "ENG-1", "Apply", taskmanager.PullRequest{})},
	}
	applyRunner := &recordingApplyRunner{}
	orch := testOrchestratorWithApply(taskManager, &recordingProposalRunner{}, applyRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "pull request branch source is missing") {
		t.Fatalf("error = %q, want branch source context", err.Error())
	}
	if len(applyRunner.inputs) != 0 {
		t.Fatalf("apply inputs len = %d, want 0", len(applyRunner.inputs))
	}
	if len(taskManager.moveTaskIDs) != 0 {
		t.Fatalf("move calls = %#v, want none", taskManager.moveTaskIDs)
	}
}

func TestRunProposalsOnceRejectsArchiveTaskWithoutBranchSource(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyArchiveTask("issue-1", "ENG-1", "Archive", taskmanager.PullRequest{})},
	}
	archiveRunner := &recordingArchiveRunner{}
	orch := testOrchestratorWithArchive(taskManager, &recordingProposalRunner{}, &recordingApplyRunner{}, archiveRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "pull request branch source is missing") {
		t.Fatalf("error = %q, want branch source context", err.Error())
	}
	if len(archiveRunner.inputs) != 0 {
		t.Fatalf("archive inputs len = %d, want 0", len(archiveRunner.inputs))
	}
	if len(taskManager.moveTaskIDs) != 0 {
		t.Fatalf("move calls = %#v, want none", taskManager.moveTaskIDs)
	}
}

func TestRunProposalsOnceArchivingInProgressMoveFailureDoesNotRunArchive(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyArchiveTask("issue-1", "ENG-1", "Archive", taskmanager.PullRequest{Branch: "feature/task"})},
		moveErrByStateID: map[string]error{
			"state-archiving-progress": errors.New("linear move failed"),
		},
	}
	archiveRunner := &recordingArchiveRunner{}
	orch := testOrchestratorWithArchive(taskManager, &recordingProposalRunner{}, &recordingApplyRunner{}, archiveRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "move to archiving in-progress state state-archiving-progress") {
		t.Fatalf("error = %q, want in-progress move context", err.Error())
	}
	if len(archiveRunner.inputs) != 0 {
		t.Fatalf("archive inputs len = %d, want 0", len(archiveRunner.inputs))
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-archiving-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceArchiveFailureDoesNotMoveTaskToArchiveReview(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyArchiveTask("issue-1", "ENG-1", "Archive", taskmanager.PullRequest{Branch: "feature/task"})},
	}
	archiveRunner := &recordingArchiveRunner{err: errors.New("archive failed")}
	orch := testOrchestratorWithArchive(taskManager, &recordingProposalRunner{}, &recordingApplyRunner{}, archiveRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "run archive") {
		t.Fatalf("error = %q, want archive context", err.Error())
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-archiving-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceArchiveReviewMoveFailureReturnsContext(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyArchiveTask("issue-1", "ENG-1", "Archive", taskmanager.PullRequest{Branch: "feature/task"})},
		moveErrByStateID: map[string]error{
			"state-archive-review": errors.New("linear move failed"),
		},
	}
	archiveRunner := &recordingArchiveRunner{}
	orch := testOrchestratorWithArchive(taskManager, &recordingProposalRunner{}, &recordingApplyRunner{}, archiveRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "move to archive review state state-archive-review") {
		t.Fatalf("error = %q, want archive review move context", err.Error())
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-archiving-progress,state-archive-review" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceCodeInProgressMoveFailureDoesNotRunApply(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyCodeTask("issue-1", "ENG-1", "Apply", taskmanager.PullRequest{Branch: "feature/task"})},
		moveErrByStateID: map[string]error{
			"state-code-progress": errors.New("linear move failed"),
		},
	}
	applyRunner := &recordingApplyRunner{}
	orch := testOrchestratorWithApply(taskManager, &recordingProposalRunner{}, applyRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "move to code in-progress state state-code-progress") {
		t.Fatalf("error = %q, want in-progress move context", err.Error())
	}
	if len(applyRunner.inputs) != 0 {
		t.Fatalf("apply inputs len = %d, want 0", len(applyRunner.inputs))
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-code-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceApplyFailureDoesNotMoveTaskToCodeReview(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyCodeTask("issue-1", "ENG-1", "Apply", taskmanager.PullRequest{Branch: "feature/task"})},
	}
	applyRunner := &recordingApplyRunner{err: errors.New("apply failed")}
	orch := testOrchestratorWithApply(taskManager, &recordingProposalRunner{}, applyRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "run apply") {
		t.Fatalf("error = %q, want apply context", err.Error())
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-code-progress" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsOnceCodeReviewMoveFailureReturnsContext(t *testing.T) {
	taskManager := &recordingTaskManager{
		tasks: []taskmanager.Task{readyCodeTask("issue-1", "ENG-1", "Apply", taskmanager.PullRequest{Branch: "feature/task"})},
		moveErrByStateID: map[string]error{
			"state-code-review": errors.New("linear move failed"),
		},
	}
	applyRunner := &recordingApplyRunner{}
	orch := testOrchestratorWithApply(taskManager, &recordingProposalRunner{}, applyRunner, nil)

	err := orch.RunProposalsOnce(context.Background())
	if err == nil {
		t.Fatal("RunProposalsOnce() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "move to code review state state-code-review") {
		t.Fatalf("error = %q, want code review move context", err.Error())
	}
	if got := strings.Join(taskManager.moveStateIDs, ","); got != "state-code-progress,state-code-review" {
		t.Fatalf("MoveTask states = %q", got)
	}
}

func TestRunProposalsLoopRepeatsAfterSuccessAndWaitsInterval(t *testing.T) {
	taskManager := &recordingTaskManager{}
	runner := &recordingProposalRunner{}
	var logs bytes.Buffer
	orch := testOrchestrator(taskManager, runner, &logs)
	ctx, cancel := context.WithCancel(context.Background())
	waiter := &recordingWaiter{
		cancelAfterCalls: 2,
		cancel:           cancel,
	}

	if err := orch.runProposalsLoop(ctx, 2*time.Second, waiter.Wait); err != nil {
		t.Fatalf("runProposalsLoop() returned error: %v", err)
	}

	if taskManager.getTasksCalls != 2 {
		t.Fatalf("GetTasks calls = %d, want 2", taskManager.getTasksCalls)
	}
	if got := durationsString(waiter.intervals); got != "2s,2s" {
		t.Fatalf("wait intervals = %s, want 2s,2s", got)
	}

	events := decodeEvents(t, logs.String())
	assertLogContains(t, events, "orchestration monitor iteration start iteration=1")
	assertLogContains(t, events, "orchestration monitor iteration start iteration=2")
	assertLogContains(t, events, "proposal monitor stopped: context canceled")
}

func TestRunProposalsLoopContinuesAfterIterationError(t *testing.T) {
	taskManager := &recordingTaskManager{getTasksErr: errors.New("linear unavailable")}
	runner := &recordingProposalRunner{}
	var logs bytes.Buffer
	orch := testOrchestrator(taskManager, runner, &logs)
	ctx, cancel := context.WithCancel(context.Background())
	waiter := &recordingWaiter{
		cancelAfterCalls: 2,
		cancel:           cancel,
	}

	if err := orch.runProposalsLoop(ctx, time.Second, waiter.Wait); err != nil {
		t.Fatalf("runProposalsLoop() returned error: %v", err)
	}

	if taskManager.getTasksCalls != 2 {
		t.Fatalf("GetTasks calls = %d, want 2", taskManager.getTasksCalls)
	}

	events := decodeEvents(t, logs.String())
	assertLogContains(t, events, "orchestration monitor iteration error iteration=1")
	assertLogContains(t, events, "orchestration monitor iteration start iteration=2")
}

func TestRunProposalsLoopStopsBeforeNextPassWhenContextCancelledDuringWait(t *testing.T) {
	taskManager := &recordingTaskManager{}
	runner := &recordingProposalRunner{}
	orch := testOrchestrator(taskManager, runner, nil)
	ctx, cancel := context.WithCancel(context.Background())
	waiter := &recordingWaiter{
		cancelAfterCalls: 1,
		cancel:           cancel,
	}

	if err := orch.runProposalsLoop(ctx, time.Second, waiter.Wait); err != nil {
		t.Fatalf("runProposalsLoop() returned error: %v", err)
	}
	if taskManager.getTasksCalls != 1 {
		t.Fatalf("GetTasks calls = %d, want 1", taskManager.getTasksCalls)
	}
}

func TestRunProposalsLoopRequiresPositiveInterval(t *testing.T) {
	orch := testOrchestrator(&recordingTaskManager{}, &recordingProposalRunner{}, nil)

	err := orch.runProposalsLoop(context.Background(), 0, func(ctx context.Context, interval time.Duration) error {
		t.Fatal("wait func must not be called")
		return nil
	})
	if err == nil {
		t.Fatal("runProposalsLoop() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "proposal poll interval") {
		t.Fatalf("error = %q, want poll interval context", err.Error())
	}
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
	wantPrompt := `Linear task:
ID: issue-1
Identifier: ENG-1
Title: Add proposal flow

Description:
Implement the proposal stage.

Comments:
1. Alex at 2026-04-25T06:30:00Z: Keep it minimal.
2. Dana: Check CLI mode.`
	if got.AgentPrompt != wantPrompt {
		t.Fatalf("BuildProposalInput().AgentPrompt =\n%s\nwant:\n%s", got.AgentPrompt, wantPrompt)
	}
	if got.Title != "Add proposal flow" {
		t.Fatalf("BuildProposalInput().Title = %q, want %q", got.Title, "Add proposal flow")
	}
	if got.Identifier != "ENG-1" {
		t.Fatalf("BuildProposalInput().Identifier = %q, want %q", got.Identifier, "ENG-1")
	}
}

func TestBuildProposalInputHandlesMissingDescriptionAndComments(t *testing.T) {
	task := readyTask("issue-1", "ENG-1", "Add proposal flow")

	got := BuildProposalInput(task)
	if !strings.Contains(got.AgentPrompt, "No description provided.") {
		t.Fatalf("input missing no-description marker:\n%s", got.AgentPrompt)
	}
	if !strings.Contains(got.AgentPrompt, "No comments available.") {
		t.Fatalf("input missing no-comments marker:\n%s", got.AgentPrompt)
	}
}

func TestBuildProposalInputProducesPRTitleFromTaskTitle(t *testing.T) {
	t.Run("identifier and title", func(t *testing.T) {
		task := readyTask("issue-1", "ZIM-42", "Add export feature")
		task.Description = "Implement the export pipeline."
		task.Comments = []taskmanager.Comment{{Body: "Looks good."}}

		input := BuildProposalInput(task)
		displayName := proposalrunner.BuildDisplayName(input.Identifier, input.Title)
		prTitle := proposalrunner.BuildPRTitle("OpenSpec proposal:", displayName)
		branchName := proposalrunner.BuildBranchName("codex/proposal", displayName, time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC))

		if !strings.Contains(prTitle, "ZIM-42: Add export feature") {
			t.Fatalf("prTitle = %q, want to contain %q", prTitle, "ZIM-42: Add export feature")
		}
		if strings.Contains(prTitle, "Linear task:") {
			t.Fatalf("prTitle = %q, must not contain leaked agent prompt header", prTitle)
		}
		if !strings.Contains(branchName, "zim-42-add-export-feature") {
			t.Fatalf("branchName = %q, want slug from identifier+title", branchName)
		}
	})

	t.Run("empty identifier falls back to title", func(t *testing.T) {
		task := readyTask("issue-1", "", "Refactor payments module")

		input := BuildProposalInput(task)
		displayName := proposalrunner.BuildDisplayName(input.Identifier, input.Title)
		prTitle := proposalrunner.BuildPRTitle("OpenSpec proposal:", displayName)

		if prTitle != "OpenSpec proposal: Refactor payments module" {
			t.Fatalf("prTitle = %q, want %q", prTitle, "OpenSpec proposal: Refactor payments module")
		}
		if strings.Contains(prTitle, ": :") || strings.HasSuffix(prTitle, ":") {
			t.Fatalf("prTitle = %q, must not have empty identifier marker", prTitle)
		}
	})

	t.Run("empty title uses fallback", func(t *testing.T) {
		task := readyTask("issue-1", "ENG-9", "")

		input := BuildProposalInput(task)
		if input.Title == "" {
			t.Fatal("BuildProposalInput().Title must not be empty after fallback")
		}
		displayName := proposalrunner.BuildDisplayName(input.Identifier, input.Title)
		prTitle := proposalrunner.BuildPRTitle("OpenSpec proposal:", displayName)
		if !strings.Contains(prTitle, input.Title) {
			t.Fatalf("prTitle = %q, want to contain fallback title %q", prTitle, input.Title)
		}
	})
}

func TestBuildApplyInputIncludesTaskPayloadAndBranchSource(t *testing.T) {
	task := readyCodeTask("issue-1", "ENG-1", "Apply flow", taskmanager.PullRequest{
		URL:    "https://github.com/example/repo/pull/42",
		Branch: "feature/task",
	})
	task.Description = "Implement code."
	task.Comments = []taskmanager.Comment{{Body: "Use existing patterns.", User: taskmanager.User{Name: "Dana"}}}

	got, err := BuildApplyInput(task)
	if err != nil {
		t.Fatalf("BuildApplyInput() error = %v", err)
	}
	if got.TaskID != "issue-1" || got.Identifier != "ENG-1" || got.Title != "Apply flow" {
		t.Fatalf("BuildApplyInput() = %#v", got)
	}
	if got.PRURL != "https://github.com/example/repo/pull/42" || got.BranchName != "feature/task" {
		t.Fatalf("branch source = pr %q branch %q", got.PRURL, got.BranchName)
	}
	if !strings.Contains(got.AgentPrompt, "Comments:") || !strings.Contains(got.AgentPrompt, "Use existing patterns.") {
		t.Fatalf("agent prompt = %q", got.AgentPrompt)
	}
}

func TestBuildApplyInputPassesGitLabMRURLAsOpaqueBranchSource(t *testing.T) {
	task := readyCodeTask("issue-1", "ENG-1", "Apply flow", taskmanager.PullRequest{
		URL: "https://gitlab.com/example/repo/-/merge_requests/42",
	})

	got, err := BuildApplyInput(task)
	if err != nil {
		t.Fatalf("BuildApplyInput() error = %v", err)
	}
	if got.PRURL != "https://gitlab.com/example/repo/-/merge_requests/42" || got.BranchName != "" {
		t.Fatalf("branch source = pr %q branch %q", got.PRURL, got.BranchName)
	}
}

func TestBuildArchiveInputIncludesTaskPayloadAndBranchSource(t *testing.T) {
	task := readyArchiveTask("issue-1", "ENG-1", "Archive flow", taskmanager.PullRequest{
		URL:    "https://github.com/example/repo/pull/42",
		Branch: "feature/task",
	})
	task.Description = "Archive accepted spec."
	task.Comments = []taskmanager.Comment{{Body: "Use existing patterns.", User: taskmanager.User{Name: "Dana"}}}

	got, err := BuildArchiveInput(task)
	if err != nil {
		t.Fatalf("BuildArchiveInput() error = %v", err)
	}
	if got.TaskID != "issue-1" || got.Identifier != "ENG-1" || got.Title != "Archive flow" {
		t.Fatalf("BuildArchiveInput() = %#v", got)
	}
	if got.PRURL != "https://github.com/example/repo/pull/42" || got.BranchName != "feature/task" {
		t.Fatalf("branch source = pr %q branch %q", got.PRURL, got.BranchName)
	}
	if !strings.Contains(got.AgentPrompt, "Comments:") || !strings.Contains(got.AgentPrompt, "Use existing patterns.") {
		t.Fatalf("agent prompt = %q", got.AgentPrompt)
	}
}

func TestBuildArchiveInputPassesGitLabMRURLAsOpaqueBranchSource(t *testing.T) {
	task := readyArchiveTask("issue-1", "ENG-1", "Archive flow", taskmanager.PullRequest{
		URL: "https://gitlab.com/example/repo/-/merge_requests/42",
	})

	got, err := BuildArchiveInput(task)
	if err != nil {
		t.Fatalf("BuildArchiveInput() error = %v", err)
	}
	if got.PRURL != "https://gitlab.com/example/repo/-/merge_requests/42" || got.BranchName != "" {
		t.Fatalf("branch source = pr %q branch %q", got.PRURL, got.BranchName)
	}
}

func TestBuildArchiveInputHandlesFallbackTitleAndPRURLOnly(t *testing.T) {
	task := readyArchiveTask("issue-1", "ENG-1", "", taskmanager.PullRequest{URL: "https://github.com/example/repo/pull/42"})

	got, err := BuildArchiveInput(task)
	if err != nil {
		t.Fatalf("BuildArchiveInput() error = %v", err)
	}
	if got.Title == "" {
		t.Fatal("BuildArchiveInput().Title must not be empty after fallback")
	}
	if got.PRURL != "https://github.com/example/repo/pull/42" || got.BranchName != "" {
		t.Fatalf("branch source = pr %q branch %q", got.PRURL, got.BranchName)
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
	getTasksCalls    int
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
	manager.getTasksCalls++
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
	inputs []proposalrunner.ProposalInput
	urls   []string
	err    error
}

type recordingApplyRunner struct {
	inputs []applyrunner.ApplyInput
	err    error
}

type recordingArchiveRunner struct {
	inputs []archiverunner.ArchiveInput
	err    error
}

func (runner *recordingApplyRunner) Run(ctx context.Context, input applyrunner.ApplyInput) error {
	runner.inputs = append(runner.inputs, input)
	return runner.err
}

func (runner *recordingArchiveRunner) Run(ctx context.Context, input archiverunner.ArchiveInput) error {
	runner.inputs = append(runner.inputs, input)
	return runner.err
}

func (runner *recordingProposalRunner) Run(ctx context.Context, input proposalrunner.ProposalInput) (string, error) {
	runner.inputs = append(runner.inputs, input)
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
	return testOrchestratorWithApply(taskManager, runner, &recordingApplyRunner{}, logs)
}

func testOrchestratorWithApply(taskManager TaskManager, runner ProposalRunner, applyRunner ApplyRunner, logs *bytes.Buffer) Orchestrator {
	return testOrchestratorWithArchive(taskManager, runner, applyRunner, &recordingArchiveRunner{}, logs)
}

func testOrchestratorWithArchive(taskManager TaskManager, runner ProposalRunner, applyRunner ApplyRunner, archiveRunner ArchiveRunner, logs *bytes.Buffer) Orchestrator {
	var logWriter ioWriter
	if logs != nil {
		logWriter = logs
	}

	return Orchestrator{
		Config: Config{
			ReadyToProposeStateID:      "state-propose",
			ProposingInProgressStateID: "state-proposing-progress",
			NeedProposalReviewStateID:  "state-proposal-review",
			ReadyToCodeStateID:         "state-code",
			CodeInProgressStateID:      "state-code-progress",
			NeedCodeReviewStateID:      "state-code-review",
			ReadyToArchiveStateID:      "state-ready-archive",
			ArchivingInProgressStateID: "state-archiving-progress",
			NeedArchiveReviewStateID:   "state-archive-review",
		},
		TaskManager:    taskManager,
		ProposalRunner: runner,
		ApplyRunner:    applyRunner,
		ArchiveRunner:  archiveRunner,
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

func readyCodeTask(id string, identifier string, title string, pullRequests ...taskmanager.PullRequest) taskmanager.Task {
	return taskmanager.Task{
		ID:           id,
		Identifier:   identifier,
		Title:        title,
		State:        taskmanager.WorkflowState{ID: "state-code", Name: "Ready to Code"},
		PullRequests: pullRequests,
	}
}

func readyArchiveTask(id string, identifier string, title string, pullRequests ...taskmanager.PullRequest) taskmanager.Task {
	return taskmanager.Task{
		ID:           id,
		Identifier:   identifier,
		Title:        title,
		State:        taskmanager.WorkflowState{ID: "state-ready-archive", Name: "Ready to Archive"},
		PullRequests: pullRequests,
	}
}

type ioWriter interface {
	Write(p []byte) (int, error)
}

type recordingWaiter struct {
	intervals        []time.Duration
	cancelAfterCalls int
	cancel           context.CancelFunc
}

func (waiter *recordingWaiter) Wait(ctx context.Context, interval time.Duration) error {
	waiter.intervals = append(waiter.intervals, interval)
	if waiter.cancelAfterCalls > 0 && len(waiter.intervals) >= waiter.cancelAfterCalls {
		waiter.cancel()
		return ctx.Err()
	}

	return ctx.Err()
}

func durationsString(durations []time.Duration) string {
	values := make([]string, 0, len(durations))
	for _, duration := range durations {
		values = append(values, duration.String())
	}

	return strings.Join(values, ",")
}
