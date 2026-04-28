package applyrunner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/gitmanager"
)

func TestRunnerHappyPathResolvesPRBranchCommitsAndPushes(t *testing.T) {
	cfg := validConfig()
	tempDir := filepath.Join(t.TempDir(), "orchv3-run")
	cloneDir := filepath.Join(tempDir, "repo")
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{},
			{stdout: "codex/proposal/branch\n"},
			{},
			{stdout: " M internal/file.go\n"},
			{},
			{},
			{},
		},
	}
	agent := &fakeAgentExecutor{}
	var stdout bytes.Buffer
	removed := false
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   agent,
		Stdout:  &stdout,
		Stderr:  io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) {
			if pattern != defaultTempPattern {
				t.Fatalf("temp pattern = %q, want %q", pattern, defaultTempPattern)
			}
			return tempDir, nil
		},
		RemoveAll: func(path string) error {
			removed = true
			return nil
		},
	}

	err := runner.Run(context.Background(), ApplyInput{
		TaskID:      "issue-1",
		Identifier:  "ENG-1",
		Title:       "Apply feature",
		AgentPrompt: "Task context",
		PRURL:       "https://github.com/example/repo/pull/42",
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if removed {
		t.Fatal("RemoveAll called with CleanupTemp=false")
	}

	assertCommand(t, fake.commands[0], "git", []string{"clone", cfg.RepositoryURL, cloneDir}, tempDir)
	assertCommand(t, fake.commands[1], "gh", []string{"pr", "view", "https://github.com/example/repo/pull/42", "--json", "headRefName", "--jq", ".headRefName"}, cloneDir)
	assertCommand(t, fake.commands[2], "git", []string{"checkout", "codex/proposal/branch"}, cloneDir)
	assertCommand(t, fake.commands[3], "git", []string{"status", "--short"}, cloneDir)
	assertCommand(t, fake.commands[4], "git", []string{"add", "-A"}, cloneDir)
	assertCommand(t, fake.commands[5], "git", []string{"commit", "-m", "Apply: ENG-1: Apply feature"}, cloneDir)
	assertCommand(t, fake.commands[6], "git", []string{"push", "origin", "codex/proposal/branch"}, cloneDir)

	if len(agent.inputs) != 1 {
		t.Fatalf("agent inputs len = %d, want 1", len(agent.inputs))
	}
	if agent.inputs[0].TaskDescription != "Task context" || agent.inputs[0].CloneDir != cloneDir {
		t.Fatalf("agent input = %#v", agent.inputs[0])
	}
	if !strings.Contains(stdout.String(), "M internal/file.go") {
		t.Fatalf("stdout logs = %q, want git status", stdout.String())
	}
}

func TestRunnerGitLabModeResolvesMRBranchCommitsAndPushes(t *testing.T) {
	cfg := validConfig()
	cfg.GitProvider = config.GitProviderGitLab
	cfg.GHPath = ""
	cfg.GLabPath = "glab"
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{},
			{stdout: `{"source_branch":"codex/proposal/gitlab"}`},
			{},
			{stdout: " M internal/file.go\n"},
			{},
			{},
			{},
		},
	}
	runner := &Runner{
		Config:    cfg,
		Command:   fake,
		Agent:     &fakeAgentExecutor{},
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) { return "/tmp/orchv3-apply", nil },
		RemoveAll: func(path string) error { return nil },
	}

	err := runner.Run(context.Background(), ApplyInput{
		Identifier:  "ENG-1",
		Title:       "Apply feature",
		AgentPrompt: "Task context",
		PRURL:       "https://gitlab.com/example/repo/-/merge_requests/42",
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	cloneDir := "/tmp/orchv3-apply/repo"
	assertCommand(t, fake.commands[1], "glab", []string{"mr", "view", "https://gitlab.com/example/repo/-/merge_requests/42", "--output", "json"}, cloneDir)
	assertCommand(t, fake.commands[2], "git", []string{"checkout", "codex/proposal/gitlab"}, cloneDir)
	assertCommand(t, fake.commands[6], "git", []string{"push", "origin", "codex/proposal/gitlab"}, cloneDir)
}

func TestRunnerUsesInjectedGitManager(t *testing.T) {
	git := &fakeGitManager{
		workspace:      gitmanager.Workspace{TempDir: "/tmp/apply", CloneDir: "/tmp/apply/repo"},
		resolvedBranch: "feature/task",
		status:         " M file.go\n",
	}
	agent := &fakeAgentExecutor{}
	runner := &Runner{
		Config: validConfig(),
		Git:    git,
		Agent:  agent,
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	err := runner.Run(context.Background(), ApplyInput{
		Identifier:  "ENG-1",
		Title:       "Apply feature",
		AgentPrompt: "Task context",
		PRURL:       "https://github.com/example/repo/pull/42",
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if got, want := strings.Join(git.calls, ","), "clone,resolve,checkout,status,commit-push,close"; got != want {
		t.Fatalf("git calls = %q, want %q", got, want)
	}
	if len(agent.inputs) != 1 || agent.inputs[0].CloneDir != "/tmp/apply/repo" {
		t.Fatalf("agent inputs = %#v", agent.inputs)
	}
}

func TestRunnerUsesProvidedBranchWithoutGitHubLookupAndCleansTemp(t *testing.T) {
	cfg := validConfig()
	cfg.CleanupTemp = true
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{},
			{},
			{stdout: " M file.go\n"},
			{},
			{},
			{},
		},
	}
	var removedPath string
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		Agent:   &fakeAgentExecutor{},
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) {
			return "/tmp/orchv3-apply", nil
		},
		RemoveAll: func(path string) error {
			removedPath = path
			return nil
		},
	}

	err := runner.Run(context.Background(), ApplyInput{
		Title:       "Apply feature",
		AgentPrompt: "Task context",
		BranchName:  "feature/task",
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if removedPath != "/tmp/orchv3-apply" {
		t.Fatalf("removedPath = %q", removedPath)
	}
	assertCommand(t, fake.commands[1], "git", []string{"checkout", "feature/task"}, "/tmp/orchv3-apply/repo")
	for _, command := range fake.commands {
		if command.Name == "gh" || command.Name == "glab" {
			t.Fatalf("unexpected provider command: %#v", command)
		}
	}
}

func TestRunnerRejectsMissingBranchSourceBeforeSideEffects(t *testing.T) {
	fake := &fakeCommandRunner{}
	mkdirCalled := false
	runner := &Runner{
		Config:  validConfig(),
		Command: fake,
		MkdirTemp: func(dir string, pattern string) (string, error) {
			mkdirCalled = true
			return "", nil
		},
	}

	err := runner.Run(context.Background(), ApplyInput{Title: "Task", AgentPrompt: "Context"})
	if err == nil || !strings.Contains(err.Error(), "branch source") {
		t.Fatalf("Run() error = %v, want branch source context", err)
	}
	if mkdirCalled || len(fake.commands) != 0 {
		t.Fatal("side effects happened before input validation")
	}
}

func TestRunnerReturnsConfigValidationBeforeSideEffects(t *testing.T) {
	cfg := validConfig()
	cfg.RepositoryURL = " "
	fake := &fakeCommandRunner{}
	mkdirCalled := false
	runner := &Runner{
		Config:  cfg,
		Command: fake,
		MkdirTemp: func(dir string, pattern string) (string, error) {
			mkdirCalled = true
			return "", nil
		},
	}

	err := runner.Run(context.Background(), ApplyInput{Title: "Task", AgentPrompt: "Context", BranchName: "branch"})
	if err == nil || !strings.Contains(err.Error(), "PROPOSAL_REPOSITORY_URL") {
		t.Fatalf("Run() error = %v, want repository context", err)
	}
	if mkdirCalled || len(fake.commands) != 0 {
		t.Fatal("side effects happened before config validation")
	}
}

func TestRunnerReturnsMissingChangesError(t *testing.T) {
	err := runWithResponses(t, []fakeResponse{{}, {}, {stdout: "\n"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "no changes produced") {
		t.Fatalf("Run() error = %v, want no changes context", err)
	}
}

func TestRunnerWrapsCommandErrors(t *testing.T) {
	errBoom := errors.New("push failed")
	err := runWithResponses(t, []fakeResponse{
		{},
		{},
		{stdout: " M file.go\n"},
		{},
		{},
		{err: errBoom},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "git push") || !strings.Contains(err.Error(), "push failed") {
		t.Fatalf("Run() error = %v, want wrapped push context", err)
	}
}

func TestRunnerWrapsPRBranchResolutionErrors(t *testing.T) {
	errBoom := errors.New("gh auth failed")
	fake := &fakeCommandRunner{responses: []fakeResponse{{}, {err: errBoom}}}
	runner := &Runner{
		Config:    validConfig(),
		Command:   fake,
		Agent:     &fakeAgentExecutor{},
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) { return "/tmp/apply", nil },
		RemoveAll: func(path string) error { return nil },
	}

	err := runner.Run(context.Background(), ApplyInput{
		Title:       "Task",
		AgentPrompt: "Context",
		PRURL:       "https://github.com/example/repo/pull/42",
	})
	if err == nil || !strings.Contains(err.Error(), "github resolve pr branch") {
		t.Fatalf("Run() error = %v, want github branch context", err)
	}
}

func TestCodexPromptUsesApplySkill(t *testing.T) {
	prompt := buildCodexPrompt("Task context")
	if !strings.Contains(prompt, "openspec-apply-change") || !strings.Contains(prompt, "Task context") {
		t.Fatalf("buildCodexPrompt() = %q", prompt)
	}

	if got, want := strings.Join(codexArgs("/tmp/clone", "/tmp/last-message.txt"), " "), "exec --json --sandbox danger-full-access --output-last-message /tmp/last-message.txt --cd /tmp/clone -"; got != want {
		t.Fatalf("codexArgs() = %q, want %q", got, want)
	}
}

func runWithResponses(t *testing.T, responses []fakeResponse, agent *fakeAgentExecutor) error {
	t.Helper()
	if agent == nil {
		agent = &fakeAgentExecutor{}
	}
	runner := &Runner{
		Config:    validConfig(),
		Command:   &fakeCommandRunner{responses: responses},
		Agent:     agent,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		MkdirTemp: func(dir string, pattern string) (string, error) { return "/tmp/orchv3-apply", nil },
		RemoveAll: func(path string) error { return nil },
	}

	return runner.Run(context.Background(), ApplyInput{
		Title:       "Apply task",
		AgentPrompt: "Task context",
		BranchName:  "feature/task",
	})
}

func assertCommand(t *testing.T, got commandrunner.Command, wantName string, wantArgs []string, wantDir string) {
	t.Helper()

	if got.Name != wantName {
		t.Fatalf("command name = %q, want %q", got.Name, wantName)
	}
	if !reflect.DeepEqual(got.Args, wantArgs) {
		t.Fatalf("command args = %#v, want %#v", got.Args, wantArgs)
	}
	if got.Dir != wantDir {
		t.Fatalf("command dir = %q, want %q", got.Dir, wantDir)
	}
}

func validConfig() config.ProposalRunnerConfig {
	return config.ProposalRunnerConfig{
		RepositoryURL: "git@github.com:example/repo.git",
		RemoteName:    "origin",
		GitPath:       "git",
		CodexPath:     "codex",
		GHPath:        "gh",
		GLabPath:      "glab",
	}
}

type fakeCommandRunner struct {
	commands  []commandrunner.Command
	responses []fakeResponse
}

type fakeResponse struct {
	stdout string
	stderr string
	err    error
}

func (runner *fakeCommandRunner) Run(ctx context.Context, command commandrunner.Command) error {
	runner.commands = append(runner.commands, command)
	index := len(runner.commands) - 1
	if index >= len(runner.responses) {
		return nil
	}

	response := runner.responses[index]
	if response.stdout != "" && command.Stdout != nil {
		_, _ = command.Stdout.Write([]byte(response.stdout))
	}
	if response.stderr != "" && command.Stderr != nil {
		_, _ = command.Stderr.Write([]byte(response.stderr))
	}

	return response.err
}

type fakeAgentExecutor struct {
	inputs []AgentExecutionInput
	err    error
}

func (executor *fakeAgentExecutor) Run(ctx context.Context, input AgentExecutionInput) (AgentExecutionResult, error) {
	executor.inputs = append(executor.inputs, input)
	return AgentExecutionResult{}, executor.err
}

type fakeGitManager struct {
	calls          []string
	workspace      gitmanager.Workspace
	resolvedBranch string
	status         string
	err            error
}

func (manager *fakeGitManager) Clone(ctx context.Context) (gitmanager.Workspace, error) {
	manager.calls = append(manager.calls, "clone")
	return manager.workspace, manager.err
}

func (manager *fakeGitManager) Close(workspace gitmanager.Workspace) error {
	manager.calls = append(manager.calls, "close")
	return nil
}

func (manager *fakeGitManager) ResolvePullRequestBranch(ctx context.Context, cloneDir string, prURL string) (string, error) {
	manager.calls = append(manager.calls, "resolve")
	return manager.resolvedBranch, manager.err
}

func (manager *fakeGitManager) Checkout(ctx context.Context, cloneDir string, branchName string) error {
	manager.calls = append(manager.calls, "checkout")
	return manager.err
}

func (manager *fakeGitManager) StatusShort(ctx context.Context, cloneDir string) (string, error) {
	manager.calls = append(manager.calls, "status")
	return manager.status, manager.err
}

func (manager *fakeGitManager) CommitAllAndPush(ctx context.Context, cloneDir string, branchName string, message string, setUpstream bool) error {
	manager.calls = append(manager.calls, "commit-push")
	return manager.err
}
