package reviewparse

import (
	"strings"
	"testing"

	"orchv3/internal/agentmeta"
)

const validProposalReview = `{
  "summary": {
    "verdict": "needs-work",
    "walkthrough": "Reviewed proposal artefacts; one major scenario gap identified.",
    "stats": {
      "findings": 1,
      "by_severity": {"blocker": 0, "major": 1, "minor": 0, "nit": 0}
    }
  },
  "findings": [
    {
      "id": "f1",
      "category": "scenario_missing",
      "severity": "major",
      "file": "openspec/changes/add-x/specs/foo/spec.md",
      "line_start": 10,
      "line_end": 14,
      "title": "Missing edge-case scenario",
      "message": "No scenario covers the empty-input case.",
      "fix_prompt": "Add a scenario for empty input."
    }
  ]
}`

func TestParseValidProposalReview(t *testing.T) {
	r, err := Parse([]byte(validProposalReview), agentmeta.StageProposal)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if r.Summary.Verdict != VerdictNeedsWork {
		t.Errorf("verdict = %q, want %q", r.Summary.Verdict, VerdictNeedsWork)
	}
	if r.Summary.Stats.Findings != 1 {
		t.Errorf("stats.findings = %d, want 1", r.Summary.Stats.Findings)
	}
	if r.Summary.Stats.BySeverity["major"] != 1 {
		t.Errorf("stats.by_severity[major] = %d, want 1", r.Summary.Stats.BySeverity["major"])
	}
	if len(r.Findings) != 1 {
		t.Fatalf("findings len = %d, want 1", len(r.Findings))
	}
	f := r.Findings[0]
	if f.Category != "scenario_missing" {
		t.Errorf("category = %q, want scenario_missing", f.Category)
	}
	if f.Severity != SeverityMajor {
		t.Errorf("severity = %q, want major", f.Severity)
	}
	if f.LineStart == nil || *f.LineStart != 10 {
		t.Errorf("line_start = %v, want *10", f.LineStart)
	}
	if f.LineEnd == nil || *f.LineEnd != 14 {
		t.Errorf("line_end = %v, want *14", f.LineEnd)
	}
}

func TestParseRejectsUnknownVerdict(t *testing.T) {
	raw := `{
  "summary": {"verdict": "awesome", "walkthrough": "x", "stats": {"findings": 0, "by_severity": {}}},
  "findings": []
}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil {
		t.Fatal("expected error for unknown verdict")
	}
	if !strings.Contains(err.Error(), "verdict") {
		t.Errorf("error %q must contain \"verdict\"", err.Error())
	}
}

func TestParseRejectsUnknownSeverity(t *testing.T) {
	raw := `{
  "summary": {"verdict": "ship-ready", "walkthrough": "x", "stats": {"findings": 1, "by_severity": {}}},
  "findings": [
    {
      "id": "f1",
      "category": "scenario_missing",
      "severity": "critical",
      "file": "a.md",
      "line_start": null,
      "line_end": null,
      "title": "t",
      "message": "m",
      "fix_prompt": "p"
    }
  ]
}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil {
		t.Fatal("expected error for unknown severity")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Errorf("error %q must contain \"severity\"", err.Error())
	}
}

func TestParseRejectsCategoryNotInStageEnum(t *testing.T) {
	// "bug" is an Apply category; reject under Proposal stage.
	raw := `{
  "summary": {"verdict": "needs-work", "walkthrough": "x", "stats": {"findings": 1, "by_severity": {"major": 1}}},
  "findings": [
    {
      "id": "f1",
      "category": "bug",
      "severity": "major",
      "file": "a.md",
      "line_start": null,
      "line_end": null,
      "title": "t",
      "message": "m",
      "fix_prompt": "p"
    }
  ]
}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil {
		t.Fatal("expected error for out-of-stage category")
	}
	msg := err.Error()
	if !strings.Contains(msg, "category") {
		t.Errorf("error %q must contain \"category\"", msg)
	}
	if !strings.Contains(msg, "stage") {
		t.Errorf("error %q must contain \"stage\"", msg)
	}
}

func TestParseAllowsNullLineRangeForGeneralFindings(t *testing.T) {
	raw := `{
  "summary": {"verdict": "ship-ready", "walkthrough": "x", "stats": {"findings": 1, "by_severity": {"nit": 1}}},
  "findings": [
    {
      "id": "f1",
      "category": "nit",
      "severity": "nit",
      "file": "a.md",
      "line_start": null,
      "line_end": null,
      "title": "t",
      "message": "m",
      "fix_prompt": "p"
    }
  ]
}`
	r, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(r.Findings) != 1 {
		t.Fatalf("findings len = %d, want 1", len(r.Findings))
	}
	if r.Findings[0].LineStart != nil {
		t.Errorf("line_start = %v, want nil", r.Findings[0].LineStart)
	}
	if r.Findings[0].LineEnd != nil {
		t.Errorf("line_end = %v, want nil", r.Findings[0].LineEnd)
	}
}

func TestParseRejectsMismatchedLineRange(t *testing.T) {
	raw := `{
  "summary": {"verdict": "needs-work", "walkthrough": "x", "stats": {"findings": 1, "by_severity": {"minor": 1}}},
  "findings": [
    {
      "id": "f1",
      "category": "nit",
      "severity": "minor",
      "file": "a.md",
      "line_start": 5,
      "line_end": null,
      "title": "t",
      "message": "m",
      "fix_prompt": "p"
    }
  ]
}`
	_, err := Parse([]byte(raw), agentmeta.StageProposal)
	if err == nil {
		t.Fatal("expected error for mismatched line range")
	}
	msg := err.Error()
	if !strings.Contains(msg, "line_start") || !strings.Contains(msg, "line_end") {
		t.Errorf("error %q must mention both line_start and line_end", msg)
	}
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	_, err := Parse([]byte("not-json"), agentmeta.StageProposal)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseRejectsUnknownStage(t *testing.T) {
	_, err := Parse([]byte(validProposalReview), agentmeta.Stage("bogus"))
	if err == nil {
		t.Fatal("expected error for unknown stage")
	}
	if !strings.Contains(err.Error(), "stage") {
		t.Errorf("error %q must contain \"stage\"", err.Error())
	}
}
