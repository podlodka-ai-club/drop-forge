package applyrunner

import (
	"context"
	"io"

	"orchv3/internal/commandrunner"
	"orchv3/internal/steplog"
)

func runLoggedCommand(ctx context.Context, service string, exec commandrunner.Runner, command commandrunner.Command, module string, stdout io.Writer, stderr io.Writer) error {
	stdoutLog := steplog.NewWithService(writerOrDiscard(stdout), service).LineWriter(module)
	stderrLog := steplog.NewWithService(writerOrDiscard(stderr), service).LineWriter(module)

	stdoutWriters := []io.Writer{stdoutLog}
	if command.Stdout != nil {
		stdoutWriters = append(stdoutWriters, command.Stdout)
	}
	command.Stdout = io.MultiWriter(stdoutWriters...)

	stderrWriters := []io.Writer{stderrLog}
	if command.Stderr != nil {
		stderrWriters = append(stderrWriters, command.Stderr)
	}
	command.Stderr = io.MultiWriter(stderrWriters...)

	err := exec.Run(ctx, command)
	stdoutLog.Flush()
	stderrLog.Flush()

	return err
}
