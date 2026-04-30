package reviewrunner

import (
	"errors"
	"fmt"

	"orchv3/internal/agentmeta"
	"orchv3/internal/config"
)

type Reviewer struct {
	Slot            string
	Model           string
	ExecutorPath    string
	ProducerUnknown bool
}

var ErrUnknownProducerSlot = errors.New("producer slot not in REVIEW_ROLE_PRIMARY or REVIEW_ROLE_SECONDARY")

// SelectReviewer chooses the reviewer slot opposite to the producer that wrote
// the most recent HEAD commit. When the producer trailer is missing (Producer{}.By == ""),
// it falls back to the secondary slot and marks ProducerUnknown=true so the
// caller can surface a tripwire in the published review.
func SelectReviewer(cfg config.ReviewRunnerConfig, producer agentmeta.Producer) (Reviewer, error) {
	if producer.By == "" {
		return Reviewer{
			Slot:            cfg.SecondarySlot,
			Model:           cfg.SecondaryModel,
			ExecutorPath:    cfg.SecondaryExecutorPath,
			ProducerUnknown: true,
		}, nil
	}
	switch producer.By {
	case cfg.PrimarySlot:
		return Reviewer{
			Slot:         cfg.SecondarySlot,
			Model:        cfg.SecondaryModel,
			ExecutorPath: cfg.SecondaryExecutorPath,
		}, nil
	case cfg.SecondarySlot:
		return Reviewer{
			Slot:         cfg.PrimarySlot,
			Model:        cfg.PrimaryModel,
			ExecutorPath: cfg.PrimaryExecutorPath,
		}, nil
	}
	return Reviewer{}, fmt.Errorf("%w: producer=%q primary=%q secondary=%q",
		ErrUnknownProducerSlot, producer.By, cfg.PrimarySlot, cfg.SecondarySlot)
}
