package steplog

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// safeBuffer is a bytes.Buffer guarded by a mutex so tests can read it
// while a background goroutine writes to it.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func startTestListener(t *testing.T) (net.Listener, string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener, listener.Addr().String()
}

func readLines(t *testing.T, listener net.Listener, want int, timeout time.Duration) []string {
	t.Helper()
	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	scanner := bufio.NewScanner(conn)
	lines := make([]string, 0, want)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) == want {
			break
		}
	}
	_ = conn.Close()
	return lines
}

func TestTCPSink_NonBlockingWrite(t *testing.T) {
	// No listener — DialTimeout fails, queue fills, further writes must not block.
	sink := NewTCPSink("127.0.0.1:1", 8, 50*time.Millisecond, io.Discard)
	t.Cleanup(func() { _ = sink.Close() })

	start := time.Now()
	for i := 0; i < 1000; i++ {
		if _, err := sink.Write([]byte("payload\n")); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("1000 writes took %v, want < 200ms", elapsed)
	}
}

func TestTCPSink_DropsOnOverflow(t *testing.T) {
	// No listener, bufferSize=4, write 10 → 4 buffered, 6 dropped.
	sink := NewTCPSink("127.0.0.1:1", 4, 50*time.Millisecond, io.Discard)
	t.Cleanup(func() { _ = sink.Close() })

	for i := 0; i < 10; i++ {
		_, _ = sink.Write([]byte("x\n"))
	}

	// Give the goroutine a beat in case it raced a couple of items out of the queue.
	time.Sleep(10 * time.Millisecond)

	got := sink.Dropped()
	if got < 6 {
		t.Fatalf("Dropped() = %d, want >= 6", got)
	}
}

func TestTCPSink_ReconnectsAfterServerRestart(t *testing.T) {
	listener1, addr := startTestListener(t)

	// Short reconnect baseline for the test.
	sink := newTCPSinkWithBackoff(addr, 16, 200*time.Millisecond, io.Discard, 50*time.Millisecond, 500*time.Millisecond)
	t.Cleanup(func() { _ = sink.Close() })

	_, _ = sink.Write([]byte("first\n"))
	lines := readLines(t, listener1, 1, 2*time.Second)
	if len(lines) != 1 || lines[0] != "first" {
		t.Fatalf("first batch = %v", lines)
	}

	_ = listener1.Close()

	// Bring up a new listener on the same port.
	listener2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("re-listen on %s: %v", addr, err)
	}
	t.Cleanup(func() { _ = listener2.Close() })

	// Write until reconnect succeeds; goroutine backoff retries dial.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, _ = sink.Write([]byte("second\n"))
		time.Sleep(50 * time.Millisecond)
	}

	lines = readLines(t, listener2, 1, 3*time.Second)
	if len(lines) == 0 {
		t.Fatalf("no lines received after reconnect")
	}
	if lines[0] != "second" {
		t.Fatalf("post-reconnect line = %q, want %q", lines[0], "second")
	}
}

func TestTCPSink_CloseFlushesPending(t *testing.T) {
	listener, addr := startTestListener(t)

	sink := NewTCPSink(addr, 64, 200*time.Millisecond, io.Discard)

	for i := 0; i < 20; i++ {
		_, _ = sink.Write([]byte(fmt.Sprintf("evt-%02d\n", i)))
	}

	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	lines := readLines(t, listener, 20, 3*time.Second)
	if len(lines) != 20 {
		t.Fatalf("received %d lines, want 20: %v", len(lines), lines)
	}
	for i, line := range lines {
		want := fmt.Sprintf("evt-%02d", i)
		if line != want {
			t.Fatalf("line[%d] = %q, want %q", i, line, want)
		}
	}
}

func TestTCPSink_PeriodicDropWarning(t *testing.T) {
	var warn safeBuffer
	sink := newTCPSinkWithBackoffAndWarnInterval(
		"127.0.0.1:1", 2, 50*time.Millisecond, &warn,
		50*time.Millisecond, 200*time.Millisecond, 100*time.Millisecond,
	)
	t.Cleanup(func() { _ = sink.Close() })

	for i := 0; i < 20; i++ {
		_, _ = sink.Write([]byte("x\n"))
	}

	// Wait long enough for at least one tick of the warning interval (100ms).
	time.Sleep(300 * time.Millisecond)

	output := warn.String()
	if !strings.Contains(output, "dropped") {
		t.Fatalf("warn output missing drop summary: %q", output)
	}
}

func TestTCPSink_DeliversEvents(t *testing.T) {
	listener, addr := startTestListener(t)

	sink := NewTCPSink(addr, 16, 200*time.Millisecond, io.Discard)
	t.Cleanup(func() { _ = sink.Close() })

	for i := 0; i < 10; i++ {
		if _, err := sink.Write([]byte("event-" + string(rune('0'+i)) + "\n")); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	lines := readLines(t, listener, 10, 2*time.Second)
	if len(lines) != 10 {
		t.Fatalf("received %d lines, want 10: %v", len(lines), lines)
	}
	for i, line := range lines {
		want := "event-" + string(rune('0'+i))
		if !strings.HasPrefix(line, want) {
			t.Fatalf("line[%d] = %q, want prefix %q", i, line, want)
		}
	}
}
