package applyrunner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

const codexLastMessageFile = "apply-codex-last-message.txt"

type CodexCLIExecutor struct {
	Config  config.ProposalRunnerConfig
	Command commandrunner.Runner
	Service string
}

func (executor CodexCLIExecutor) Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error) {
	command := executor.Command
	if command == nil {
		command = commandrunner.ExecRunner{LogWriter: writerOrDiscard(input.Stdout)}
	}

	prompt := buildCodexPrompt(input.TaskDescription)
	lastMessagePath := filepath.Join(input.TempDir, codexLastMessageFile)
	steplog.NewWithService(writerOrDiscard(input.Stdout), executor.Service).Infof("codex", "prompt:\n%s", prompt)
	if err := runLoggedCommand(ctx, executor.Service, command, commandrunner.Command{
		Name:  executor.Config.CodexPath,
		Args:  codexArgs(input.CloneDir, lastMessagePath),
		Dir:   input.CloneDir,
		Stdin: strings.NewReader(prompt),
	}, "codex", input.Stdout, input.Stderr); err != nil {
		return AgentExecutionResult{}, fmt.Errorf("codex apply: %w", err)
	}

	return AgentExecutionResult{}, nil
}

func buildCodexPrompt(taskDescription string) string {
	return fmt.Sprintf(`Use the openspec-apply-change skill to implement the OpenSpec change for the task below.

If more than one active OpenSpec change is present and the relevant change cannot be inferred from the task context, stop with a clear error instead of applying an arbitrary change.

Task context:
%s
`, strings.TrimSpace(taskDescription))
}

func codexArgs(cloneDir string, lastMessagePath string) []string {
	args := []string{"exec", "--json", "--sandbox", "danger-full-access"}
	if strings.TrimSpace(lastMessagePath) != "" {
		args = append(args, "--output-last-message", lastMessagePath)
	}
	args = append(args, "--cd", cloneDir, "-")

	return args
}
