package proposalrunner

import (
	"context"
	"io"
)

type AgentExecutionInput struct {
	TaskDescription string
	CloneDir        string
	TempDir         string
	Stdout          io.Writer
	Stderr          io.Writer
}

type AgentExecutionResult struct {
	FinalMessage string
}

type AgentExecutor interface {
	Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error)
}
