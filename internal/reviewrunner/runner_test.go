package reviewrunner

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/reviewrunner/prcommenter"
)

const validReviewJSON = `{
  "summary": {"verdict":"ship-ready","walkthrough":"ok","stats":{"findings":0,"by_severity":{"blocker":0,"major":0,"minor":0,"nit":0}}},
  "findings": []
}`

const validReviewWithFindingJSON = `{
  "summary": {"verdict":"needs-work","walkthrough":"ok","stats":{"findings":1,"by_severity":{"blocker":0,"major":1,"minor":0,"nit":0}}},
  "findings": [{
    "id":"F1","category":"requirement_unclear","severity":"major",
    "file":"openspec/changes/x/proposal.md","line_start":null,"line_end":null,
    "title":"t","message":"m","fix_prompt":"fp"
  }]
}`

type fakePlan struct {
	stdout string
	stderr string
	err    error
}

type fakeCmd struct {
	commands []commandrunner.Command
	plans    []fakePlan
	idx      int
}

func (f *fakeCmd) Run(_ context.Context, c commandrunner.Command) error {
	f.commands = append(f.commands, c)
	if f.idx >= len(f.plans) {
		return nil
	}
	p := f.plans[f.idx]
	f.idx++
	if p.stdout != "" && c.Stdout != nil {
		_, _ = io.WriteString(c.Stdout, p.stdout)
	}
	if p.stderr != "" && c.Stderr != nil {
		_, _ = io.WriteString(c.Stderr, p.stderr)
	}
	return p.err
}

type fakeExec struct {
	responses []string
	calls     int
	err       error
	lastIn    AgentExecutionInput
	inputs    []AgentExecutionInput
}

func (f *fakeExec) Run(_ context.Context, in AgentExecutionInput) (AgentExecutionResult, error) {
	f.lastIn = in
	f.inputs = append(f.inputs, in)
	if f.err != nil {
		return AgentExecutionResult{}, f.err
	}
	if f.calls >= len(f.responses) {
		return AgentExecutionResult{}, errors.New("no more fake responses")
	}
	r := f.responses[f.calls]
	f.calls++
	return AgentExecutionResult{FinalMessage: r}, nil
}

type fakeCommenter struct {
	called bool
	in     prcommenter.PostReviewInput
	res    prcommenter.PostReviewResult
	err    error
}

func (f *fakeCommenter) PostReview(_ context.Context, in prcommenter.PostReviewInput) (prcommenter.PostReviewResult, error) {
	f.called = true
	f.in = in
	return f.res, f.err
}

func defaultReviewCfg() config.ReviewRunnerConfig {
	return config.ReviewRunnerConfig{
		PrimarySlot:           "codex",
		SecondarySlot:         "claude",
		PrimaryModel:          "gpt-5",
		SecondaryModel:        "claude-3",
		PrimaryExecutorPath:   "/p",
		SecondaryExecutorPath: "/c",
		MaxContextBytes:       1 << 16,
		ParseRepairRetries:    1,
	}
}

func defaultProposalCfg() config.ProposalRunnerConfig {
	return config.ProposalRunnerConfig{
		RepositoryURL: "https://example/repo",
		BaseBranch:    "main",
		RemoteName:    "origin",
		BranchPrefix:  "p",
		PRTitlePrefix: "P:",
		GitPath:       "git",
		CodexPath:     "codex",
		GHPath:        "gh",
		CleanupTemp:   false, // tests use t.TempDir cleanup
	}
}

func defaultInput() ReviewInput {
	return ReviewInput{
		Stage:      agentmeta.StageProposal,
		Identifier: "ZIM-1",
		Title:      "T",
		BranchName: "feature/x",
		PRNumber:   1,
		RepoOwner:  "o",
		RepoName:   "p",
		PRURL:      "https://example/pr/1",
	}
}

// makeMkdirTemp returns a MkdirTemp implementation that creates a fresh
// "review" subdirectory inside t.TempDir() and seeds it with a minimal
// openspec/changes/x change so that target collection succeeds (proposal stage
// requires a non-empty ChangePath; default fakeCmd returns "" for diff and
// detectChangePath, so we leave change detection empty and rely on stage Apply
// or seed change dir manually when needed).
func makeMkdirTemp(t *testing.T, seedChange bool) func(string, string) (string, error) {
	t.Helper()
	root := t.TempDir()
	return func(_, _ string) (string, error) {
		dir := filepath.Join(root, "review")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		if seedChange {
			changeDir := filepath.Join(dir, "repo", "openspec", "changes", "x")
			if err := os.MkdirAll(changeDir, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte("# proposal\n"), 0o644); err != nil {
				return "", err
			}
		}
		return dir, nil
	}
}

// happyPathPlans returns plans for: clone, rev-parse, log -1, diff --name-only, diff.
// commitMessage controls the git log -1 stdout (which carries the trailer).
// nameOnly controls git diff --name-only stdout (used to detect change path).
func happyPathPlans(commitMessage, nameOnly string) []fakePlan {
	return []fakePlan{
		{},                            // git clone
		{stdout: "deadbeef\n"},        // git rev-parse HEAD
		{stdout: commitMessage},       // git log -1 --format=%B
		{stdout: nameOnly},            // git diff --name-only main...HEAD
		{stdout: "some diff content"}, // git diff main...HEAD
	}
}

func TestRunHappyPathPublishesReviewAndReturnsResult(t *testing.T) {
	cmd := &fakeCmd{plans: happyPathPlans(
		"subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: proposal\n",
		"openspec/changes/x/proposal.md\n",
	)}
	exec := &fakeExec{responses: []string{validReviewWithFindingJSON}}
	commenter := &fakeCommenter{}

	r := &Runner{
		Config:      defaultReviewCfg(),
		ProposalCfg: defaultProposalCfg(),
		Command:     cmd,
		Executors:   map[string]AgentExecutor{"claude": exec},
		Commenter:   commenter,
		MkdirTemp:   makeMkdirTemp(t, true),
		RemoveAll:   os.RemoveAll,
	}

	res, err := r.Run(context.Background(), defaultInput())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Skipped {
		t.Fatalf("expected Skipped=false, got true")
	}
	if !commenter.called {
		t.Fatalf("expected commenter.PostReview to be called")
	}
	if commenter.in.HeadSHA != "deadbeef" {
		t.Fatalf("expected HeadSHA=deadbeef, got %q", commenter.in.HeadSHA)
	}
	if commenter.in.ReviewerSlot != "claude" {
		t.Fatalf("expected ReviewerSlot=claude (opposite of producer codex), got %q", commenter.in.ReviewerSlot)
	}
	if commenter.in.Stage != agentmeta.StageProposal {
		t.Fatalf("expected stage proposal, got %s", commenter.in.Stage)
	}
	if exec.calls != 1 {
		t.Fatalf("expected 1 executor call on happy path, got %d", exec.calls)
	}
	if len(commenter.in.Review.Findings) != 1 {
		t.Fatalf("expected one finding propagated, got %d", len(commenter.in.Review.Findings))
	}
	// no producer-unknown tripwire on happy path
	for _, e := range commenter.in.WalkthroughExtras {
		if strings.Contains(e, "Producer trailer absent") {
			t.Fatalf("did not expect producer-absent tripwire, got %q", e)
		}
	}
}

func TestRunMissingTrailerFallsBackAndAddsTripwire(t *testing.T) {
	cmd := &fakeCmd{plans: happyPathPlans(
		"subject only\n", // no trailer
		"openspec/changes/x/proposal.md\n",
	)}
	exec := &fakeExec{responses: []string{validReviewJSON}}
	commenter := &fakeCommenter{}

	r := &Runner{
		Config:      defaultReviewCfg(),
		ProposalCfg: defaultProposalCfg(),
		Command:     cmd,
		Executors:   map[string]AgentExecutor{"claude": exec},
		Commenter:   commenter,
		MkdirTemp:   makeMkdirTemp(t, true),
		RemoveAll:   os.RemoveAll,
	}

	res, err := r.Run(context.Background(), defaultInput())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.Skipped {
		t.Fatalf("expected Skipped=false")
	}
	if !commenter.called {
		t.Fatalf("expected commenter to be called")
	}
	if commenter.in.ReviewerSlot != "claude" {
		t.Fatalf("expected fallback to secondary slot 'claude', got %q", commenter.in.ReviewerSlot)
	}
	found := false
	for _, e := range commenter.in.WalkthroughExtras {
		if strings.Contains(e, "Producer trailer absent") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected WalkthroughExtras to contain 'Producer trailer absent', got %v", commenter.in.WalkthroughExtras)
	}
}

func TestRunInvalidJSONTriggersExactlyOneRepairAndSucceeds(t *testing.T) {
	cmd := &fakeCmd{plans: happyPathPlans(
		"subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: proposal\n",
		"openspec/changes/x/proposal.md\n",
	)}
	exec := &fakeExec{responses: []string{"not-json", validReviewJSON}}
	commenter := &fakeCommenter{}

	r := &Runner{
		Config:      defaultReviewCfg(),
		ProposalCfg: defaultProposalCfg(),
		Command:     cmd,
		Executors:   map[string]AgentExecutor{"claude": exec},
		Commenter:   commenter,
		MkdirTemp:   makeMkdirTemp(t, true),
		RemoveAll:   os.RemoveAll,
	}

	if _, err := r.Run(context.Background(), defaultInput()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if exec.calls != 2 {
		t.Fatalf("expected exactly 2 executor calls (initial + repair), got %d", exec.calls)
	}
	if !commenter.called {
		t.Fatalf("expected commenter to be called after successful repair")
	}
	if !strings.Contains(exec.inputs[1].Prompt, "невалиден") {
		t.Fatalf("expected repair prompt to mention previous error, got: %q", exec.inputs[1].Prompt)
	}
}

func TestRunSecondInvalidJSONReturnsErrorAndDoesNotCallCommenter(t *testing.T) {
	cmd := &fakeCmd{plans: happyPathPlans(
		"subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: proposal\n",
		"openspec/changes/x/proposal.md\n",
	)}
	exec := &fakeExec{responses: []string{"not-json", "still-not-json"}}
	commenter := &fakeCommenter{}

	r := &Runner{
		Config:      defaultReviewCfg(),
		ProposalCfg: defaultProposalCfg(),
		Command:     cmd,
		Executors:   map[string]AgentExecutor{"claude": exec},
		Commenter:   commenter,
		MkdirTemp:   makeMkdirTemp(t, true),
		RemoveAll:   os.RemoveAll,
	}

	_, err := r.Run(context.Background(), defaultInput())
	if err == nil {
		t.Fatalf("expected error after two invalid JSON responses, got nil")
	}
	if !strings.Contains(err.Error(), "parse review JSON after repair") {
		t.Fatalf("expected error containing 'parse review JSON after repair', got %v", err)
	}
	if commenter.called {
		t.Fatalf("commenter must not be called when both attempts fail to parse")
	}
	if exec.calls != 2 {
		t.Fatalf("expected exactly 2 executor calls, got %d", exec.calls)
	}
}

func TestRunReportsSkippedWhenCommenterReportsSkip(t *testing.T) {
	cmd := &fakeCmd{plans: happyPathPlans(
		"subject\n\nProduced-By: codex\nProduced-Model: gpt-5\nProduced-Stage: proposal\n",
		"openspec/changes/x/proposal.md\n",
	)}
	exec := &fakeExec{responses: []string{validReviewJSON}}
	commenter := &fakeCommenter{res: prcommenter.PostReviewResult{Skipped: true}}

	r := &Runner{
		Config:      defaultReviewCfg(),
		ProposalCfg: defaultProposalCfg(),
		Command:     cmd,
		Executors:   map[string]AgentExecutor{"claude": exec},
		Commenter:   commenter,
		MkdirTemp:   makeMkdirTemp(t, true),
		RemoveAll:   os.RemoveAll,
	}

	res, err := r.Run(context.Background(), defaultInput())
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !res.Skipped {
		t.Fatalf("expected Skipped=true, got false")
	}
	if !commenter.called {
		t.Fatalf("expected commenter to be called")
	}
}
