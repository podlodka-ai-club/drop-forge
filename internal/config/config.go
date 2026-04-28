package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv   = "development"
	defaultAppName  = "orchv3"
	defaultLogLevel = "debug"
	defaultHTTPPort = 8080

	defaultProposalBaseBranch    = "main"
	defaultProposalRemoteName    = "origin"
	defaultProposalBranchPrefix  = "codex/proposal"
	defaultProposalPRTitlePrefix = "OpenSpec proposal:"
	defaultProposalGitPath       = "git"
	defaultProposalCodexPath     = "codex"
	defaultProposalGHPath        = "gh"
	defaultProposalGLabPath      = "glab"

	defaultLinearAPIURL         = "https://api.linear.app/graphql"
	defaultLogstashBufferSize   = 1024
	defaultLogstashDialTimeout  = 2 * time.Second
	defaultProposalPollInterval = 30 * time.Second
)

const (
	GitProviderGitHub = "github"
	GitProviderGitLab = "gitlab"
)

type Config struct {
	AppEnv               string
	AppName              string
	LogLevel             string
	HTTPPort             int
	OpenAIAPIKey         string
	ProposalPollInterval time.Duration
	ProposalRunner       ProposalRunnerConfig
	TaskManager          LinearTaskManagerConfig
	Logstash             LogstashConfig
}

type LogstashConfig struct {
	Addr        string
	BufferSize  int
	DialTimeout time.Duration
}

type ProposalRunnerConfig struct {
	RepositoryURL string
	BaseBranch    string
	RemoteName    string
	BranchPrefix  string
	PRTitlePrefix string
	CleanupTemp   bool
	GitProvider   string
	GitPath       string
	CodexPath     string
	GHPath        string
	GLabPath      string
}

type LinearTaskManagerConfig struct {
	APIURL                     string
	APIToken                   string
	ProjectID                  string
	ReadyToProposeStateID      string
	ReadyToCodeStateID         string
	ReadyToArchiveStateID      string
	ProposingInProgressStateID string
	CodeInProgressStateID      string
	ArchivingInProgressStateID string
	NeedProposalReviewStateID  string
	NeedCodeReviewStateID      string
	NeedArchiveReviewStateID   string
}

func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	httpPort, err := intFromEnv("HTTP_PORT", defaultHTTPPort)
	if err != nil {
		return Config{}, err
	}

	cleanupTemp, err := boolFromEnv("PROPOSAL_CLEANUP_TEMP", false)
	if err != nil {
		return Config{}, err
	}

	logstashCfg, err := loadLogstashConfig()
	if err != nil {
		return Config{}, err
	}

	proposalPollInterval, err := positiveDurationFromEnv("PROPOSAL_POLL_INTERVAL", defaultProposalPollInterval)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:               trimmedStringFromEnv("APP_ENV", defaultAppEnv),
		AppName:              trimmedStringFromEnv("APP_NAME", defaultAppName),
		LogLevel:             trimmedStringFromEnv("LOG_LEVEL", defaultLogLevel),
		HTTPPort:             httpPort,
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		ProposalPollInterval: proposalPollInterval,
		ProposalRunner: ProposalRunnerConfig{
			RepositoryURL: trimmedStringFromEnv("PROPOSAL_REPOSITORY_URL", ""),
			BaseBranch:    trimmedStringFromEnv("PROPOSAL_BASE_BRANCH", defaultProposalBaseBranch),
			RemoteName:    trimmedStringFromEnv("PROPOSAL_REMOTE_NAME", defaultProposalRemoteName),
			BranchPrefix:  trimmedStringFromEnv("PROPOSAL_BRANCH_PREFIX", defaultProposalBranchPrefix),
			PRTitlePrefix: trimmedStringFromEnv("PROPOSAL_PR_TITLE_PREFIX", defaultProposalPRTitlePrefix),
			CleanupTemp:   cleanupTemp,
			GitProvider:   trimmedStringFromEnv("PROPOSAL_GIT_PROVIDER", GitProviderGitHub),
			GitPath:       trimmedStringFromEnv("PROPOSAL_GIT_PATH", defaultProposalGitPath),
			CodexPath:     trimmedStringFromEnv("PROPOSAL_CODEX_PATH", defaultProposalCodexPath),
			GHPath:        trimmedStringFromEnv("PROPOSAL_GH_PATH", defaultProposalGHPath),
			GLabPath:      trimmedStringFromEnv("PROPOSAL_GLAB_PATH", defaultProposalGLabPath),
		},
		TaskManager: LinearTaskManagerConfig{
			APIURL:                     trimmedStringFromEnv("LINEAR_API_URL", defaultLinearAPIURL),
			APIToken:                   trimmedStringFromEnv("LINEAR_API_TOKEN", ""),
			ProjectID:                  trimmedStringFromEnv("LINEAR_PROJECT_ID", ""),
			ReadyToProposeStateID:      trimmedStringFromEnv("LINEAR_STATE_READY_TO_PROPOSE_ID", ""),
			ReadyToCodeStateID:         trimmedStringFromEnv("LINEAR_STATE_READY_TO_CODE_ID", ""),
			ReadyToArchiveStateID:      trimmedStringFromEnv("LINEAR_STATE_READY_TO_ARCHIVE_ID", ""),
			ProposingInProgressStateID: trimmedStringFromEnv("LINEAR_STATE_PROPOSING_IN_PROGRESS_ID", ""),
			CodeInProgressStateID:      trimmedStringFromEnv("LINEAR_STATE_CODE_IN_PROGRESS_ID", ""),
			ArchivingInProgressStateID: trimmedStringFromEnv("LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID", ""),
			NeedProposalReviewStateID:  trimmedStringFromEnv("LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID", ""),
			NeedCodeReviewStateID:      trimmedStringFromEnv("LINEAR_STATE_NEED_CODE_REVIEW_ID", ""),
			NeedArchiveReviewStateID:   trimmedStringFromEnv("LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID", ""),
		},
		Logstash: logstashCfg,
	}, nil
}

func loadLogstashConfig() (LogstashConfig, error) {
	bufferSize, err := intFromEnv("LOGSTASH_BUFFER_SIZE", defaultLogstashBufferSize)
	if err != nil {
		return LogstashConfig{}, err
	}
	if bufferSize < 1 {
		return LogstashConfig{}, fmt.Errorf("LOGSTASH_BUFFER_SIZE must be >= 1, got %d", bufferSize)
	}

	dialTimeout, err := durationFromEnv("LOGSTASH_DIAL_TIMEOUT", defaultLogstashDialTimeout)
	if err != nil {
		return LogstashConfig{}, err
	}

	return LogstashConfig{
		Addr:        trimmedStringFromEnv("LOGSTASH_ADDR", ""),
		BufferSize:  bufferSize,
		DialTimeout: dialTimeout,
	}, nil
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", key, err)
	}

	return parsed, nil
}

func positiveDurationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	parsed, err := durationFromEnv(key, fallback)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, got %s", key, parsed)
	}

	return parsed, nil
}

func (cfg ProposalRunnerConfig) Validate() error {
	if strings.TrimSpace(cfg.RepositoryURL) == "" {
		return errors.New("proposal runner repository URL is required: set PROPOSAL_REPOSITORY_URL")
	}

	requiredValues := map[string]string{
		"PROPOSAL_BASE_BRANCH":     cfg.BaseBranch,
		"PROPOSAL_REMOTE_NAME":     cfg.RemoteName,
		"PROPOSAL_BRANCH_PREFIX":   cfg.BranchPrefix,
		"PROPOSAL_PR_TITLE_PREFIX": cfg.PRTitlePrefix,
		"PROPOSAL_GIT_PATH":        cfg.GitPath,
		"PROPOSAL_CODEX_PATH":      cfg.CodexPath,
	}

	for key, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", key)
		}
	}

	if err := cfg.ValidateProvider(); err != nil {
		return err
	}

	return nil
}

func (cfg ProposalRunnerConfig) NormalizedGitProvider() string {
	provider := strings.ToLower(strings.TrimSpace(cfg.GitProvider))
	if provider == "" {
		return GitProviderGitHub
	}

	return provider
}

func (cfg ProposalRunnerConfig) ValidateProvider() error {
	switch cfg.NormalizedGitProvider() {
	case GitProviderGitHub:
		if strings.TrimSpace(cfg.GHPath) == "" {
			return errors.New("PROPOSAL_GH_PATH must not be empty for github provider")
		}
	case GitProviderGitLab:
		if strings.TrimSpace(cfg.GLabPath) == "" {
			return errors.New("PROPOSAL_GLAB_PATH must not be empty for gitlab provider")
		}
	default:
		return fmt.Errorf("PROPOSAL_GIT_PROVIDER must be one of %q or %q, got %q", GitProviderGitHub, GitProviderGitLab, cfg.GitProvider)
	}

	return nil
}

func (cfg LinearTaskManagerConfig) Validate() error {
	requiredValues := map[string]string{
		"LINEAR_API_URL":                        cfg.APIURL,
		"LINEAR_API_TOKEN":                      cfg.APIToken,
		"LINEAR_PROJECT_ID":                     cfg.ProjectID,
		"LINEAR_STATE_READY_TO_PROPOSE_ID":      cfg.ReadyToProposeStateID,
		"LINEAR_STATE_READY_TO_CODE_ID":         cfg.ReadyToCodeStateID,
		"LINEAR_STATE_READY_TO_ARCHIVE_ID":      cfg.ReadyToArchiveStateID,
		"LINEAR_STATE_PROPOSING_IN_PROGRESS_ID": cfg.ProposingInProgressStateID,
		"LINEAR_STATE_CODE_IN_PROGRESS_ID":      cfg.CodeInProgressStateID,
		"LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID": cfg.ArchivingInProgressStateID,
		"LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID":  cfg.NeedProposalReviewStateID,
		"LINEAR_STATE_NEED_CODE_REVIEW_ID":      cfg.NeedCodeReviewStateID,
		"LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID":   cfg.NeedArchiveReviewStateID,
	}

	for key, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", key)
		}
	}

	return nil
}

func (cfg LinearTaskManagerConfig) ManagedStateIDs() []string {
	states := []string{
		strings.TrimSpace(cfg.ReadyToProposeStateID),
		strings.TrimSpace(cfg.ReadyToCodeStateID),
		strings.TrimSpace(cfg.ReadyToArchiveStateID),
	}

	result := make([]string, 0, len(states))
	seen := make(map[string]struct{}, len(states))
	for _, state := range states {
		if state == "" {
			continue
		}
		if _, ok := seen[state]; ok {
			continue
		}
		seen[state] = struct{}{}
		result = append(result, state)
	}

	return result
}

func loadDotEnv() error {
	path, err := findDotEnv()
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	return nil
}

func findDotEnv() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("stat .env: %w", err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func stringFromEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func trimmedStringFromEnv(key string, fallback string) string {
	return strings.TrimSpace(stringFromEnv(key, fallback))
}

func intFromEnv(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return parsed, nil
}

func boolFromEnv(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}

	return parsed, nil
}
