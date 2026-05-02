package proposalrunner

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
)

func TestRunnerCommitMessageContainsProducerTrailerWhenProducerSet(t *testing.T) {
	cfg := validConfig()
	tempDir := filepath.Join(t.TempDir(), "orchv3-run")
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{}, // git clone
			{stdout: "?? openspec/changes/x/proposal.md\n"}, // git status
			{}, // git checkout -b
			{}, // git add -A
			{}, // git commit -m
			{}, // git push
			{stdout: "https://github.com/example/project/pull/42\n"}, // gh pr create
			{}, // gh pr comment
		},
	}
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   &fakeAgentExecutor{},
		Producer: agentmeta.Producer{
			By:    "codex",
			Model: "gpt-5-codex",
			Stage: agentmeta.StageProposal,
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) {
			return tempDir, nil
		},
		RemoveAll: func(path string) error { return nil },
		Now:       fixedTime,
	}

	if _, err := runner.Run(context.Background(), ProposalInput{Title: "t", AgentPrompt: "p"}); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	commitCmd := findCommitRecord(fake.commands)
	if commitCmd == nil {
		t.Fatal("no git commit command captured")
	}
	msg := commitCmd.args[len(commitCmd.args)-1]
	for _, want := range []string{"Produced-By: codex", "Produced-Model: gpt-5-codex", "Produced-Stage: proposal"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("commit message missing %q:\n%s", want, msg)
		}
	}
}

func TestRunnerCommitMessageOmitsTrailerWhenProducerUnset(t *testing.T) {
	cfg := validConfig()
	tempDir := filepath.Join(t.TempDir(), "orchv3-run")
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{}, {stdout: "?? a\n"}, {}, {}, {}, {}, {stdout: "https://github.com/example/project/pull/42\n"}, {},
		},
	}
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   &fakeAgentExecutor{},
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) {
			return tempDir, nil
		},
		RemoveAll: func(path string) error { return nil },
		Now:       fixedTime,
	}
	if _, err := runner.Run(context.Background(), ProposalInput{Title: "t", AgentPrompt: "p"}); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	commitCmd := findCommitRecord(fake.commands)
	if commitCmd == nil {
		t.Fatal("no git commit command captured")
	}
	msg := commitCmd.args[len(commitCmd.args)-1]
	if strings.Contains(msg, "Produced-By") {
		t.Fatalf("commit message should not contain producer trailer:\n%s", msg)
	}
}

func findCommitRecord(commands []recordedCommand) *recordedCommand {
	for i := range commands {
		c := &commands[i]
		if c.name == "git" && len(c.args) > 0 && c.args[0] == "commit" {
			return c
		}
	}
	return nil
}
