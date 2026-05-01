package steplog

import (
	"bytes"
	"strings"
	"testing"
)

func TestConsoleColorWriterColorizesErrorLines(t *testing.T) {
	var out bytes.Buffer
	writer := NewConsoleColorWriter(&out, true)

	line := `{"module":"cli","type":"error","message":"failed"}`
	if _, err := writer.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	want := ansiRed + line + ansiReset + "\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestConsoleColorWriterLeavesInfoLinesUnchanged(t *testing.T) {
	var out bytes.Buffer
	writer := NewConsoleColorWriter(&out, true)

	line := `{"module":"cli","type":"info","message":"hello"}`
	if _, err := writer.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	want := line + "\n"
	if out.String() != want {
		t.Fatalf("output = %q, want %q", out.String(), want)
	}
}

func TestConsoleColorWriterLeavesMalformedLinesUnchanged(t *testing.T) {
	var out bytes.Buffer
	writer := NewConsoleColorWriter(&out, true)

	input := "not-json\n"
	if _, err := writer.Write([]byte(input)); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	if out.String() != input {
		t.Fatalf("output = %q, want %q", out.String(), input)
	}
}

func TestConsoleColorWriterFlushesPartialLineUnchanged(t *testing.T) {
	var out bytes.Buffer
	writer := NewConsoleColorWriter(&out, true)

	input := `{"module":"cli","type":"error","message":"partial"}`
	if _, err := writer.Write([]byte(input)); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("output before Flush() = %q, want empty", out.String())
	}

	if err := writer.Flush(); err != nil {
		t.Fatalf("Flush() returned error: %v", err)
	}

	if out.String() != input {
		t.Fatalf("output = %q, want %q", out.String(), input)
	}
}

func TestConsoleColorWriterBuffersSplitLines(t *testing.T) {
	var out bytes.Buffer
	writer := NewConsoleColorWriter(&out, true)

	if _, err := writer.Write([]byte(`{"type":"err`)); err != nil {
		t.Fatalf("first Write() returned error: %v", err)
	}
	if _, err := writer.Write([]byte(`or","message":"failed"}` + "\n")); err != nil {
		t.Fatalf("second Write() returned error: %v", err)
	}

	if !strings.HasPrefix(out.String(), ansiRed) || !strings.HasSuffix(out.String(), ansiReset+"\n") {
		t.Fatalf("output is not wrapped as a red line: %q", out.String())
	}
}

func TestConsoleColorWriterDisabledPassesThrough(t *testing.T) {
	var out bytes.Buffer
	writer := NewConsoleColorWriter(&out, false)

	input := `{"module":"cli","type":"error","message":"failed"}` + "\n"
	if _, err := writer.Write([]byte(input)); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}

	if out.String() != input {
		t.Fatalf("output = %q, want %q", out.String(), input)
	}
}
