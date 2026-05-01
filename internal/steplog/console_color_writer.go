package steplog

import (
	"bytes"
	"encoding/json"
	"io"
)

const (
	ansiRed   = "\x1b[31m"
	ansiReset = "\x1b[0m"
)

type ConsoleColorWriter struct {
	out     io.Writer
	enabled bool
	buffer  bytes.Buffer
}

func NewConsoleColorWriter(out io.Writer, enabled bool) *ConsoleColorWriter {
	if out == nil {
		out = io.Discard
	}

	return &ConsoleColorWriter{out: out, enabled: enabled}
}

func (writer *ConsoleColorWriter) Write(p []byte) (int, error) {
	if !writer.enabled {
		return writer.out.Write(p)
	}

	written := len(p)
	for len(p) > 0 {
		index := bytes.IndexByte(p, '\n')
		if index < 0 {
			_, _ = writer.buffer.Write(p)
			return written, nil
		}

		_, _ = writer.buffer.Write(p[:index])
		if err := writer.writeCompleteLine(); err != nil {
			return 0, err
		}
		p = p[index+1:]
	}

	return written, nil
}

func (writer *ConsoleColorWriter) Flush() error {
	if writer.buffer.Len() == 0 {
		return nil
	}

	_, err := writer.out.Write(writer.buffer.Bytes())
	writer.buffer.Reset()
	return err
}

func (writer *ConsoleColorWriter) writeCompleteLine() error {
	line := append([]byte(nil), writer.buffer.Bytes()...)
	writer.buffer.Reset()

	if isErrorLogLine(line) {
		if _, err := io.WriteString(writer.out, ansiRed); err != nil {
			return err
		}
		if _, err := writer.out.Write(line); err != nil {
			return err
		}
		if _, err := io.WriteString(writer.out, ansiReset); err != nil {
			return err
		}
		if _, err := writer.out.Write([]byte("\n")); err != nil {
			return err
		}
		return nil
	}

	if _, err := writer.out.Write(line); err != nil {
		return err
	}
	_, err := writer.out.Write([]byte("\n"))
	return err
}

func isErrorLogLine(line []byte) bool {
	var event struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &event); err != nil {
		return false
	}

	return event.Type == TypeError
}
