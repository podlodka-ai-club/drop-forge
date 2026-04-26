package proposalrunner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/steplog"
)

const codexLastMessageFile = "codex-last-message.txt"

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
		return AgentExecutionResult{}, fmt.Errorf("codex proposal: %w", err)
	}

	lastMessage, err := readLastCodexMessage(lastMessagePath)
	if err != nil {
		return AgentExecutionResult{}, fmt.Errorf("read final codex message: %w", err)
	}

	return AgentExecutionResult{FinalMessage: lastMessage}, nil
}

func buildCodexPrompt(taskDescription string) string {
	return fmt.Sprintf(`Use the openspec-propose skill to create a complete OpenSpec proposal for the task below.

Task description:
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

func readLastCodexMessage(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}
