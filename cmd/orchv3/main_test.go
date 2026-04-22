package main

import (
	"os"
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
