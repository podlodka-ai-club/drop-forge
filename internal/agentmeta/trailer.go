package agentmeta

import (
	"errors"
	"fmt"
	"strings"
)

type Stage string

const (
	StageProposal Stage = "proposal"
	StageApply    Stage = "apply"
	StageArchive  Stage = "archive"
)

func ParseStage(value string) (Stage, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(StageProposal):
		return StageProposal, nil
	case string(StageApply):
		return StageApply, nil
	case string(StageArchive):
		return StageArchive, nil
	}
	return "", fmt.Errorf("unknown producer stage %q", value)
}

type Producer struct {
	By    string
	Model string
	Stage Stage
}

const (
	keyBy    = "Produced-By"
	keyModel = "Produced-Model"
	keyStage = "Produced-Stage"
)

var ErrTrailerNotFound = errors.New("producer trailer not found")

func AppendTrailer(message string, producer Producer) string {
	message = strings.TrimRight(message, "\n")

	trailer := fmt.Sprintf("%s: %s\n%s: %s\n%s: %s",
		keyBy, producer.By,
		keyModel, producer.Model,
		keyStage, string(producer.Stage),
	)

	if message == "" {
		return trailer + "\n"
	}

	if hasExistingTrailerBlock(message) {
		return message + "\n" + trailer + "\n"
	}

	return message + "\n\n" + trailer + "\n"
}

func ParseTrailer(message string) (Producer, error) {
	lines := strings.Split(message, "\n")

	var byVal, modelVal, stageVal string
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		colonIdx := strings.Index(line, ":")
		if colonIdx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])
		switch strings.ToLower(key) {
		case strings.ToLower(keyBy):
			byVal = value
		case strings.ToLower(keyModel):
			modelVal = value
		case strings.ToLower(keyStage):
			stageVal = value
		}
	}

	if byVal == "" && modelVal == "" && stageVal == "" {
		return Producer{}, ErrTrailerNotFound
	}
	if byVal == "" || modelVal == "" || stageVal == "" {
		return Producer{}, fmt.Errorf("incomplete producer trailer: by=%q model=%q stage=%q", byVal, modelVal, stageVal)
	}

	stage, err := ParseStage(stageVal)
	if err != nil {
		return Producer{}, err
	}
	return Producer{By: byVal, Model: modelVal, Stage: stage}, nil
}

func hasExistingTrailerBlock(message string) bool {
	lines := strings.Split(message, "\n")
	if len(lines) < 3 {
		return false
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			return false
		}
		if !strings.Contains(line, ":") {
			return false
		}
		if i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			return true
		}
	}
	return false
}
