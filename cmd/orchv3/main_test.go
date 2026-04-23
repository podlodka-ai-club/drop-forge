package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
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
