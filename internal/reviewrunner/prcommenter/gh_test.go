package prcommenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/commandrunner"
	"orchv3/internal/reviewrunner/reviewparse"
)

type fakeResp struct {
	stdout string
	stderr string
	err    error
}

type fakeRunner struct {
	commands  []commandrunner.Command
	responses []fakeResp
	idx       int
}

func (f *fakeRunner) Run(_ context.Context, c commandrunner.Command) error {
	// Capture command (deep-copy stdin so tests can read it after Run returns).
	snapshot := c
	if c.Stdin != nil {
		b, _ := io.ReadAll(c.Stdin)
		snapshot.Stdin = bytes.NewReader(b)
	}
	f.commands = append(f.commands, snapshot)

	var r fakeResp
	if f.idx < len(f.responses) {
		r = f.responses[f.idx]
		f.idx++
	}
	if r.stdout != "" && c.Stdout != nil {
		_, _ = io.WriteString(c.Stdout, r.stdout)
	}
	if r.stderr != "" && c.Stderr != nil {
		_, _ = io.WriteString(c.Stderr, r.stderr)
	}
	return r.err
}

func newInput() PostReviewInput {
	lineStart := 10
	lineEnd := 14
	return PostReviewInput{
		RepoOwner:    "octo",
		RepoName:     "demo",
		PRNumber:     42,
		HeadSHA:      "abc123",
		Stage:        agentmeta.StageProposal,
		ReviewerSlot: "codex",
		Review: reviewparse.Review{
			Summary: reviewparse.Summary{
				Verdict:     reviewparse.VerdictNeedsWork,
				Walkthrough: "summary",
				Stats: reviewparse.Stats{
					Findings:   1,
					BySeverity: map[string]int{"major": 1},
				},
			},
			Findings: []reviewparse.Finding{
				{
					ID:        "F1",
					Category:  "scenario_missing",
					Severity:  reviewparse.SeverityMajor,
					File:      "openspec/x.md",
					LineStart: &lineStart,
					LineEnd:   &lineEnd,
					Title:     "Missing scenario",
					Message:   "scenario is missing",
					FixPrompt: "do this",
				},
			},
		},
	}
}

func TestPostReviewSkipsWhenMarkerAlreadyPresent(t *testing.T) {
	in := newInput()
	marker := MarkerFor(in.ReviewerSlot, in.Stage, in.HeadSHA)
	existing, err := json.Marshal([]ghReview{{Body: "already posted\n" + marker + "\nrest"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	fr := &fakeRunner{responses: []fakeResp{{stdout: string(existing)}}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	res, err := c.PostReview(context.Background(), in)
	if err != nil {
		t.Fatalf("PostReview: %v", err)
	}
	if !res.Skipped {
		t.Errorf("expected Skipped=true, got %+v", res)
	}
	if len(fr.commands) != 1 {
		t.Errorf("expected exactly 1 command (GET only), got %d", len(fr.commands))
	}
}

func TestPostReviewSendsAtomicPOSTWithSummaryAndInlineComments(t *testing.T) {
	in := newInput()
	fr := &fakeRunner{responses: []fakeResp{
		{stdout: "[]"},
		{},
	}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	res, err := c.PostReview(context.Background(), in)
	if err != nil {
		t.Fatalf("PostReview: %v", err)
	}
	if res.Skipped {
		t.Errorf("expected Skipped=false, got %+v", res)
	}
	if len(fr.commands) != 2 {
		t.Fatalf("expected 2 commands (GET, POST), got %d", len(fr.commands))
	}

	post := fr.commands[1]
	if post.Stdin == nil {
		t.Fatalf("POST command stdin must not be nil")
	}
	raw, err := io.ReadAll(post.Stdin)
	if err != nil {
		t.Fatalf("read post stdin: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v\n%s", err, raw)
	}
	if payload["event"] != "COMMENT" {
		t.Errorf("expected event=COMMENT, got %v", payload["event"])
	}
	if payload["commit_id"] != "abc123" {
		t.Errorf("expected commit_id=abc123, got %v", payload["commit_id"])
	}
	body, _ := payload["body"].(string)
	marker := MarkerFor(in.ReviewerSlot, in.Stage, in.HeadSHA)
	if !strings.Contains(body, marker) {
		t.Errorf("expected summary body to contain marker, body:\n%s", body)
	}
	comments, ok := payload["comments"].([]interface{})
	if !ok {
		t.Fatalf("expected comments array, got %T", payload["comments"])
	}
	if len(comments) != 1 {
		t.Errorf("expected 1 inline comment, got %d", len(comments))
	}
}

func TestPostReviewSkipsLineNullFindings(t *testing.T) {
	in := newInput()
	lineStart := 5
	in.Review.Findings = []reviewparse.Finding{
		{
			ID:       "G1",
			Category: "scenario_missing",
			Severity: reviewparse.SeverityMinor,
			File:     "openspec/g.md",
			Title:    "general",
			Message:  "general note",
		},
		{
			ID:        "F2",
			Category:  "scenario_missing",
			Severity:  reviewparse.SeverityMajor,
			File:      "openspec/x.md",
			LineStart: &lineStart,
			LineEnd:   &lineStart,
			Title:     "single line",
			Message:   "msg",
		},
	}
	fr := &fakeRunner{responses: []fakeResp{
		{stdout: "[]"},
		{},
	}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	if _, err := c.PostReview(context.Background(), in); err != nil {
		t.Fatalf("PostReview: %v", err)
	}

	raw, _ := io.ReadAll(fr.commands[1].Stdin)
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	comments := payload["comments"].([]interface{})
	if len(comments) != 1 {
		t.Errorf("expected 1 inline comment (line-null skipped), got %d", len(comments))
	}
}

func TestPostReviewWrapsCommandErrorOnPost(t *testing.T) {
	in := newInput()
	fr := &fakeRunner{responses: []fakeResp{
		{stdout: "[]"},
		{err: errors.New("boom")},
	}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	_, err := c.PostReview(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "gh api POST review") {
		t.Errorf("expected error to mention 'gh api POST review', got %v", err)
	}
}

func TestPostReviewWrapsCommandErrorOnGet(t *testing.T) {
	in := newInput()
	fr := &fakeRunner{responses: []fakeResp{
		{err: errors.New("net down")},
	}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	_, err := c.PostReview(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "check existing review marker") {
		t.Errorf("expected error to mention 'check existing review marker', got %v", err)
	}
}

func TestPostReviewFallsBackToSummaryOnlyWhen422(t *testing.T) {
	in := newInput()
	fr := &fakeRunner{responses: []fakeResp{
		{stdout: "[]"}, // GET reviews → no existing marker
		{ // first POST: GitHub rejects inline anchors
			stderr: `{"message":"Unprocessable Entity","errors":["Path could not be resolved"],"documentation_url":"https://docs.github.com/rest/pulls/reviews#create-a-review-for-a-pull-request","status":"422"}` + "\n" + "gh: Unprocessable Entity (HTTP 422)",
			err:    errors.New("exit status 1"),
		},
		{}, // second POST (summary-only): success
	}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	res, err := c.PostReview(context.Background(), in)
	if err != nil {
		t.Fatalf("PostReview() returned error: %v", err)
	}
	if res.Skipped {
		t.Fatalf("Skipped = true, want false (review was published via fallback)")
	}
	if len(fr.commands) != 3 {
		t.Fatalf("commands len = %d, want 3 (GET + first POST + fallback POST)", len(fr.commands))
	}

	// Inspect the fallback POST payload.
	fallback := fr.commands[2]
	if fallback.Stdin == nil {
		t.Fatal("fallback POST has no stdin payload")
	}
	body, err := io.ReadAll(fallback.Stdin)
	if err != nil {
		t.Fatalf("read fallback stdin: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode fallback payload: %v", err)
	}
	comments, _ := payload["comments"].([]interface{})
	if len(comments) != 0 {
		t.Fatalf("fallback comments len = %d, want 0", len(comments))
	}
	bodyStr, _ := payload["body"].(string)
	if !strings.Contains(bodyStr, "Inline review comments were rejected") {
		t.Fatalf("fallback summary missing tripwire:\n%s", bodyStr)
	}
}

func TestPostReviewSurfacesNon422POSTErrors(t *testing.T) {
	in := newInput()
	fr := &fakeRunner{responses: []fakeResp{
		{stdout: "[]"},
		{
			stderr: "gh: Internal Server Error (HTTP 500)",
			err:    errors.New("exit status 1"),
		},
	}}
	c := GHPostReviewCommenter{Command: fr, GHPath: "gh"}

	_, err := c.PostReview(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "gh api POST review") {
		t.Fatalf("expected wrapped POST error, got %v", err)
	}
	if len(fr.commands) != 2 {
		t.Fatalf("commands len = %d, want 2 (no fallback for non-422)", len(fr.commands))
	}
}
