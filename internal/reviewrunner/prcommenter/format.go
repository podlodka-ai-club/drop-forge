package prcommenter

import (
	"fmt"
	"strings"

	"orchv3/internal/reviewrunner/reviewparse"
)

// FormatSummaryBody returns the markdown body of the PR review summary.
// The first line is the idempotency marker; the body contains verdict, stats,
// walkthrough, findings list, and tripwires (if any).
func FormatSummaryBody(in PostReviewInput) string {
	var b strings.Builder
	b.WriteString(MarkerFor(in.ReviewerSlot, in.Stage, in.HeadSHA))
	b.WriteString("\n\n## 🤖 Review by ")
	b.WriteString(in.ReviewerSlot)
	b.WriteString(" (stage: ")
	b.WriteString(string(in.Stage))
	b.WriteString(")\n\n")

	stats := in.Review.Summary.Stats
	fmt.Fprintf(&b, "**Verdict:** %s · **Findings:** %d (🛑 %d · ⚠️ %d · 💡 %d · 🪶 %d)\n\n",
		in.Review.Summary.Verdict,
		stats.Findings,
		stats.BySeverity["blocker"],
		stats.BySeverity["major"],
		stats.BySeverity["minor"],
		stats.BySeverity["nit"],
	)
	b.WriteString("### Walkthrough\n")
	b.WriteString(in.Review.Summary.Walkthrough)
	b.WriteString("\n\n")

	if len(in.Review.Findings) > 0 {
		b.WriteString("### Findings\n")
		for _, f := range in.Review.Findings {
			fmt.Fprintf(&b, "- %s **%s** [%s] %s — %s\n",
				severityIcon(f.Severity), f.ID, f.Category, formatLineRef(f), f.Title,
			)
		}
		b.WriteString("\n")
	}

	if len(in.WalkthroughExtras) > 0 {
		b.WriteString("### Tripwires\n")
		for _, t := range in.WalkthroughExtras {
			fmt.Fprintf(&b, "- %s\n", t)
		}
	}
	return b.String()
}

// FormatInlineBody returns the markdown body of an inline review comment.
func FormatInlineBody(reviewerSlot string, f reviewparse.Finding) string {
	return fmt.Sprintf(
		"%s **[review by %s · severity: %s · category: %s]**\n\n%s\n\n<details>\n<summary>🤖 Prompt for AI Agent</summary>\n\n%s\n</details>\n",
		severityIcon(f.Severity), reviewerSlot, f.Severity, f.Category, f.Message, f.FixPrompt,
	)
}

// severityIcon mirrors reviewrunner.SeverityIcon. It is duplicated locally to
// avoid a future import cycle once reviewrunner imports prcommenter.
func severityIcon(s reviewparse.Severity) string {
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

func formatLineRef(f reviewparse.Finding) string {
	if f.LineStart == nil {
		return f.File
	}
	if f.LineEnd != nil && *f.LineEnd != *f.LineStart {
		return fmt.Sprintf("%s:%d-%d", f.File, *f.LineStart, *f.LineEnd)
	}
	return fmt.Sprintf("%s:%d", f.File, *f.LineStart)
}
