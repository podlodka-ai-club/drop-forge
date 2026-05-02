package reviewrunner

import (
	"fmt"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

type StageProfile struct {
	Stage      agentmeta.Stage
	Categories map[string]struct{}
	PromptName string
}

var profiles = map[agentmeta.Stage]StageProfile{
	agentmeta.StageProposal: {
		Stage:      agentmeta.StageProposal,
		PromptName: "proposal_review.tmpl",
		Categories: setOf(
			"requirement_unclear",
			"requirement_contradicts_existing",
			"scenario_missing",
			"acceptance_criteria_weak",
			"scope_creep",
			"tasks_misaligned",
			"architecture_violation",
			"nit",
		),
	},
	agentmeta.StageApply: {
		Stage:      agentmeta.StageApply,
		PromptName: "apply_review.tmpl",
		Categories: setOf(
			"spec_mismatch",
			"bug",
			"error_handling",
			"concurrency",
			"test_gap",
			"architecture_violation",
			"idiom",
			"config_drift",
			"nit",
		),
	},
	agentmeta.StageArchive: {
		Stage:      agentmeta.StageArchive,
		PromptName: "archive_review.tmpl",
		Categories: setOf(
			"incomplete_archive",
			"spec_drift",
			"dangling_reference",
			"metadata_missing",
			"nit",
		),
	},
}

func ProfileFor(stage agentmeta.Stage) (StageProfile, error) {
	p, ok := profiles[stage]
	if !ok {
		return StageProfile{}, fmt.Errorf("no stage profile for %s", stage)
	}
	return p, nil
}

func MustProfile(stage agentmeta.Stage) StageProfile {
	p, err := ProfileFor(stage)
	if err != nil {
		panic(err)
	}
	return p
}

func ComputeVerdict(findings []reviewparse.Finding) reviewparse.Verdict {
	hasBlocker, hasMajor := false, false
	for _, f := range findings {
		switch f.Severity {
		case reviewparse.SeverityBlocker:
			hasBlocker = true
		case reviewparse.SeverityMajor:
			hasMajor = true
		}
	}
	switch {
	case hasBlocker:
		return reviewparse.VerdictBlocked
	case hasMajor:
		return reviewparse.VerdictNeedsWork
	default:
		return reviewparse.VerdictShipReady
	}
}

func SeverityIcon(s reviewparse.Severity) string {
	switch s {
	case reviewparse.SeverityBlocker:
		return "🛑"
	case reviewparse.SeverityMajor:
		return "⚠️"
	case reviewparse.SeverityMinor:
		return "💡"
	case reviewparse.SeverityNit:
		return "🪶"
	}
	return "•"
}

func setOf(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}
