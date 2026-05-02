package reviewrunner

import (
	"errors"
	"testing"

	"orchv3/internal/agentmeta"
	"orchv3/internal/config"
)

func testReviewConfig() config.ReviewRunnerConfig {
	return config.ReviewRunnerConfig{
		PrimarySlot:           "claude",
		SecondarySlot:         "codex",
		PrimaryModel:          "claude-opus",
		SecondaryModel:        "gpt-5",
		PrimaryExecutorPath:   "/usr/local/bin/claude",
		SecondaryExecutorPath: "/usr/local/bin/codex",
	}
}

func TestSelectReviewerProducerIsPrimaryReturnsSecondary(t *testing.T) {
	cfg := testReviewConfig()
	producer := agentmeta.Producer{By: cfg.PrimarySlot, Model: cfg.PrimaryModel}

	reviewer, err := SelectReviewer(cfg, producer)
	if err != nil {
		t.Fatalf("SelectReviewer returned error: %v", err)
	}
	if reviewer.Slot != cfg.SecondarySlot {
		t.Errorf("Slot = %q, want %q", reviewer.Slot, cfg.SecondarySlot)
	}
	if reviewer.Model != cfg.SecondaryModel {
		t.Errorf("Model = %q, want %q", reviewer.Model, cfg.SecondaryModel)
	}
	if reviewer.ExecutorPath != cfg.SecondaryExecutorPath {
		t.Errorf("ExecutorPath = %q, want %q", reviewer.ExecutorPath, cfg.SecondaryExecutorPath)
	}
	if reviewer.ProducerUnknown {
		t.Errorf("ProducerUnknown = true, want false when producer is known")
	}
}

func TestSelectReviewerProducerIsSecondaryReturnsPrimary(t *testing.T) {
	cfg := testReviewConfig()
	producer := agentmeta.Producer{By: cfg.SecondarySlot, Model: cfg.SecondaryModel}

	reviewer, err := SelectReviewer(cfg, producer)
	if err != nil {
		t.Fatalf("SelectReviewer returned error: %v", err)
	}
	if reviewer.Slot != cfg.PrimarySlot {
		t.Errorf("Slot = %q, want %q", reviewer.Slot, cfg.PrimarySlot)
	}
	if reviewer.Model != cfg.PrimaryModel {
		t.Errorf("Model = %q, want %q", reviewer.Model, cfg.PrimaryModel)
	}
	if reviewer.ExecutorPath != cfg.PrimaryExecutorPath {
		t.Errorf("ExecutorPath = %q, want %q", reviewer.ExecutorPath, cfg.PrimaryExecutorPath)
	}
	if reviewer.ProducerUnknown {
		t.Errorf("ProducerUnknown = true, want false when producer is known")
	}
}

func TestSelectReviewerWithoutProducerFallsBackToSecondary(t *testing.T) {
	cfg := testReviewConfig()
	producer := agentmeta.Producer{}

	reviewer, err := SelectReviewer(cfg, producer)
	if err != nil {
		t.Fatalf("SelectReviewer returned error: %v", err)
	}
	if reviewer.Slot != cfg.SecondarySlot {
		t.Errorf("Slot = %q, want %q (fallback to secondary)", reviewer.Slot, cfg.SecondarySlot)
	}
	if reviewer.Model != cfg.SecondaryModel {
		t.Errorf("Model = %q, want %q", reviewer.Model, cfg.SecondaryModel)
	}
	if reviewer.ExecutorPath != cfg.SecondaryExecutorPath {
		t.Errorf("ExecutorPath = %q, want %q", reviewer.ExecutorPath, cfg.SecondaryExecutorPath)
	}
	if !reviewer.ProducerUnknown {
		t.Errorf("ProducerUnknown = false, want true when producer trailer is missing")
	}
}

func TestSelectReviewerUnknownProducerSlotIsConfigError(t *testing.T) {
	cfg := testReviewConfig()
	producer := agentmeta.Producer{By: "bardo"}

	reviewer, err := SelectReviewer(cfg, producer)
	if err == nil {
		t.Fatalf("SelectReviewer returned no error, want ErrUnknownProducerSlot; reviewer=%+v", reviewer)
	}
	if !errors.Is(err, ErrUnknownProducerSlot) {
		t.Errorf("err = %v, want errors.Is(err, ErrUnknownProducerSlot) == true", err)
	}
}
