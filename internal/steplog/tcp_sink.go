package steplog

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultBackoffMin       = 1 * time.Second
	defaultBackoffMax       = 30 * time.Second
	defaultDropWarnInterval = 30 * time.Second

	closeFlushTimeout = 2 * time.Second
)

type TCPSink struct {
	addr         string
	dialTimeout  time.Duration
	warnOut      io.Writer
	backoffMin   time.Duration
	backoffMax   time.Duration
	warnInterval time.Duration

	queue chan []byte

	dropped atomic.Uint64
	done    chan struct{}
	wg      sync.WaitGroup
}

func NewTCPSink(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer) *TCPSink {
	return newTCPSinkWithBackoffAndWarnInterval(
		addr, bufferSize, dialTimeout, warnOut,
		defaultBackoffMin, defaultBackoffMax, defaultDropWarnInterval,
	)
}

func newTCPSinkWithBackoff(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer, backoffMin, backoffMax time.Duration) *TCPSink {
	return newTCPSinkWithBackoffAndWarnInterval(
		addr, bufferSize, dialTimeout, warnOut,
		backoffMin, backoffMax, defaultDropWarnInterval,
	)
}

func newTCPSinkWithBackoffAndWarnInterval(
	addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer,
	backoffMin, backoffMax, warnInterval time.Duration,
) *TCPSink {
	if bufferSize < 1 {
		bufferSize = 1
	}
	if warnOut == nil {
		warnOut = io.Discard
	}

	sink := &TCPSink{
		addr:         addr,
		dialTimeout:  dialTimeout,
		warnOut:      warnOut,
		backoffMin:   backoffMin,
		backoffMax:   backoffMax,
		warnInterval: warnInterval,
		queue:        make(chan []byte, bufferSize),
		done:         make(chan struct{}),
	}

	sink.wg.Add(2)
	go sink.run()
	go sink.warnLoop()

	return sink
}

func (s *TCPSink) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case s.queue <- buf:
	default:
		s.dropped.Add(1)
	}
	return len(p), nil
}

func (s *TCPSink) Close() error {
	close(s.done)
	s.wg.Wait()
	return nil
}

func (s *TCPSink) Dropped() uint64 {
	return s.dropped.Load()
}

func (s *TCPSink) run() {
	defer s.wg.Done()

	backoff := s.backoffMin
	firstFailureLogged := false

	for {
		select {
		case <-s.done:
			s.finalFlush()
			return
		default:
		}

		conn, err := net.DialTimeout("tcp", s.addr, s.dialTimeout)
		if err != nil {
			if !firstFailureLogged {
				s.warnf("steplog: logstash sink unavailable at %s, will retry: %v", s.addr, err)
				firstFailureLogged = true
			}
			select {
			case <-s.done:
				s.finalFlush()
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > s.backoffMax {
				backoff = s.backoffMax
			}
			continue
		}

		backoff = s.backoffMin
		firstFailureLogged = false
		s.drain(conn)
	}
}

// finalFlush is called when Close is observed without a live connection.
// It tries one last dial and flushes pending events if any.
func (s *TCPSink) finalFlush() {
	if len(s.queue) == 0 {
		return
	}
	conn, err := net.DialTimeout("tcp", s.addr, s.dialTimeout)
	if err != nil {
		return
	}
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	s.flushRemaining(writer)
}

func (s *TCPSink) drain(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-s.done:
			s.flushRemaining(writer)
			return
		case payload := <-s.queue:
			if _, err := writer.Write(payload); err != nil {
				return
			}
			if err := writer.Flush(); err != nil {
				return
			}
		}
	}
}

func (s *TCPSink) flushRemaining(writer *bufio.Writer) {
	deadline := time.Now().Add(closeFlushTimeout)
	for time.Now().Before(deadline) {
		select {
		case payload := <-s.queue:
			if _, err := writer.Write(payload); err != nil {
				return
			}
		default:
			_ = writer.Flush()
			return
		}
	}
	_ = writer.Flush()
}

func (s *TCPSink) warnf(format string, args ...any) {
	_, _ = fmt.Fprintln(s.warnOut, fmt.Sprintf(format, args...))
}

func (s *TCPSink) warnLoop() {
	defer s.wg.Done()
	if s.warnInterval <= 0 {
		return
	}
	ticker := time.NewTicker(s.warnInterval)
	defer ticker.Stop()

	var lastReported uint64
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			current := s.dropped.Load()
			if current > lastReported {
				s.warnf("steplog: dropped %d events due to sink overflow", current-lastReported)
				lastReported = current
			}
		}
	}
}
