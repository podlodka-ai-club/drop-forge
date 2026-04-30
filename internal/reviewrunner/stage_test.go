package reviewrunner

import (
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

func assertCategoryMembers(t *testing.T, profile StageProfile, expected []string) {
	t.Helper()
	for _, cat := range expected {
		if _, ok := profile.Categories[cat]; !ok {
			t.Errorf("expected category %q to be present in stage %s profile", cat, profile.Stage)
		}
	}
}

func TestStageProfileForProposalHasExpectedCategories(t *testing.T) {
	p, err := ProfileFor(agentmeta.StageProposal)
	if err != nil {
		t.Fatalf("ProfileFor(proposal) returned error: %v", err)
	}
	if p.Stage != agentmeta.StageProposal {
		t.Errorf("profile.Stage = %q, want %q", p.Stage, agentmeta.StageProposal)
	}
	if p.PromptName != "proposal_review.tmpl" {
		t.Errorf("profile.PromptName = %q, want %q", p.PromptName, "proposal_review.tmpl")
	}
	assertCategoryMembers(t, p, []string{
		"requirement_unclear",
		"scenario_missing",
		"scope_creep",
		"tasks_misaligned",
		"architecture_violation",
		"nit",
	})
}

func TestStageProfileForApplyHasExpectedCategories(t *testing.T) {
	p, err := ProfileFor(agentmeta.StageApply)
	if err != nil {
		t.Fatalf("ProfileFor(apply) returned error: %v", err)
	}
	if p.Stage != agentmeta.StageApply {
		t.Errorf("profile.Stage = %q, want %q", p.Stage, agentmeta.StageApply)
	}
	if p.PromptName != "apply_review.tmpl" {
		t.Errorf("profile.PromptName = %q, want %q", p.PromptName, "apply_review.tmpl")
	}
	assertCategoryMembers(t, p, []string{
		"spec_mismatch",
		"bug",
		"concurrency",
		"test_gap",
		"config_drift",
		"idiom",
	})
}

func TestStageProfileForArchiveHasExpectedCategories(t *testing.T) {
	p, err := ProfileFor(agentmeta.StageArchive)
	if err != nil {
		t.Fatalf("ProfileFor(archive) returned error: %v", err)
	}
	if p.Stage != agentmeta.StageArchive {
		t.Errorf("profile.Stage = %q, want %q", p.Stage, agentmeta.StageArchive)
	}
	if p.PromptName != "archive_review.tmpl" {
		t.Errorf("profile.PromptName = %q, want %q", p.PromptName, "archive_review.tmpl")
	}
	assertCategoryMembers(t, p, []string{
		"incomplete_archive",
		"spec_drift",
		"dangling_reference",
		"metadata_missing",
		"nit",
	})
}

func TestProfileForUnknownStageReturnsError(t *testing.T) {
	_, err := ProfileFor(agentmeta.Stage("bogus"))
	if err == nil {
		t.Fatal("expected error for unknown stage, got nil")
	}
}

func TestMustProfilePanicsForUnknownStage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected MustProfile to panic for unknown stage")
		}
	}()
	_ = MustProfile(agentmeta.Stage("bogus"))
}

func TestComputeVerdictBlockedWhenAnyBlocker(t *testing.T) {
	findings := []reviewparse.Finding{
		{Severity: reviewparse.SeverityNit},
		{Severity: reviewparse.SeverityMajor},
		{Severity: reviewparse.SeverityBlocker},
		{Severity: reviewparse.SeverityMinor},
	}
	if got := ComputeVerdict(findings); got != reviewparse.VerdictBlocked {
		t.Errorf("ComputeVerdict = %q, want %q", got, reviewparse.VerdictBlocked)
	}
}

func TestComputeVerdictNeedsWorkWhenMajorButNoBlocker(t *testing.T) {
	findings := []reviewparse.Finding{
		{Severity: reviewparse.SeverityNit},
		{Severity: reviewparse.SeverityMajor},
	}
	if got := ComputeVerdict(findings); got != reviewparse.VerdictNeedsWork {
		t.Errorf("ComputeVerdict = %q, want %q", got, reviewparse.VerdictNeedsWork)
	}
}

func TestComputeVerdictShipReadyWhenOnlyMinorAndNit(t *testing.T) {
	findings := []reviewparse.Finding{
		{Severity: reviewparse.SeverityMinor},
		{Severity: reviewparse.SeverityNit},
	}
	if got := ComputeVerdict(findings); got != reviewparse.VerdictShipReady {
		t.Errorf("ComputeVerdict = %q, want %q", got, reviewparse.VerdictShipReady)
	}
}

func TestComputeVerdictShipReadyForEmptyFindings(t *testing.T) {
	if got := ComputeVerdict(nil); got != reviewparse.VerdictShipReady {
		t.Errorf("ComputeVerdict(nil) = %q, want %q", got, reviewparse.VerdictShipReady)
	}
	if got := ComputeVerdict([]reviewparse.Finding{}); got != reviewparse.VerdictShipReady {
		t.Errorf("ComputeVerdict(empty) = %q, want %q", got, reviewparse.VerdictShipReady)
	}
}

func TestSeverityIconForEachLevel(t *testing.T) {
	cases := []struct {
		name     string
		severity reviewparse.Severity
		want     string
	}{
		{"blocker", reviewparse.SeverityBlocker, "🛑"},
		{"major", reviewparse.SeverityMajor, "⚠️"},
		{"minor", reviewparse.SeverityMinor, "💡"},
		{"nit", reviewparse.SeverityNit, "🪶"},
		{"unknown", reviewparse.Severity("mystery"), "•"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SeverityIcon(tc.severity); got != tc.want {
				t.Errorf("SeverityIcon(%q) = %q, want %q", tc.severity, got, tc.want)
			}
		})
	}
}
