package commandrunner

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"orchv3/internal/steplog"
)

type Command struct {
	Name   string
	Args   []string
	Dir    string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type Runner interface {
	Run(ctx context.Context, command Command) error
}

type ExecRunner struct {
	LogWriter io.Writer
}

func (runner ExecRunner) Run(ctx context.Context, command Command) error {
	if strings.TrimSpace(command.Name) == "" {
		return fmt.Errorf("command name must not be empty")
	}

	if runner.LogWriter != nil {
		steplog.New(runner.LogWriter).Infof("command", "%s", commandLine(command))
	}

	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdin = command.Stdin
	cmd.Stdout = writerOrDiscard(command.Stdout)
	cmd.Stderr = writerOrDiscard(command.Stderr)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %s: %w", commandLine(command), err)
	}

	return nil
}

func commandLine(command Command) string {
	parts := make([]string, 0, len(command.Args)+1)
	parts = append(parts, command.Name)
	parts = append(parts, command.Args...)

	return strings.Join(parts, " ")
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
