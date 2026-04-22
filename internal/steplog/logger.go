package steplog

import (
	"fmt"
	"io"
)

type Logger struct {
	out io.Writer
}

func New(out io.Writer) Logger {
	if out == nil {
		out = io.Discard
	}

	return Logger{out: out}
}

func (logger Logger) Printf(step string, format string, args ...any) {
	fmt.Fprintf(logger.out, "[%s] %s\n", step, fmt.Sprintf(format, args...))
}
