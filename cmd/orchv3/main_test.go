package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"orchv3/internal/config"
	"orchv3/internal/coreorch"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
	"orchv3/internal/taskmanager"
)

func TestReadTaskDescriptionFromArgs(t *testing.T) {
	got, err := readTaskDescription([]string{"Add", "proposal", "flow"}, os.Stdin)
	if err != nil {
		t.Fatalf("readTaskDescription() returned error: %v", err)
	}

	if got != "Add proposal flow" {
		t.Fatalf("description = %q, want %q", got, "Add proposal flow")
	}
}

func TestReadTaskDescriptionFromPipe(t *testing.T) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	t.Cleanup(func() {
		readPipe.Close()
	})

	if _, err := writePipe.WriteString("  Add stdin task\n"); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	if err := writePipe.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}

	got, err := readTaskDescription(nil, readPipe)
	if err != nil {
		t.Fatalf("readTaskDescription() returned error: %v", err)
	}

	if got != "Add stdin task" {
		t.Fatalf("description = %q, want %q", got, "Add stdin task")
	}
}

func TestRunWithoutTaskLogsStartupAsJSON(t *testing.T) {
	t.Setenv("APP_NAME", "orchv3-test")
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_PORT", "19090")

	stdin := emptyTempFile(t)
	defer stdin.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode := run(nil, stdin, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	event := decodeLogEvent(t, stderr.String())
	if event.Module != "cli" {
		t.Fatalf("module = %q, want %q", event.Module, "cli")
	}
	if event.Type != "info" {
		t.Fatalf("type = %q, want %q", event.Type, "info")
	}
	if !strings.Contains(event.Message, "orchv3-test starting in test on port 19090") {
		t.Fatalf("message = %q, want startup message", event.Message)
	}
}

func TestRunConfigErrorLogsJSONError(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-port")

	var stderr bytes.Buffer
	exitCode := run(nil, os.Stdin, io.Discard, &stderr)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}

	event := decodeLogEvent(t, stderr.String())
	if event.Module != "cli" {
		t.Fatalf("module = %q, want %q", event.Module, "cli")
	}
	if event.Type != "error" {
		t.Fatalf("type = %q, want %q", event.Type, "error")
	}
	if !strings.Contains(event.Message, "load config") {
		t.Fatalf("message = %q, want config context", event.Message)
	}
}

func TestRunOrchestrateProposalsWiresDependenciesAndKeepsStdoutEmpty(t *testing.T) {
	stdin := emptyTempFile(t)
	defer stdin.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := testDeps()
	state := &cliTestState{}
	deps.newTaskManager = func(cfg config.LinearTaskManagerConfig, logOut io.Writer) coreorch.TaskManager {
		state.taskManagerConfig = cfg
		state.taskManagerLogOut = logOut
		return &fakeTaskManager{}
	}
	deps.newProposalRunner = func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
		state.runnerConfig = cfg
		state.runnerService = service
		state.runnerLogOut = logOut
		return &fakeSingleProposalRunner{prURL: "https://github.com/example/repo/pull/1"}
	}
	deps.newProposalOrchestrator = func(cfg config.Config, tasks coreorch.TaskManager, runner coreorch.ProposalRunner, logOut io.Writer) proposalOrchestrator {
		state.orchestratorConfig = cfg
		state.orchestratorLogOut = logOut
		state.orchestratorTaskManager = tasks
		state.orchestratorRunner = runner
		return &fakeProposalOrchestrator{}
	}

	exitCode := runWithDeps([]string{orchestrateProposalsCommand}, stdin, &stdout, &stderr, deps)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if state.taskManagerConfig.ProjectID != "project-123" {
		t.Fatalf("task manager project = %q", state.taskManagerConfig.ProjectID)
	}
	if state.runnerConfig.RepositoryURL != "git@github.com:example/repo.git" {
		t.Fatalf("runner repository = %q", state.runnerConfig.RepositoryURL)
	}
	if state.runnerService != "orchv3-test" {
		t.Fatalf("runner service = %q", state.runnerService)
	}
	if state.orchestratorConfig.TaskManager.ReadyToProposeStateID != "state-propose" {
		t.Fatalf("orchestrator ready state = %q", state.orchestratorConfig.TaskManager.ReadyToProposeStateID)
	}
	if state.orchestratorConfig.TaskManager.ProposingInProgressStateID != "state-proposing-progress" {
		t.Fatalf("orchestrator proposing in-progress state = %q", state.orchestratorConfig.TaskManager.ProposingInProgressStateID)
	}
	if state.taskManagerLogOut == nil || state.runnerLogOut == nil || state.orchestratorLogOut == nil {
		t.Fatal("log outputs should be wired")
	}
	if state.orchestratorTaskManager == nil || state.orchestratorRunner == nil {
		t.Fatal("orchestrator dependencies should be wired")
	}
}

func TestDefaultProposalOrchestratorWiresProposingInProgressState(t *testing.T) {
	cfg := config.Config{
		AppName: "orchv3-test",
		TaskManager: config.LinearTaskManagerConfig{
			ReadyToProposeStateID:      "state-propose",
			ProposingInProgressStateID: "state-proposing-progress",
			NeedProposalReviewStateID:  "state-proposal-review",
		},
	}
	tasks := &fakeTaskManager{}
	runner := &fakeSingleProposalRunner{}

	orchestrator, ok := defaultDeps().newProposalOrchestrator(cfg, tasks, runner, io.Discard).(*coreorch.Orchestrator)
	if !ok {
		t.Fatalf("orchestrator type = %T, want *coreorch.Orchestrator", orchestrator)
	}
	if orchestrator.Config.ProposingInProgressStateID != "state-proposing-progress" {
		t.Fatalf("ProposingInProgressStateID = %q", orchestrator.Config.ProposingInProgressStateID)
	}
}

func TestRunDirectProposalModeStillPrintsPRURL(t *testing.T) {
	stdin := emptyTempFile(t)
	defer stdin.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &fakeSingleProposalRunner{prURL: "https://github.com/example/repo/pull/42"}
	deps := testDeps()
	deps.newProposalRunner = func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
		return runner
	}

	exitCode := runWithDeps([]string{"Add", "proposal", "flow"}, stdin, &stdout, &stderr, deps)
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stdout.String() != "https://github.com/example/repo/pull/42\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if len(runner.inputs) != 1 {
		t.Fatalf("runner inputs len = %d, want 1", len(runner.inputs))
	}
	got := runner.inputs[0]
	if got.Title != "Add proposal flow" {
		t.Fatalf("runner inputs[0].Title = %q, want %q", got.Title, "Add proposal flow")
	}
	if got.AgentPrompt != "Add proposal flow" {
		t.Fatalf("runner inputs[0].AgentPrompt = %q, want %q", got.AgentPrompt, "Add proposal flow")
	}
	if got.Identifier != "" {
		t.Fatalf("runner inputs[0].Identifier = %q, want empty", got.Identifier)
	}
}

type logEvent struct {
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func decodeLogEvent(t *testing.T, output string) logEvent {
	t.Helper()

	var event logEvent
	if err := json.Unmarshal(bytes.TrimSpace([]byte(output)), &event); err != nil {
		t.Fatalf("decode log event %q: %v", output, err)
	}

	return event
}

func emptyTempFile(t *testing.T) *os.File {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("create stdin file: %v", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("seek stdin file: %v", err)
	}

	return file
}

type cliTestState struct {
	taskManagerConfig       config.LinearTaskManagerConfig
	taskManagerLogOut       io.Writer
	runnerConfig            config.ProposalRunnerConfig
	runnerService           string
	runnerLogOut            io.Writer
	orchestratorConfig      config.Config
	orchestratorLogOut      io.Writer
	orchestratorTaskManager coreorch.TaskManager
	orchestratorRunner      coreorch.ProposalRunner
}

type fakeSingleProposalRunner struct {
	inputs []proposalrunner.ProposalInput
	prURL  string
	err    error
}

func (runner *fakeSingleProposalRunner) Run(ctx context.Context, input proposalrunner.ProposalInput) (string, error) {
	runner.inputs = append(runner.inputs, input)
	return runner.prURL, runner.err
}

type fakeTaskManager struct{}

func (manager *fakeTaskManager) GetTasks(ctx context.Context) ([]taskmanager.Task, error) {
	return nil, nil
}

func (manager *fakeTaskManager) AddPR(ctx context.Context, taskID string, prURL string) error {
	return nil
}

func (manager *fakeTaskManager) MoveTask(ctx context.Context, taskID string, stateID string) error {
	return nil
}

type fakeProposalOrchestrator struct {
	err error
}

func (orchestrator *fakeProposalOrchestrator) RunProposalsOnce(ctx context.Context) error {
	return orchestrator.err
}

func testDeps() appDeps {
	return appDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{
				AppName: "orchv3-test",
				ProposalRunner: config.ProposalRunnerConfig{
					RepositoryURL: "git@github.com:example/repo.git",
				},
				TaskManager: config.LinearTaskManagerConfig{
					ProjectID:                  "project-123",
					ReadyToProposeStateID:      "state-propose",
					ProposingInProgressStateID: "state-proposing-progress",
					NeedProposalReviewStateID:  "state-proposal-review",
				},
			}, nil
		},
		buildLogger: func(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error) {
			return steplog.NewWithService(stderr, cfg.AppName), stderr, nil, nil
		},
		newProposalRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
			return &fakeSingleProposalRunner{}
		},
		newTaskManager: func(cfg config.LinearTaskManagerConfig, logOut io.Writer) coreorch.TaskManager {
			return &fakeTaskManager{}
		},
		newProposalOrchestrator: func(cfg config.Config, tasks coreorch.TaskManager, runner coreorch.ProposalRunner, logOut io.Writer) proposalOrchestrator {
			return &fakeProposalOrchestrator{}
		},
	}
}
