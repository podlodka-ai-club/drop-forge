package gitmanager

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
)

func TestManagerCloneStatusCommitPushAndCreatePR(t *testing.T) {
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{},
			{stdout: " M file.go\n"},
			{},
			{},
			{},
			{},
			{stdout: `{"url":"https://github.com/example/repo/pull/7"}`},
			{},
		},
	}
	var stdout bytes.Buffer
	manager := newTestManager(fake, &stdout)

	workspace, err := manager.Clone(context.Background())
	if err != nil {
		t.Fatalf("Clone() returned error: %v", err)
	}
	if workspace.TempDir != "/tmp/orchv3-git" || workspace.CloneDir != "/tmp/orchv3-git/repo" {
		t.Fatalf("workspace = %#v", workspace)
	}

	status, err := manager.StatusShort(context.Background(), workspace.CloneDir)
	if err != nil {
		t.Fatalf("StatusShort() returned error: %v", err)
	}
	if status != " M file.go\n" {
		t.Fatalf("status = %q", status)
	}

	if err := manager.CheckoutNewBranch(context.Background(), workspace.CloneDir, "feature/task"); err != nil {
		t.Fatalf("CheckoutNewBranch() returned error: %v", err)
	}
	if err := manager.CommitAllAndPush(context.Background(), workspace.CloneDir, "feature/task", "Commit message", true); err != nil {
		t.Fatalf("CommitAllAndPush() returned error: %v", err)
	}

	prURL, err := manager.CreatePullRequest(context.Background(), workspace.CloneDir, PullRequest{
		BaseBranch: "main",
		HeadBranch: "feature/task",
		Title:      "PR title",
		Body:       "PR body",
	})
	if err != nil {
		t.Fatalf("CreatePullRequest() returned error: %v", err)
	}
	if prURL != "https://github.com/example/repo/pull/7" {
		t.Fatalf("prURL = %q", prURL)
	}
	if err := manager.CommentPullRequest(context.Background(), workspace.CloneDir, prURL, "Final response"); err != nil {
		t.Fatalf("CommentPullRequest() returned error: %v", err)
	}

	assertCommand(t, fake.commands[0], "git", []string{"clone", "git@github.com:example/repo.git", "/tmp/orchv3-git/repo"}, "/tmp/orchv3-git")
	assertCommand(t, fake.commands[1], "git", []string{"status", "--short"}, "/tmp/orchv3-git/repo")
	assertCommand(t, fake.commands[2], "git", []string{"checkout", "-b", "feature/task"}, "/tmp/orchv3-git/repo")
	assertCommand(t, fake.commands[3], "git", []string{"add", "-A"}, "/tmp/orchv3-git/repo")
	assertCommand(t, fake.commands[4], "git", []string{"commit", "-m", "Commit message"}, "/tmp/orchv3-git/repo")
	assertCommand(t, fake.commands[5], "git", []string{"push", "-u", "origin", "feature/task"}, "/tmp/orchv3-git/repo")
	assertCommand(t, fake.commands[6], "gh", []string{"pr", "create", "--base", "main", "--head", "feature/task", "--title", "PR title", "--body", "PR body"}, "/tmp/orchv3-git/repo")
	assertCommand(t, fake.commands[7], "gh", []string{"pr", "comment", prURL, "--body", "Final response"}, "/tmp/orchv3-git/repo")

	if !strings.Contains(stdout.String(), "M file.go") {
		t.Fatalf("stdout logs = %q, want status output", stdout.String())
	}
}

func TestManagerResolveBranchAndCheckoutExisting(t *testing.T) {
	fake := &fakeCommandRunner{responses: []fakeResponse{{stdout: "feature/task\n"}, {}}}
	manager := newTestManager(fake, io.Discard)

	branch, err := manager.ResolvePullRequestBranch(context.Background(), "/tmp/repo", "https://github.com/example/repo/pull/7")
	if err != nil {
		t.Fatalf("ResolvePullRequestBranch() returned error: %v", err)
	}
	if branch != "feature/task" {
		t.Fatalf("branch = %q", branch)
	}
	if err := manager.Checkout(context.Background(), "/tmp/repo", branch); err != nil {
		t.Fatalf("Checkout() returned error: %v", err)
	}

	assertCommand(t, fake.commands[0], "gh", []string{"pr", "view", "https://github.com/example/repo/pull/7", "--json", "headRefName", "--jq", ".headRefName"}, "/tmp/repo")
	assertCommand(t, fake.commands[1], "git", []string{"checkout", "feature/task"}, "/tmp/repo")
}

func TestManagerGitLabCreateResolveAndComment(t *testing.T) {
	fake := &fakeCommandRunner{
		responses: []fakeResponse{
			{stdout: `{"url":"https://gitlab.com/example/repo/-/merge_requests/7"}`},
			{stdout: `{"source_branch":"feature/task"}`},
			{},
		},
	}
	manager := newTestManager(fake, io.Discard)
	manager.Config.GitProvider = config.GitProviderGitLab

	mrURL, err := manager.CreatePullRequest(context.Background(), "/tmp/repo", PullRequest{
		BaseBranch: "main",
		HeadBranch: "feature/task",
		Title:      "MR title",
		Body:       "MR body",
	})
	if err != nil {
		t.Fatalf("CreatePullRequest() returned error: %v", err)
	}
	if mrURL != "https://gitlab.com/example/repo/-/merge_requests/7" {
		t.Fatalf("mrURL = %q", mrURL)
	}

	branch, err := manager.ResolvePullRequestBranch(context.Background(), "/tmp/repo", mrURL)
	if err != nil {
		t.Fatalf("ResolvePullRequestBranch() returned error: %v", err)
	}
	if branch != "feature/task" {
		t.Fatalf("branch = %q", branch)
	}

	if err := manager.CommentPullRequest(context.Background(), "/tmp/repo", mrURL, "Final response"); err != nil {
		t.Fatalf("CommentPullRequest() returned error: %v", err)
	}

	assertCommand(t, fake.commands[0], "glab", []string{
		"mr", "create",
		"--source-branch", "feature/task",
		"--target-branch", "main",
		"--title", "MR title",
		"--description", "MR body",
		"--yes",
	}, "/tmp/repo")
	assertCommand(t, fake.commands[1], "glab", []string{"mr", "view", mrURL, "--output", "json"}, "/tmp/repo")
	assertCommand(t, fake.commands[2], "glab", []string{"mr", "note", "create", mrURL, "--message", "Final response"}, "/tmp/repo")
}

func TestManagerSkipsEmptyComment(t *testing.T) {
	fake := &fakeCommandRunner{}
	var stdout bytes.Buffer
	manager := newTestManager(fake, &stdout)

	if err := manager.CommentPullRequest(context.Background(), "/tmp/repo", "https://github.com/example/repo/pull/7", " \n\t "); err != nil {
		t.Fatalf("CommentPullRequest() returned error: %v", err)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("commands len = %d, want 0", len(fake.commands))
	}
	if !strings.Contains(stdout.String(), "skipped PR comment") {
		t.Fatalf("stdout = %q, want skip log", stdout.String())
	}
}

func TestManagerSkipsEmptyGitLabComment(t *testing.T) {
	fake := &fakeCommandRunner{}
	var stdout bytes.Buffer
	manager := newTestManager(fake, &stdout)
	manager.Config.GitProvider = config.GitProviderGitLab

	if err := manager.CommentPullRequest(context.Background(), "/tmp/repo", "https://gitlab.com/example/repo/-/merge_requests/7", " \n\t "); err != nil {
		t.Fatalf("CommentPullRequest() returned error: %v", err)
	}
	if len(fake.commands) != 0 {
		t.Fatalf("commands len = %d, want 0", len(fake.commands))
	}
	if !strings.Contains(stdout.String(), "skipped MR comment") {
		t.Fatalf("stdout = %q, want skip log", stdout.String())
	}
}

func TestManagerCleanupBehavior(t *testing.T) {
	manager := newTestManager(&fakeCommandRunner{}, io.Discard)
	removed := false
	if err := manager.Close(Workspace{TempDir: "/tmp/orchv3-git"}); err != nil {
		t.Fatalf("Close() preserve returned error: %v", err)
	}
	if removed {
		t.Fatal("remove called when cleanup disabled")
	}

	manager.Config.CleanupTemp = true
	manager.RemoveAll = func(path string) error {
		removed = true
		if path != "/tmp/orchv3-git" {
			t.Fatalf("remove path = %q", path)
		}
		return nil
	}
	if err := manager.Close(Workspace{TempDir: "/tmp/orchv3-git"}); err != nil {
		t.Fatalf("Close() cleanup returned error: %v", err)
	}
	if !removed {
		t.Fatal("remove was not called")
	}
}

func TestManagerWrapsFailures(t *testing.T) {
	errBoom := errors.New("boom")
	tests := []struct {
		name string
		run  func(*Manager) error
		want string
	}{
		{
			name: "clone",
			run:  func(manager *Manager) error { _, err := manager.Clone(context.Background()); return err },
			want: "git clone",
		},
		{
			name: "status",
			run: func(manager *Manager) error {
				_, err := manager.StatusShort(context.Background(), "/tmp/repo")
				return err
			},
			want: "git status",
		},
		{
			name: "checkout",
			run:  func(manager *Manager) error { return manager.Checkout(context.Background(), "/tmp/repo", "branch") },
			want: "git checkout branch",
		},
		{
			name: "resolve branch",
			run: func(manager *Manager) error {
				_, err := manager.ResolvePullRequestBranch(context.Background(), "/tmp/repo", "https://github.com/example/repo/pull/7")
				return err
			},
			want: "github resolve pr branch",
		},
		{
			name: "gitlab resolve branch",
			run: func(manager *Manager) error {
				manager.Config.GitProvider = config.GitProviderGitLab
				_, err := manager.ResolvePullRequestBranch(context.Background(), "/tmp/repo", "https://gitlab.com/example/repo/-/merge_requests/7")
				return err
			},
			want: "gitlab resolve mr branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := newTestManager(&fakeCommandRunner{responses: []fakeResponse{{err: errBoom}}}, io.Discard)
			err := tt.run(manager)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "boom") {
				t.Fatalf("error = %v, want %q and boom", err, tt.want)
			}
		})
	}
}

func TestManagerParsePRURL(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "plain", output: "https://github.com/example/repo/pull/1\n", want: "https://github.com/example/repo/pull/1"},
		{name: "json", output: `{"url":"https://github.com/example/repo/pull/2"}`, want: "https://github.com/example/repo/pull/2"},
		{name: "mixed", output: "Created\nhttps://github.com/example/repo/pull/3\n", want: "https://github.com/example/repo/pull/3"},
		{name: "gitlab plain", output: "https://gitlab.com/example/repo/-/merge_requests/4\n", want: "https://gitlab.com/example/repo/-/merge_requests/4"},
		{name: "gitlab mixed", output: "Creating merge request\nhttps://gitlab.com/example/repo/-/merge_requests/5\n", want: "https://gitlab.com/example/repo/-/merge_requests/5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePRURL(tt.output)
			if err != nil {
				t.Fatalf("ParsePRURL() returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParsePRURL() = %q, want %q", got, tt.want)
			}
		})
	}

	if _, err := ParsePRURL("no url here"); err == nil {
		t.Fatal("ParsePRURL() error = nil, want non-nil")
	}
}

func TestParseGitLabSourceBranch(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "snake case", output: `{"source_branch":"feature/task"}`, want: "feature/task"},
		{name: "camel case", output: `{"sourceBranch":"feature/camel"}`, want: "feature/camel"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitLabSourceBranch(tt.output)
			if err != nil {
				t.Fatalf("parseGitLabSourceBranch() returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("parseGitLabSourceBranch() = %q, want %q", got, tt.want)
			}
		})
	}

	if _, err := parseGitLabSourceBranch(`{"source_branch":""}`); err == nil {
		t.Fatal("parseGitLabSourceBranch() error = nil, want non-nil")
	}
}

func newTestManager(command commandrunner.Runner, stdout io.Writer) *Manager {
	cfg := config.ProposalRunnerConfig{
		RepositoryURL: "git@github.com:example/repo.git",
		BaseBranch:    "main",
		RemoteName:    "origin",
		GitPath:       "git",
		GHPath:        "gh",
		GLabPath:      "glab",
	}
	manager := NewFromProposalConfig(cfg)
	manager.Command = command
	manager.Stdout = stdout
	manager.Stderr = io.Discard
	manager.MkdirTemp = func(dir string, pattern string) (string, error) {
		if pattern != defaultTempPattern {
			return "", errors.New("unexpected temp pattern")
		}
		return "/tmp/orchv3-git", nil
	}
	manager.RemoveAll = func(path string) error {
		return errors.New("remove should not be called")
	}
	return manager
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
