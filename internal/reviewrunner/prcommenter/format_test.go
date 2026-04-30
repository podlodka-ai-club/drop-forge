package prcommenter

import (
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

func intPtr(v int) *int { return &v }

func sampleInput() PostReviewInput {
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
				Walkthrough: "high level look",
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
					Message:   "scenario is missing for this requirement",
					FixPrompt: "do this",
				},
			},
		},
	}
}

func TestFormatSummaryBodyIncludesMarkerVerdictAndFindings(t *testing.T) {
	in := sampleInput()
	body := FormatSummaryBody(in)

	want := []string{
		MarkerFor("codex", agentmeta.StageProposal, "abc123"),
		"Review by codex",
		"needs-work",
		"F1",
		"scenario_missing",
		"openspec/x.md:10-14",
	}
	for _, w := range want {
		if !strings.Contains(body, w) {
			t.Errorf("summary body missing %q.\nbody:\n%s", w, body)
		}
	}
	if !strings.HasPrefix(body, MarkerFor("codex", agentmeta.StageProposal, "abc123")) {
		t.Errorf("summary body should start with marker; got:\n%s", body)
	}
}

func TestFormatSummaryBodyIncludesTripwiresWhenPresent(t *testing.T) {
	in := sampleInput()
	in.WalkthroughExtras = []string{"Producer trailer absent"}
	body := FormatSummaryBody(in)

	if !strings.Contains(body, "### Tripwires") {
		t.Errorf("expected '### Tripwires' section in body:\n%s", body)
	}
	if !strings.Contains(body, "Producer trailer absent") {
		t.Errorf("expected tripwire line in body:\n%s", body)
	}
}

func TestFormatSummaryBodyOmitsFindingsSectionWhenEmpty(t *testing.T) {
	in := sampleInput()
	in.Review.Findings = nil
	body := FormatSummaryBody(in)

	if strings.Contains(body, "### Findings") {
		t.Errorf("expected no '### Findings' section, body:\n%s", body)
	}
}

func TestFormatInlineBodyHasFixPromptDetailsBlock(t *testing.T) {
	in := sampleInput()
	body := FormatInlineBody(in.ReviewerSlot, in.Review.Findings[0])

	if !strings.Contains(body, "🤖 Prompt for AI Agent") {
		t.Errorf("expected fix prompt details summary, body:\n%s", body)
	}
	if !strings.Contains(body, "do this") {
		t.Errorf("expected fix prompt text, body:\n%s", body)
	}
	if !strings.Contains(body, "<details>") || !strings.Contains(body, "</details>") {
		t.Errorf("expected <details> block, body:\n%s", body)
	}
}

func TestFormatInlineBodyHasReviewByPrefix(t *testing.T) {
	f := reviewparse.Finding{
		ID:       "B1",
		Category: "bug",
		Severity: reviewparse.SeverityBlocker,
		File:     "x.go",
		Message:  "boom",
	}
	body := FormatInlineBody("codex", f)

	if !strings.Contains(body, "[review by codex · severity: blocker · category: bug]") {
		t.Errorf("expected '[review by ...]' prefix in body:\n%s", body)
	}
}

func TestFormatLineRefForGeneralFinding(t *testing.T) {
	f := reviewparse.Finding{File: "openspec/x.md"}
	got := formatLineRef(f)
	if got != "openspec/x.md" {
		t.Errorf("expected file-only ref, got %q", got)
	}
}

func TestFormatLineRefForSingleLine(t *testing.T) {
	f := reviewparse.Finding{
		File:      "x.go",
		LineStart: intPtr(5),
		LineEnd:   intPtr(5),
	}
	got := formatLineRef(f)
	if got != "x.go:5" {
		t.Errorf("expected x.go:5, got %q", got)
	}
}

func TestFormatLineRefForRange(t *testing.T) {
	f := reviewparse.Finding{
		File:      "x.go",
		LineStart: intPtr(10),
		LineEnd:   intPtr(14),
	}
	got := formatLineRef(f)
	if got != "x.go:10-14" {
		t.Errorf("expected x.go:10-14, got %q", got)
	}
}
