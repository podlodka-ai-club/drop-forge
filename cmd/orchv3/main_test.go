package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"orchv3/internal/applyrunner"
	"orchv3/internal/archiverunner"
	"orchv3/internal/config"
	"orchv3/internal/coreorch"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
	"orchv3/internal/taskmanager"
)

func TestRejectManualProposalInputFromArgs(t *testing.T) {
	err := rejectManualProposalInput([]string{"Add", "proposal", "flow"}, os.Stdin)
	if err == nil {
		t.Fatal("rejectManualProposalInput() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "manual proposal execution is unsupported in Drop Forge") {
		t.Fatalf("error = %q, want manual proposal removal context", err.Error())
	}
}

func TestRejectManualProposalInputFromPipe(t *testing.T) {
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

	err = rejectManualProposalInput(nil, readPipe)
	if err == nil {
		t.Fatal("rejectManualProposalInput() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "manual proposal execution is unsupported in Drop Forge") {
		t.Fatalf("error = %q, want manual proposal removal context", err.Error())
	}
}

func TestRunWithoutTaskStartsProposalMonitor(t *testing.T) {
	stdin := emptyTempFile(t)
	defer stdin.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	deps := testDeps()
	monitor := &fakeProposalMonitor{}
	deps.newProposalOrchestrator = func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, logOut io.Writer) proposalMonitor {
		return monitor
	}

	exitCode := runWithDeps(nil, stdin, &stdout, &stderr, deps)
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
	if !strings.Contains(event.Message, "orchv3-test starting Drop Forge orchestration monitor") {
		t.Fatalf("message = %q, want startup message", event.Message)
	}
	if monitor.calls != 1 {
		t.Fatalf("monitor calls = %d, want 1", monitor.calls)
	}
	if monitor.interval != 45*time.Second {
		t.Fatalf("monitor interval = %v, want 45s", monitor.interval)
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

func TestRunDefaultWiresDependenciesAndKeepsStdoutEmpty(t *testing.T) {
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
	deps.newApplyRunner = func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleApplyRunner {
		state.applyRunnerConfig = cfg
		state.applyRunnerService = service
		state.applyRunnerLogOut = logOut
		return &fakeSingleApplyRunner{}
	}
	deps.newArchiveRunner = func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleArchiveRunner {
		state.archiveRunnerConfig = cfg
		state.archiveRunnerService = service
		state.archiveRunnerLogOut = logOut
		return &fakeSingleArchiveRunner{}
	}
	deps.newProposalOrchestrator = func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, logOut io.Writer) proposalMonitor {
		state.orchestratorConfig = cfg
		state.orchestratorLogOut = logOut
		state.orchestratorTaskManager = tasks
		state.orchestratorRunner = proposalRunner
		state.orchestratorApplyRunner = applyRunner
		state.orchestratorArchiveRunner = archiveRunner
		return &fakeProposalMonitor{}
	}

	exitCode := runWithDeps(nil, stdin, &stdout, &stderr, deps)
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
	if state.applyRunnerConfig.RepositoryURL != "git@github.com:example/repo.git" {
		t.Fatalf("apply runner repository = %q", state.applyRunnerConfig.RepositoryURL)
	}
	if state.applyRunnerService != "orchv3-test" {
		t.Fatalf("apply runner service = %q", state.applyRunnerService)
	}
	if state.archiveRunnerConfig.RepositoryURL != "git@github.com:example/repo.git" {
		t.Fatalf("archive runner repository = %q", state.archiveRunnerConfig.RepositoryURL)
	}
	if state.archiveRunnerService != "orchv3-test" {
		t.Fatalf("archive runner service = %q", state.archiveRunnerService)
	}
	if state.orchestratorConfig.TaskManager.ReadyToProposeStateID != "state-propose" {
		t.Fatalf("orchestrator ready state = %q", state.orchestratorConfig.TaskManager.ReadyToProposeStateID)
	}
	if state.orchestratorConfig.TaskManager.ProposingInProgressStateID != "state-proposing-progress" {
		t.Fatalf("orchestrator proposing in-progress state = %q", state.orchestratorConfig.TaskManager.ProposingInProgressStateID)
	}
	if state.orchestratorConfig.TaskManager.CodeInProgressStateID != "state-code-progress" {
		t.Fatalf("orchestrator code in-progress state = %q", state.orchestratorConfig.TaskManager.CodeInProgressStateID)
	}
	if state.orchestratorConfig.TaskManager.NeedCodeReviewStateID != "state-code-review" {
		t.Fatalf("orchestrator need code review state = %q", state.orchestratorConfig.TaskManager.NeedCodeReviewStateID)
	}
	if state.orchestratorConfig.TaskManager.ArchivingInProgressStateID != "state-archiving-progress" {
		t.Fatalf("orchestrator archiving in-progress state = %q", state.orchestratorConfig.TaskManager.ArchivingInProgressStateID)
	}
	if state.orchestratorConfig.TaskManager.NeedArchiveReviewStateID != "state-archive-review" {
		t.Fatalf("orchestrator need archive review state = %q", state.orchestratorConfig.TaskManager.NeedArchiveReviewStateID)
	}
	if state.taskManagerLogOut == nil || state.runnerLogOut == nil || state.applyRunnerLogOut == nil || state.archiveRunnerLogOut == nil || state.orchestratorLogOut == nil {
		t.Fatal("log outputs should be wired")
	}
	if state.orchestratorTaskManager == nil || state.orchestratorRunner == nil || state.orchestratorApplyRunner == nil || state.orchestratorArchiveRunner == nil {
		t.Fatal("orchestrator dependencies should be wired")
	}
}

func TestDefaultProposalOrchestratorWiresApplyAndArchiveStates(t *testing.T) {
	cfg := config.Config{
		AppName: "orchv3-test",
		TaskManager: config.LinearTaskManagerConfig{
			ReadyToProposeStateID:      "state-propose",
			ReadyToCodeStateID:         "state-code",
			ReadyToArchiveStateID:      "state-archive",
			ProposingInProgressStateID: "state-proposing-progress",
			CodeInProgressStateID:      "state-code-progress",
			ArchivingInProgressStateID: "state-archiving-progress",
			NeedProposalReviewStateID:  "state-proposal-review",
			NeedCodeReviewStateID:      "state-code-review",
			NeedArchiveReviewStateID:   "state-archive-review",
		},
	}
	tasks := &fakeTaskManager{}
	runner := &fakeSingleProposalRunner{}
	apply := &fakeSingleApplyRunner{}
	archive := &fakeSingleArchiveRunner{}

	orchestrator, ok := defaultDeps().newProposalOrchestrator(cfg, tasks, runner, apply, archive, io.Discard).(*coreorch.Orchestrator)
	if !ok {
		t.Fatalf("orchestrator type = %T, want *coreorch.Orchestrator", orchestrator)
	}
	if orchestrator.Config.ProposingInProgressStateID != "state-proposing-progress" {
		t.Fatalf("ProposingInProgressStateID = %q", orchestrator.Config.ProposingInProgressStateID)
	}
	if orchestrator.Config.ReadyToCodeStateID != "state-code" {
		t.Fatalf("ReadyToCodeStateID = %q", orchestrator.Config.ReadyToCodeStateID)
	}
	if orchestrator.Config.CodeInProgressStateID != "state-code-progress" {
		t.Fatalf("CodeInProgressStateID = %q", orchestrator.Config.CodeInProgressStateID)
	}
	if orchestrator.Config.NeedCodeReviewStateID != "state-code-review" {
		t.Fatalf("NeedCodeReviewStateID = %q", orchestrator.Config.NeedCodeReviewStateID)
	}
	if orchestrator.Config.ReadyToArchiveStateID != "state-archive" {
		t.Fatalf("ReadyToArchiveStateID = %q", orchestrator.Config.ReadyToArchiveStateID)
	}
	if orchestrator.Config.ArchivingInProgressStateID != "state-archiving-progress" {
		t.Fatalf("ArchivingInProgressStateID = %q", orchestrator.Config.ArchivingInProgressStateID)
	}
	if orchestrator.Config.NeedArchiveReviewStateID != "state-archive-review" {
		t.Fatalf("NeedArchiveReviewStateID = %q", orchestrator.Config.NeedArchiveReviewStateID)
	}
	if orchestrator.ApplyRunner != apply {
		t.Fatal("apply runner should be wired")
	}
	if orchestrator.ArchiveRunner != archive {
		t.Fatal("archive runner should be wired")
	}
}

func TestRunArgsRejectsManualProposalModeWithoutCallingRunner(t *testing.T) {
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
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if len(runner.inputs) != 0 {
		t.Fatalf("runner inputs len = %d, want 0", len(runner.inputs))
	}

	event := decodeLogEvent(t, stderr.String())
	if !strings.Contains(event.Message, "manual proposal execution is unsupported in Drop Forge") {
		t.Fatalf("message = %q, want removal usage error", event.Message)
	}
}

func TestRunStdinRejectsManualProposalModeWithoutCallingRunner(t *testing.T) {
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	t.Cleanup(func() {
		readPipe.Close()
	})
	if _, err := writePipe.WriteString("manual task\n"); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	if err := writePipe.Close(); err != nil {
		t.Fatalf("close write pipe: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := &fakeSingleProposalRunner{prURL: "https://github.com/example/repo/pull/42"}
	deps := testDeps()
	deps.newProposalRunner = func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
		return runner
	}

	exitCode := runWithDeps(nil, readPipe, &stdout, &stderr, deps)
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if len(runner.inputs) != 0 {
		t.Fatalf("runner inputs len = %d, want 0", len(runner.inputs))
	}

	event := decodeLogEvent(t, stderr.String())
	if !strings.Contains(event.Message, "manual proposal execution is unsupported in Drop Forge") {
		t.Fatalf("message = %q, want removal usage error", event.Message)
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
	taskManagerConfig         config.LinearTaskManagerConfig
	taskManagerLogOut         io.Writer
	runnerConfig              config.ProposalRunnerConfig
	runnerService             string
	runnerLogOut              io.Writer
	applyRunnerConfig         config.ProposalRunnerConfig
	applyRunnerService        string
	applyRunnerLogOut         io.Writer
	archiveRunnerConfig       config.ProposalRunnerConfig
	archiveRunnerService      string
	archiveRunnerLogOut       io.Writer
	orchestratorConfig        config.Config
	orchestratorLogOut        io.Writer
	orchestratorTaskManager   coreorch.TaskManager
	orchestratorRunner        coreorch.ProposalRunner
	orchestratorApplyRunner   coreorch.ApplyRunner
	orchestratorArchiveRunner coreorch.ArchiveRunner
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

type fakeSingleApplyRunner struct {
	inputs []applyrunner.ApplyInput
	err    error
}

func (runner *fakeSingleApplyRunner) Run(ctx context.Context, input applyrunner.ApplyInput) error {
	runner.inputs = append(runner.inputs, input)
	return runner.err
}

type fakeSingleArchiveRunner struct {
	inputs []archiverunner.ArchiveInput
	err    error
}

func (runner *fakeSingleArchiveRunner) Run(ctx context.Context, input archiverunner.ArchiveInput) error {
	runner.inputs = append(runner.inputs, input)
	return runner.err
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

type fakeProposalMonitor struct {
	calls    int
	interval time.Duration
	err      error
}

func (monitor *fakeProposalMonitor) RunProposalsLoop(ctx context.Context, interval time.Duration) error {
	monitor.calls++
	monitor.interval = interval
	return monitor.err
}

func testDeps() appDeps {
	return appDeps{
		loadConfig: func() (config.Config, error) {
			return config.Config{
				AppName: "orchv3-test",
				ProposalRunner: config.ProposalRunnerConfig{
					RepositoryURL: "git@github.com:example/repo.git",
				},
				ProposalPollInterval: 45 * time.Second,
				TaskManager: config.LinearTaskManagerConfig{
					ProjectID:                  "project-123",
					ReadyToProposeStateID:      "state-propose",
					ReadyToCodeStateID:         "state-code",
					ReadyToArchiveStateID:      "state-archive",
					ProposingInProgressStateID: "state-proposing-progress",
					CodeInProgressStateID:      "state-code-progress",
					ArchivingInProgressStateID: "state-archiving-progress",
					NeedProposalReviewStateID:  "state-proposal-review",
					NeedCodeReviewStateID:      "state-code-review",
					NeedArchiveReviewStateID:   "state-archive-review",
				},
			}, nil
		},
		buildLogger: func(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error) {
			return steplog.NewWithService(stderr, cfg.AppName), stderr, nil, nil
		},
		newProposalRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
			return &fakeSingleProposalRunner{}
		},
		newApplyRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleApplyRunner {
			return &fakeSingleApplyRunner{}
		},
		newArchiveRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleArchiveRunner {
			return &fakeSingleArchiveRunner{}
		},
		newTaskManager: func(cfg config.LinearTaskManagerConfig, logOut io.Writer) coreorch.TaskManager {
			return &fakeTaskManager{}
		},
		newProposalOrchestrator: func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, logOut io.Writer) proposalMonitor {
			return &fakeProposalMonitor{}
		},
	}
}
