package reviewrunner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/steplog"
)

// CodexCLIExecutor invokes the codex CLI with a stage-agnostic review prompt,
// reading the assistant's final message from --output-last-message.
type CodexCLIExecutor struct {
	Command   commandrunner.Runner
	CodexPath string
	Model     string
	Service   string
}

func (e CodexCLIExecutor) Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error) {
	if strings.TrimSpace(e.CodexPath) == "" {
		return AgentExecutionResult{}, fmt.Errorf("codex path must not be empty")
	}
	if strings.TrimSpace(input.CloneDir) == "" {
		return AgentExecutionResult{}, fmt.Errorf("clone dir must not be empty")
	}
	if strings.TrimSpace(input.TempDir) == "" {
		return AgentExecutionResult{}, fmt.Errorf("temp dir must not be empty")
	}
	finalPath := filepath.Join(input.TempDir, "review-final.txt")

	// Note: we do not pass --model to the Codex CLI. Codex with a ChatGPT
	// account rejects an explicit --model flag even when it matches the
	// account's default. The Model field is used only as metadata (for the
	// producer commit trailer and the published review summary) so a future
	// reviewer can see what produced the artefact. The CLI uses the account's
	// default model.
	args := []string{
		"exec", "--json",
		"--sandbox", "danger-full-access",
		"--output-last-message", finalPath,
		"--cd", input.CloneDir,
		"-",
	}

	if input.Stdout != nil {
		steplog.NewWithService(input.Stdout, e.Service).Infof("codex", "%s %s", e.CodexPath, strings.Join(args, " "))
	}

	cmd := commandrunner.Command{
		Name:   e.CodexPath,
		Args:   args,
		Stdin:  bytes.NewReader([]byte(input.Prompt)),
		Stdout: input.Stdout,
		Stderr: input.Stderr,
	}
	if err := e.Command.Run(ctx, cmd); err != nil {
		return AgentExecutionResult{}, fmt.Errorf("codex exec: %w", err)
	}
	final, err := os.ReadFile(finalPath)
	if err != nil {
		return AgentExecutionResult{}, fmt.Errorf("read final review message: %w", err)
	}
	return AgentExecutionResult{FinalMessage: strings.TrimSpace(string(final))}, nil
}
