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
	defaultBackoffMin = 1 * time.Second
	defaultBackoffMax = 30 * time.Second
)

type TCPSink struct {
	addr        string
	dialTimeout time.Duration
	warnOut     io.Writer
	backoffMin  time.Duration
	backoffMax  time.Duration

	queue chan []byte

	dropped atomic.Uint64
	done    chan struct{}
	wg      sync.WaitGroup
}

func NewTCPSink(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer) *TCPSink {
	return newTCPSinkWithBackoff(addr, bufferSize, dialTimeout, warnOut, defaultBackoffMin, defaultBackoffMax)
}

func newTCPSinkWithBackoff(addr string, bufferSize int, dialTimeout time.Duration, warnOut io.Writer, backoffMin, backoffMax time.Duration) *TCPSink {
	if bufferSize < 1 {
		bufferSize = 1
	}
	if warnOut == nil {
		warnOut = io.Discard
	}

	sink := &TCPSink{
		addr:        addr,
		dialTimeout: dialTimeout,
		warnOut:     warnOut,
		backoffMin:  backoffMin,
		backoffMax:  backoffMax,
		queue:       make(chan []byte, bufferSize),
		done:        make(chan struct{}),
	}

	sink.wg.Add(1)
	go sink.run()

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

func (s *TCPSink) drain(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	for {
		select {
		case <-s.done:
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

func (s *TCPSink) warnf(format string, args ...any) {
	_, _ = fmt.Fprintln(s.warnOut, fmt.Sprintf(format, args...))
}
