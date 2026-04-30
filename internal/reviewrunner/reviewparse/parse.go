package reviewparse

import (
	"encoding/json"
	"fmt"

	"orchv3/internal/agentmeta"
)

type Verdict string

const (
	VerdictShipReady Verdict = "ship-ready"
	VerdictNeedsWork Verdict = "needs-work"
	VerdictBlocked   Verdict = "blocked"
)

type Severity string

const (
	SeverityBlocker Severity = "blocker"
	SeverityMajor   Severity = "major"
	SeverityMinor   Severity = "minor"
	SeverityNit     Severity = "nit"
)

type Stats struct {
	Findings   int            `json:"findings"`
	BySeverity map[string]int `json:"by_severity"`
}

type Summary struct {
	Verdict     Verdict `json:"verdict"`
	Walkthrough string  `json:"walkthrough"`
	Stats       Stats   `json:"stats"`
}

type Finding struct {
	ID        string   `json:"id"`
	Category  string   `json:"category"`
	Severity  Severity `json:"severity"`
	File      string   `json:"file"`
	LineStart *int     `json:"line_start"`
	LineEnd   *int     `json:"line_end"`
	Title     string   `json:"title"`
	Message   string   `json:"message"`
	FixPrompt string   `json:"fix_prompt"`
}

type Review struct {
	Summary  Summary   `json:"summary"`
	Findings []Finding `json:"findings"`
}

var validVerdicts = map[Verdict]struct{}{
	VerdictShipReady: {},
	VerdictNeedsWork: {},
	VerdictBlocked:   {},
}

var validSeverities = map[Severity]struct{}{
	SeverityBlocker: {},
	SeverityMajor:   {},
	SeverityMinor:   {},
	SeverityNit:     {},
}

// Parse decodes a JSON review response and validates it against the closed
// schema for the given stage. Returns the parsed Review on success or a
// contextual error on any validation failure.
func Parse(raw []byte, stage agentmeta.Stage) (Review, error) {
	var r Review
	if err := json.Unmarshal(raw, &r); err != nil {
		return Review{}, fmt.Errorf("decode review JSON: %w", err)
	}
	if _, ok := validVerdicts[r.Summary.Verdict]; !ok {
		return Review{}, fmt.Errorf("invalid summary.verdict %q", r.Summary.Verdict)
	}

	allowedCategories, err := CategoriesForStage(stage)
	if err != nil {
		return Review{}, err
	}

	for i, f := range r.Findings {
		if _, ok := validSeverities[f.Severity]; !ok {
			return Review{}, fmt.Errorf("findings[%d].severity %q is not in {blocker, major, minor, nit}", i, f.Severity)
		}
		if _, ok := allowedCategories[f.Category]; !ok {
			return Review{}, fmt.Errorf("findings[%d].category %q not allowed for stage %s", i, f.Category, stage)
		}
		if (f.LineStart == nil) != (f.LineEnd == nil) {
			return Review{}, fmt.Errorf("findings[%d]: line_start and line_end must both be set or both null", i)
		}
	}
	return r, nil
}
