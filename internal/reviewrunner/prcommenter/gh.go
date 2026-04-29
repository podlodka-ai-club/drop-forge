package prcommenter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"orchv3/internal/commandrunner"
)

// inlineRejectionTripwire is appended to summary tripwires when GitHub rejects
// inline comments and the runner falls back to a summary-only review.
const inlineRejectionTripwire = "Inline review comments were rejected by GitHub (path/line not in diff). See Findings list above; fix-prompts are not anchored to specific lines for this run."

// postError carries the stderr captured from `gh api` so callers can detect
// HTTP-422 / "Path could not be resolved" rejections from GitHub.
type postError struct {
	wrapped error
	stderr  string
}

func (e *postError) Error() string {
	stderr := strings.TrimSpace(e.stderr)
	if stderr == "" {
		return fmt.Sprintf("gh api POST review: %v", e.wrapped)
	}
	return fmt.Sprintf("gh api POST review: %v: %s", e.wrapped, stderr)
}

func (e *postError) Unwrap() error { return e.wrapped }

// isInlineCommentRejection reports whether a postError represents the
// 422 "Path could not be resolved" failure GitHub returns when at least one
// inline comment points outside the PR diff. Any of these signals counts.
func isInlineCommentRejection(err error) bool {
	var pe *postError
	if !errors.As(err, &pe) {
		return false
	}
	text := strings.ToLower(pe.stderr)
	return strings.Contains(text, "422") ||
		strings.Contains(text, "unprocessable entity") ||
		strings.Contains(text, "path could not be resolved")
}

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
// summary body + all inline comments. If GitHub rejects the inline comments
// with HTTP 422 ("Path could not be resolved"), it retries the POST with no
// inline comments and a tripwire in the summary so reviewers still receive
// the findings list.
func (c GHPostReviewCommenter) PostReview(ctx context.Context, in PostReviewInput) (PostReviewResult, error) {
	exists, err := c.markerExists(ctx, in)
	if err != nil {
		return PostReviewResult{}, fmt.Errorf("check existing review marker: %w", err)
	}
	if exists {
		return PostReviewResult{Skipped: true}, nil
	}

	if err := c.postPayload(ctx, in, false); err != nil {
		if !isInlineCommentRejection(err) {
			return PostReviewResult{}, err
		}
		// Fallback: summary only, with a tripwire explaining the missing inlines.
		in.WalkthroughExtras = append(in.WalkthroughExtras, inlineRejectionTripwire)
		if err := c.postPayload(ctx, in, true); err != nil {
			return PostReviewResult{}, fmt.Errorf("fallback summary-only review: %w", err)
		}
	}
	return PostReviewResult{}, nil
}

// postPayload marshals and POSTs a review payload. When skipInline is true the
// `comments` array is forced to empty so GitHub accepts the review even if
// inline anchors don't resolve to the PR diff.
func (c GHPostReviewCommenter) postPayload(ctx context.Context, in PostReviewInput, skipInline bool) error {
	payload := buildPayload(in)
	if skipInline {
		payload["comments"] = []map[string]interface{}{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode review payload: %w", err)
	}

	var stderrBuf bytes.Buffer
	var stderrSink io.Writer = &stderrBuf
	if c.Stderr != nil {
		stderrSink = io.MultiWriter(&stderrBuf, c.Stderr)
	}

	endpoint := fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", in.RepoOwner, in.RepoName, in.PRNumber)
	cmd := commandrunner.Command{
		Name:   c.GHPath,
		Args:   []string{"api", "-X", "POST", endpoint, "--input", "-"},
		Stdin:  bytes.NewReader(body),
		Stdout: c.Stdout,
		Stderr: stderrSink,
	}
	if err := c.Command.Run(ctx, cmd); err != nil {
		return &postError{wrapped: err, stderr: stderrBuf.String()}
	}
	return nil
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
