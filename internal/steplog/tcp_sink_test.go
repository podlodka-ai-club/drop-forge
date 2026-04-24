package steplog

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

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
