package proposalrunner

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCodexCLIExecutorRunsCommandAndReturnsLastMessage(t *testing.T) {
	tempDir := t.TempDir()
	cloneDir := filepath.Join(tempDir, "repo")
	lastMessagePath := filepath.Join(tempDir, codexLastMessageFile)
	fakeCommand := &fakeCommandRunner{
		responses: []fakeResponse{{
			stdout:           "codex stdout",
			stderr:           "codex stderr",
			writeLastMessage: true,
			lastMessage:      "\nFinal Codex response\n",
		}},
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	result, err := CodexCLIExecutor{
		Config:  validConfig(),
		Command: fakeCommand,
	}.Run(context.Background(), AgentExecutionInput{
		TaskDescription: "Describe task",
		CloneDir:        cloneDir,
		TempDir:         tempDir,
		Stdout:          &stdout,
		Stderr:          &stderr,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if result.FinalMessage != "Final Codex response" {
		t.Fatalf("FinalMessage = %q, want trimmed Codex response", result.FinalMessage)
	}
	if len(fakeCommand.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(fakeCommand.commands))
	}

	command := fakeCommand.commands[0]
	wantArgs := []string{"exec", "--json", "--sandbox", "danger-full-access", "--output-last-message", lastMessagePath, "--cd", cloneDir, "-"}
	if command.name != "codex" {
		t.Fatalf("command name = %q, want codex", command.name)
	}
	if !reflect.DeepEqual(command.args, wantArgs) {
		t.Fatalf("command args = %#v, want %#v", command.args, wantArgs)
	}
	if command.dir != cloneDir {
		t.Fatalf("command dir = %q, want %q", command.dir, cloneDir)
	}
	if !strings.Contains(command.stdin, "openspec-propose") || !strings.Contains(command.stdin, "Describe task") {
		t.Fatalf("command stdin = %q, want prompt with skill and task", command.stdin)
	}

	stdoutEvents := decodeLogEvents(t, stdout.String())
	assertLogMessage(t, stdoutEvents, "codex", "codex stdout")
	stderrEvents := decodeLogEvents(t, stderr.String())
	assertLogMessage(t, stderrEvents, "codex", "codex stderr")
}

func TestCodexCLIExecutorReturnsCommandFailure(t *testing.T) {
	errBoom := errors.New("codex failed")
	_, err := CodexCLIExecutor{
		Config:  validConfig(),
		Command: &fakeCommandRunner{responses: []fakeResponse{{err: errBoom}}},
	}.Run(context.Background(), AgentExecutionInput{
		TaskDescription: "Describe task",
		CloneDir:        filepath.Join(t.TempDir(), "repo"),
		TempDir:         t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "codex proposal") {
		t.Fatalf("Run() error = %v, want codex context", err)
	}
}
