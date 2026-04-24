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
