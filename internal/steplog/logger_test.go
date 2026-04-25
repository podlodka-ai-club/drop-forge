package steplog

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestInfofWritesJSONEvent(t *testing.T) {
	var out bytes.Buffer

	New(&out).Infof("cli", "starting %s", "orchv3")

	events := decodeEvents(t, out.String())
	if len(events) != 1 {
		t.Fatalf("events count = %d, want 1", len(events))
	}

	event := events[0]
	if event.Module != "cli" {
		t.Fatalf("module = %q, want %q", event.Module, "cli")
	}
	if event.Type != TypeInfo {
		t.Fatalf("type = %q, want %q", event.Type, TypeInfo)
	}
	if event.Message != "starting orchv3" {
		t.Fatalf("message = %q, want %q", event.Message, "starting orchv3")
	}
	if event.Time == "" {
		t.Fatal("time is empty")
	}
	if _, err := time.Parse(time.RFC3339Nano, event.Time); err != nil {
		t.Fatalf("time = %q is not RFC3339Nano: %v", event.Time, err)
	}
	if parsed, _ := time.Parse(time.RFC3339Nano, event.Time); parsed.Location() != time.UTC {
		t.Fatalf("time location = %v, want UTC", parsed.Location())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Fatalf("output = %q, want newline-terminated JSON", out.String())
	}
}

func TestErrorfWritesJSONEvent(t *testing.T) {
	var out bytes.Buffer

	New(&out).Errorf("proposalrunner", "failed: %s", "git clone")

	event := decodeEvents(t, out.String())[0]
	if event.Type != TypeError {
		t.Fatalf("type = %q, want %q", event.Type, TypeError)
	}
	if event.Message != "failed: git clone" {
		t.Fatalf("message = %q, want %q", event.Message, "failed: git clone")
	}
}

func TestModuleNormalization(t *testing.T) {
	var out bytes.Buffer

	New(&out).Infof(" \t\n", "message")

	event := decodeEvents(t, out.String())[0]
	if event.Module != "unknown" {
		t.Fatalf("module = %q, want %q", event.Module, "unknown")
	}
}

func TestSafeMessageEncoding(t *testing.T) {
	var out bytes.Buffer
	message := "line one\nline \"two\" with \\ slash"

	New(&out).Infof("codex", "%s", message)

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("line count = %d, want 1 for multiline message", len(lines))
	}

	event := decodeEvents(t, out.String())[0]
	if event.Message != message {
		t.Fatalf("message = %q, want %q", event.Message, message)
	}
}

func TestMultipleEventsAreSeparateLines(t *testing.T) {
	var out bytes.Buffer
	logger := New(&out)

	logger.Infof("git", "first")
	logger.Errorf("git", "second")

	events := decodeEvents(t, out.String())
	if len(events) != 2 {
		t.Fatalf("events count = %d, want 2", len(events))
	}
	if events[0].Message != "first" || events[1].Message != "second" {
		t.Fatalf("messages = %#v", events)
	}
}

func TestLineWriterWritesOneEventPerLine(t *testing.T) {
	var out bytes.Buffer
	writer := New(&out).LineWriter("codex")

	if _, err := writer.Write([]byte("first\nsecond")); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	writer.Flush()

	events := decodeEvents(t, out.String())
	if len(events) != 2 {
		t.Fatalf("events count = %d, want 2", len(events))
	}
	if events[0].Module != "codex" || events[0].Message != "first" {
		t.Fatalf("first event = %#v", events[0])
	}
	if events[1].Module != "codex" || events[1].Message != "second" {
		t.Fatalf("second event = %#v", events[1])
	}
}

func TestNewWithServiceIncludesServiceField(t *testing.T) {
	var out bytes.Buffer

	NewWithService(&out, "orchv3-test").Infof("cli", "hello")

	event := decodeEvents(t, out.String())[0]
	if event.Service != "orchv3-test" {
		t.Fatalf("service = %q, want %q", event.Service, "orchv3-test")
	}
}

func TestNewOmitsServiceField(t *testing.T) {
	var out bytes.Buffer

	New(&out).Infof("cli", "hello")

	if strings.Contains(out.String(), `"service"`) {
		t.Fatalf("output contains service field: %s", out.String())
	}
}

func decodeEvents(t *testing.T, output string) []Event {
	t.Helper()

	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	events := make([]Event, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode %q: %v", line, err)
		}
		events = append(events, event)
	}

	return events
}
