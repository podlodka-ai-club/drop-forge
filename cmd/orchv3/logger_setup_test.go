package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"

	"orchv3/internal/config"
)

type testEvent struct {
	Service string `json:"service"`
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func TestBuildLogger_DisabledWhenAddrEmpty(t *testing.T) {
	var stderr bytes.Buffer

	logger, out, closer, err := buildLogger(&stderr, config.Config{
		AppName:  "orchv3",
		Logstash: config.LogstashConfig{Addr: ""},
	}, io.Discard)
	if err != nil {
		t.Fatalf("buildLogger: %v", err)
	}
	if closer != nil {
		t.Fatal("closer != nil when sink disabled")
	}
	if out == nil {
		t.Fatal("out writer nil when sink disabled")
	}
	logger.Infof("cli", "hello")

	var evt testEvent
	if err := json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &evt); err != nil {
		t.Fatalf("decode stderr: %v", err)
	}
	if evt.Service != "orchv3" {
		t.Fatalf("service = %q, want orchv3", evt.Service)
	}
}

func TestBuildLogger_NonInteractiveStderrPreservesErrorJSON(t *testing.T) {
	var stderr bytes.Buffer

	logger, _, closer, err := buildLoggerWithTerminalCheck(&stderr, config.Config{
		AppName:  "orchv3",
		Logstash: config.LogstashConfig{Addr: ""},
	}, io.Discard, func(io.Writer) bool { return false })
	if err != nil {
		t.Fatalf("buildLogger: %v", err)
	}
	if closer != nil {
		t.Fatal("closer != nil when sink disabled")
	}

	logger.Errorf("cli", "failed")

	if bytes.Contains(stderr.Bytes(), []byte("\x1b[")) {
		t.Fatalf("stderr contains ANSI escape sequences: %q", stderr.String())
	}
	var evt testEvent
	if err := json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &evt); err != nil {
		t.Fatalf("decode stderr: %v", err)
	}
	if evt.Type != "error" {
		t.Fatalf("type = %q, want error", evt.Type)
	}
}

func TestBuildLogger_FanoutsToSinkWhenEnabled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	var stderr bytes.Buffer
	logger, out, closer, err := buildLogger(&stderr, config.Config{
		AppName: "orchv3",
		Logstash: config.LogstashConfig{
			Addr:        listener.Addr().String(),
			BufferSize:  16,
			DialTimeout: 200 * time.Millisecond,
		},
	}, io.Discard)
	if err != nil {
		t.Fatalf("buildLogger: %v", err)
	}
	if closer == nil {
		t.Fatal("closer == nil when sink enabled")
	}
	if out == nil {
		t.Fatal("out writer nil when sink enabled")
	}
	t.Cleanup(func() { _ = closer.Close() })

	// Accept in background and capture first line.
	accepted := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			accepted <- ""
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 512)
		n, _ := conn.Read(buf)
		accepted <- string(buf[:n])
		_ = conn.Close()
	}()

	logger.Infof("cli", "hello")

	select {
	case got := <-accepted:
		if got == "" {
			t.Fatal("sink received no data")
		}
		// Must be a valid JSON event with correct service.
		var evt testEvent
		if err := json.Unmarshal([]byte(got[:len(got)-1]), &evt); err != nil { // strip trailing newline
			t.Fatalf("decode sink payload: %v (raw %q)", err, got)
		}
		if evt.Service != "orchv3" {
			t.Fatalf("sink event.service = %q, want orchv3", evt.Service)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for sink delivery")
	}

	// Stderr must also have received the event.
	if !bytes.Contains(stderr.Bytes(), []byte(`"service":"orchv3"`)) {
		t.Fatalf("stderr missing event: %q", stderr.String())
	}
}

func TestBuildLogger_ColorizesConsoleAndLeavesSinkRaw(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	var stderr bytes.Buffer
	logger, _, closer, err := buildLoggerWithTerminalCheck(&stderr, config.Config{
		AppName: "orchv3",
		Logstash: config.LogstashConfig{
			Addr:        listener.Addr().String(),
			BufferSize:  16,
			DialTimeout: 200 * time.Millisecond,
		},
	}, io.Discard, func(io.Writer) bool { return true })
	if err != nil {
		t.Fatalf("buildLogger: %v", err)
	}
	if closer == nil {
		t.Fatal("closer == nil when sink enabled")
	}
	t.Cleanup(func() { _ = closer.Close() })

	accepted := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			accepted <- ""
			return
		}
		defer conn.Close()

		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		buf := make([]byte, 512)
		n, _ := conn.Read(buf)
		accepted <- string(buf[:n])
	}()

	logger.Errorf("cli", "failed")

	if !bytes.Contains(stderr.Bytes(), []byte("\x1b[31m")) || !bytes.Contains(stderr.Bytes(), []byte("\x1b[0m")) {
		t.Fatalf("stderr is not colorized: %q", stderr.String())
	}

	select {
	case got := <-accepted:
		if got == "" {
			t.Fatal("sink received no data")
		}
		if bytes.Contains([]byte(got), []byte("\x1b[")) {
			t.Fatalf("sink payload contains ANSI escape sequences: %q", got)
		}
		var evt testEvent
		if err := json.Unmarshal(bytes.TrimSpace([]byte(got)), &evt); err != nil {
			t.Fatalf("decode sink payload: %v (raw %q)", err, got)
		}
		if evt.Type != "error" {
			t.Fatalf("sink event.type = %q, want error", evt.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for sink delivery")
	}
}
