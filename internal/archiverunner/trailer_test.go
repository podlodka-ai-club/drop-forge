package archiverunner

import (
	"context"
	"io"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
)

func TestRunnerCommitMessageContainsProducerTrailerWhenProducerSet(t *testing.T) {
	cfg := validConfig()
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{},                                // git clone
			{},                                // git checkout
			{stdout: " M internal/file.go\n"}, // git status
			{}, {}, {},                        // add, commit, push
		},
	}
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   &fakeAgentExecutor{},
		Producer: agentmeta.Producer{
			By:    "codex",
			Model: "gpt-5-codex",
			Stage: agentmeta.StageArchive,
		},
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) { return "/tmp/archive", nil },
		RemoveAll: func(path string) error { return nil },
	}
	err := runner.Run(context.Background(), ArchiveInput{
		Title:       "Archive feature",
		Identifier:  "ENG-1",
		AgentPrompt: "Task",
		BranchName:  "feature/task",
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	commitCmd := findCommitCommand(fake.commands)
	if commitCmd == nil {
		t.Fatal("no git commit command captured")
	}
	msg := commitCmd.Args[len(commitCmd.Args)-1]
	for _, want := range []string{"Produced-By: codex", "Produced-Model: gpt-5-codex", "Produced-Stage: archive"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("commit message missing %q:\n%s", want, msg)
		}
	}
}

func TestRunnerCommitMessageOmitsTrailerWhenProducerUnset(t *testing.T) {
	cfg := validConfig()
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{}, {}, {stdout: " M internal/file.go\n"}, {}, {}, {},
		},
	}
	runner := &Runner{
		Config:    cfg,
		Command:   fake,
		Agent:     &fakeAgentExecutor{},
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		MkdirTemp: func(dir, pattern string) (string, error) { return "/tmp/archive", nil },
		RemoveAll: func(path string) error { return nil },
	}
	err := runner.Run(context.Background(), ArchiveInput{
		Title:       "Archive feature",
		AgentPrompt: "Task",
		BranchName:  "feature/task",
	})
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	commitCmd := findCommitCommand(fake.commands)
	if commitCmd == nil {
		t.Fatal("no git commit command captured")
	}
	msg := commitCmd.Args[len(commitCmd.Args)-1]
	if strings.Contains(msg, "Produced-By") {
		t.Fatalf("unexpected trailer in commit message:\n%s", msg)
	}
}

func findCommitCommand(commands []commandrunner.Command) *commandrunner.Command {
	for i := range commands {
		c := &commands[i]
		if c.Name == "git" && len(c.Args) > 0 && c.Args[0] == "commit" {
			return c
		}
	}
	return nil
}
