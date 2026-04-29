package prcommenter

import (
	"context"

	"orchv3/internal/agentmeta"
	"orchv3/internal/reviewrunner/reviewparse"
)

// PostReviewInput carries the data required to publish a single atomic PR
// review on GitHub through the GitHub Pull Request Reviews API.
type PostReviewInput struct {
	RepoOwner         string
	RepoName          string
	PRNumber          int
	HeadSHA           string
	Stage             agentmeta.Stage
	ReviewerSlot      string
	Review            reviewparse.Review
	WalkthroughExtras []string
}

// PostReviewResult reports whether posting was a no-op because an existing
// review with the same idempotency marker was already present.
type PostReviewResult struct {
	Skipped bool
}

// PRCommenter abstracts the act of publishing a parsed review to a PR. It
// guarantees idempotency by reviewer slot, stage and HEAD-sha.
type PRCommenter interface {
	PostReview(ctx context.Context, in PostReviewInput) (PostReviewResult, error)
}

// MarkerFor returns the HTML-comment marker string used to deduplicate reviews
// posted to the same PR for the same (reviewer, stage, HEAD-sha) tuple.
func MarkerFor(reviewerSlot string, stage agentmeta.Stage, headSHA string) string {
	return "<!-- drop-forge-review-marker:" + reviewerSlot + ":" + string(stage) + ":" + headSHA + " -->"
}
