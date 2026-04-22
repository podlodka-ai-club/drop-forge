package commandrunner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecRunnerStreamsOutputAndLogsCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell argv differs on windows")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var logs bytes.Buffer

	err := ExecRunner{LogWriter: &logs}.Run(context.Background(), Command{
		Name:   "sh",
		Args:   []string{"-c", "printf stdout; printf stderr >&2"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if stdout.String() != "stdout" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "stdout")
	}

	if stderr.String() != "stderr" {
		t.Fatalf("stderr = %q, want %q", stderr.String(), "stderr")
	}

	if !strings.Contains(logs.String(), "[command] sh -c") {
		t.Fatalf("logs = %q, want command log", logs.String())
	}
}

func TestExecRunnerUsesWorkingDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell argv differs on windows")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "marker.txt"), []byte("ok"), 0600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	var stdout bytes.Buffer
	err := ExecRunner{}.Run(context.Background(), Command{
		Name:   "sh",
		Args:   []string{"-c", "pwd; test -f marker.txt"},
		Dir:    dir,
		Stdout: &stdout,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != dir {
		t.Fatalf("pwd = %q, want %q", got, dir)
	}
}

func TestExecRunnerReturnsContextualError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell argv differs on windows")
	}

	err := ExecRunner{}.Run(context.Background(), Command{
		Name: "sh",
		Args: []string{"-c", "exit 7"},
	})
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "run sh -c exit 7") {
		t.Fatalf("Run() error = %q, want command context", err.Error())
	}
}
