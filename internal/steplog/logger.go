package steplog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	TypeInfo  = "info"
	TypeError = "error"
)

type Event struct {
	Time    string `json:"time"`
	Module  string `json:"module"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type Logger struct {
	out io.Writer
}

func New(out io.Writer) Logger {
	if out == nil {
		out = io.Discard
	}

	return Logger{out: out}
}

func (logger Logger) Infof(module string, format string, args ...any) {
	logger.write(module, TypeInfo, format, args...)
}

func (logger Logger) Errorf(module string, format string, args ...any) {
	logger.write(module, TypeError, format, args...)
}

func (logger Logger) LineWriter(module string) *LineWriter {
	return &LineWriter{
		logger:  logger,
		module:  normalizeModule(module),
		logType: TypeInfo,
	}
}

func (logger Logger) write(module string, logType string, format string, args ...any) {
	event := Event{
		Time:    time.Now().UTC().Format(time.RFC3339Nano),
		Module:  normalizeModule(module),
		Type:    logType,
		Message: fmt.Sprintf(format, args...),
	}

	encoder := json.NewEncoder(logger.out)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(event)
}

type LineWriter struct {
	logger  Logger
	module  string
	logType string
	buffer  bytes.Buffer
}

func (writer *LineWriter) Write(p []byte) (int, error) {
	written := len(p)
	for len(p) > 0 {
		index := bytes.IndexByte(p, '\n')
		if index < 0 {
			_, _ = writer.buffer.Write(p)
			return written, nil
		}

		_, _ = writer.buffer.Write(p[:index])
		writer.flushBuffer()
		p = p[index+1:]
	}

	return written, nil
}

func (writer *LineWriter) Flush() {
	if writer.buffer.Len() == 0 {
		return
	}

	writer.flushBuffer()
}

func (writer *LineWriter) flushBuffer() {
	message := strings.TrimSuffix(writer.buffer.String(), "\r")
	writer.buffer.Reset()
	writer.logger.write(writer.module, writer.logType, "%s", message)
}

func normalizeModule(module string) string {
	module = strings.TrimSpace(module)
	if module == "" {
		return "unknown"
	}

	return module
}
