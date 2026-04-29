package reviewrunner

import (
	"context"
	"io"
)

type AgentExecutionInput struct {
	Prompt   string
	CloneDir string
	TempDir  string
	Stdout   io.Writer
	Stderr   io.Writer
}

type AgentExecutionResult struct {
	FinalMessage string
}

type AgentExecutor interface {
	Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error)
}
