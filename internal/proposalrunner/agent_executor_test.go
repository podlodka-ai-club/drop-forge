package proposalrunner

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunnerInvokesInjectedAgentExecutor(t *testing.T) {
	task := "Add project roadmap workflow"
	cfg := validConfig()
	tempDir := filepath.Join(t.TempDir(), "orchv3-run")
	cloneDir := filepath.Join(tempDir, "repo")
	fakeCommand := &fakeCommandRunner{
		responses: []fakeResponse{
			{},
			{stdout: "?? openspec/changes/add-project-roadmap-workflow/proposal.md\n"},
			{},
			{},
			{},
			{},
			{stdout: "https://github.com/example/project/pull/42\n"},
			{},
		},
	}
	fakeAgent := &fakeAgentExecutor{
		result: AgentExecutionResult{FinalMessage: "Final agent response for PR comment."},
	}

	runner := &Runner{
		Config:  cfg,
		Command: fakeCommand,
		Agent:   fakeAgent,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) {
			return tempDir, nil
		},
		RemoveAll: func(path string) error {
			return nil
		},
		Now: fixedTime,
	}

	prURL, err := runner.Run(context.Background(), task)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if prURL != "https://github.com/example/project/pull/42" {
		t.Fatalf("prURL = %q", prURL)
	}
	if len(fakeAgent.inputs) != 1 {
		t.Fatalf("agent calls = %d, want 1", len(fakeAgent.inputs))
	}

	input := fakeAgent.inputs[0]
	if input.TaskDescription != task {
		t.Fatalf("agent task = %q, want %q", input.TaskDescription, task)
	}
	if input.CloneDir != cloneDir {
		t.Fatalf("agent clone dir = %q, want %q", input.CloneDir, cloneDir)
	}
	if input.TempDir != tempDir {
		t.Fatalf("agent temp dir = %q, want %q", input.TempDir, tempDir)
	}

	for _, command := range fakeCommand.commands {
		if command.name == cfg.CodexPath {
			t.Fatalf("runner executed Codex command directly: %#v", command)
		}
	}
}

func TestRunnerReturnsAgentExecutorFailure(t *testing.T) {
	tempDir := filepath.Join(t.TempDir(), "orchv3-run")
	errBoom := errors.New("agent failed")
	fakeCommand := &fakeCommandRunner{responses: []fakeResponse{{}}}
	runner := &Runner{
		Config:  validConfig(),
		Command: fakeCommand,
		Agent:   &fakeAgentExecutor{err: errBoom},
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) {
			return tempDir, nil
		},
		RemoveAll: func(path string) error {
			return nil
		},
		Now: fixedTime,
	}

	_, err := runner.Run(context.Background(), "Task")
	if err == nil || !strings.Contains(err.Error(), "agent proposal") {
		t.Fatalf("Run() error = %v, want agent proposal context", err)
	}
	if len(fakeCommand.commands) != 1 {
		t.Fatalf("commands = %d, want only git clone before agent failure", len(fakeCommand.commands))
	}
	assertCommand(t, fakeCommand.commands[0], "git", []string{"clone", validConfig().RepositoryURL, filepath.Join(tempDir, "repo")}, tempDir)
}

type fakeAgentExecutor struct {
	inputs []AgentExecutionInput
	result AgentExecutionResult
	err    error
}

func (executor *fakeAgentExecutor) Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error) {
	executor.inputs = append(executor.inputs, input)
	return executor.result, executor.err
}
