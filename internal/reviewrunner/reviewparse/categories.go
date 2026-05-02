package reviewparse

import (
	"fmt"

	"orchv3/internal/agentmeta"
)

var stageCategories = map[agentmeta.Stage]map[string]struct{}{
	agentmeta.StageProposal: setOf(
		"requirement_unclear",
		"requirement_contradicts_existing",
		"scenario_missing",
		"acceptance_criteria_weak",
		"scope_creep",
		"tasks_misaligned",
		"architecture_violation",
		"nit",
	),
	agentmeta.StageApply: setOf(
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
	agentmeta.StageArchive: setOf(
		"incomplete_archive",
		"spec_drift",
		"dangling_reference",
		"metadata_missing",
		"nit",
	),
}

func CategoriesForStage(stage agentmeta.Stage) (map[string]struct{}, error) {
	cats, ok := stageCategories[stage]
	if !ok {
		return nil, fmt.Errorf("no categories registered for stage %s", stage)
	}
	return cats, nil
}

func setOf(values ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(values))
	for _, v := range values {
		m[v] = struct{}{}
	}
	return m
}
