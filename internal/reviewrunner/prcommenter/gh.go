package prcommenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"orchv3/internal/commandrunner"
)

// GHPostReviewCommenter publishes review output through the GitHub CLI's
// `gh api` command. It performs an idempotency check by listing existing
// reviews and skipping when a marker already exists, then sends a single
// atomic POST containing summary + all inline comments.
type GHPostReviewCommenter struct {
	Command commandrunner.Runner
	GHPath  string
	Service string
	Stdout  io.Writer
	Stderr  io.Writer
}

type ghReview struct {
	Body string `json:"body"`
}

// PostReview implements PRCommenter. It first GETs existing reviews to detect
// a previously-published marker; if none, it POSTs a single atomic review with
// summary body + all inline comments.
func (c GHPostReviewCommenter) PostReview(ctx context.Context, in PostReviewInput) (PostReviewResult, error) {
	exists, err := c.markerExists(ctx, in)
	if err != nil {
		return PostReviewResult{}, fmt.Errorf("check existing review marker: %w", err)
	}
	if exists {
		return PostReviewResult{Skipped: true}, nil
	}

	payload := buildPayload(in)
	body, err := json.Marshal(payload)
	if err != nil {
		return PostReviewResult{}, fmt.Errorf("encode review payload: %w", err)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", in.RepoOwner, in.RepoName, in.PRNumber)
	cmd := commandrunner.Command{
		Name:   c.GHPath,
		Args:   []string{"api", "-X", "POST", endpoint, "--input", "-"},
		Stdin:  bytes.NewReader(body),
		Stdout: c.Stdout,
		Stderr: c.Stderr,
	}
	if err := c.Command.Run(ctx, cmd); err != nil {
		return PostReviewResult{}, fmt.Errorf("gh api POST review: %w", err)
	}
	return PostReviewResult{}, nil
}

func (c GHPostReviewCommenter) markerExists(ctx context.Context, in PostReviewInput) (bool, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", in.RepoOwner, in.RepoName, in.PRNumber)
	var out bytes.Buffer
	if err := c.Command.Run(ctx, commandrunner.Command{
		Name:   c.GHPath,
		Args:   []string{"api", endpoint, "--paginate"},
		Stdout: &out,
		Stderr: c.Stderr,
	}); err != nil {
		return false, fmt.Errorf("gh api GET reviews: %w", err)
	}

	var reviews []ghReview
	text := strings.TrimSpace(out.String())
	if text == "" {
		return false, nil
	}
	// gh api --paginate emits concatenated JSON arrays. Use a streaming decoder.
	dec := json.NewDecoder(strings.NewReader(text))
	for dec.More() {
		var batch []ghReview
		if err := dec.Decode(&batch); err != nil {
			return false, fmt.Errorf("decode reviews JSON: %w", err)
		}
		reviews = append(reviews, batch...)
	}

	marker := MarkerFor(in.ReviewerSlot, in.Stage, in.HeadSHA)
	for _, r := range reviews {
		if strings.Contains(r.Body, marker) {
			return true, nil
		}
	}
	return false, nil
}

// buildPayload composes the review payload sent to GitHub's Reviews API.
// `event: COMMENT` keeps the review informational (not approval/changes-requested).
// Only findings with non-nil LineStart become inline comments.
func buildPayload(in PostReviewInput) map[string]interface{} {
	body := FormatSummaryBody(in)
	comments := make([]map[string]interface{}, 0)
	for _, f := range in.Review.Findings {
		if f.LineStart == nil {
			continue
		}
		c := map[string]interface{}{
			"path": f.File,
			"line": *f.LineStart,
			"side": "RIGHT",
			"body": FormatInlineBody(in.ReviewerSlot, f),
		}
		if f.LineEnd != nil && *f.LineEnd != *f.LineStart {
			c["start_line"] = *f.LineStart
			c["line"] = *f.LineEnd
			c["start_side"] = "RIGHT"
		}
		comments = append(comments, c)
	}
	return map[string]interface{}{
		"commit_id": in.HeadSHA,
		"event":     "COMMENT",
		"body":      body,
		"comments":  comments,
	}
}
