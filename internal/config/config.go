package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

	defaultLinearAPIURL         = "https://api.linear.app/graphql"
	defaultLogstashBufferSize   = 1024
	defaultLogstashDialTimeout  = 2 * time.Second
	defaultProposalPollInterval = 30 * time.Second

	defaultReviewMaxContextBytes    = 256 * 1024
	defaultReviewParseRepairRetries = 1
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
	Review               ReviewRunnerConfig
	Telegram             TelegramConfig
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
	GitPath       string
	CodexPath     string
	GHPath        string
}

type LinearTaskManagerConfig struct {
	APIURL                      string
	APIToken                    string
	ProjectID                   string
	ReadyToProposeStateID       string
	ReadyToCodeStateID          string
	ReadyToArchiveStateID       string
	ProposingInProgressStateID  string
	CodeInProgressStateID       string
	ArchivingInProgressStateID  string
	NeedProposalReviewStateID   string
	NeedCodeReviewStateID       string
	NeedArchiveReviewStateID    string
	NeedProposalAIReviewStateID string
	NeedCodeAIReviewStateID     string
	NeedArchiveAIReviewStateID  string
}

type ReviewRunnerConfig struct {
	PrimarySlot           string
	SecondarySlot         string
	PrimaryModel          string
	SecondaryModel        string
	PrimaryExecutorPath   string
	SecondaryExecutorPath string
	MaxContextBytes       int
	ParseRepairRetries    int
	PromptDir             string
}

type TelegramConfig struct {
	Enabled  bool
	BotToken string
	ChatID   string
	APIURL   string
	Timeout  time.Duration
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

	reviewMaxContextBytes, err := intFromEnv("REVIEW_MAX_CONTEXT_BYTES", defaultReviewMaxContextBytes)
	if err != nil {
		return Config{}, err
	}

	reviewParseRepairRetries, err := intFromEnv("REVIEW_PARSE_REPAIR_RETRIES", defaultReviewParseRepairRetries)
	if err != nil {
		return Config{}, err
	}

	telegramCfg, err := loadTelegramConfig()
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
			GitPath:       trimmedStringFromEnv("PROPOSAL_GIT_PATH", defaultProposalGitPath),
			CodexPath:     trimmedStringFromEnv("PROPOSAL_CODEX_PATH", defaultProposalCodexPath),
			GHPath:        trimmedStringFromEnv("PROPOSAL_GH_PATH", defaultProposalGHPath),
		},
		TaskManager: LinearTaskManagerConfig{
			APIURL:                      trimmedStringFromEnv("LINEAR_API_URL", defaultLinearAPIURL),
			APIToken:                    trimmedStringFromEnv("LINEAR_API_TOKEN", ""),
			ProjectID:                   trimmedStringFromEnv("LINEAR_PROJECT_ID", ""),
			ReadyToProposeStateID:       trimmedStringFromEnv("LINEAR_STATE_READY_TO_PROPOSE_ID", ""),
			ReadyToCodeStateID:          trimmedStringFromEnv("LINEAR_STATE_READY_TO_CODE_ID", ""),
			ReadyToArchiveStateID:       trimmedStringFromEnv("LINEAR_STATE_READY_TO_ARCHIVE_ID", ""),
			ProposingInProgressStateID:  trimmedStringFromEnv("LINEAR_STATE_PROPOSING_IN_PROGRESS_ID", ""),
			CodeInProgressStateID:       trimmedStringFromEnv("LINEAR_STATE_CODE_IN_PROGRESS_ID", ""),
			ArchivingInProgressStateID:  trimmedStringFromEnv("LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID", ""),
			NeedProposalReviewStateID:   trimmedStringFromEnv("LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID", ""),
			NeedCodeReviewStateID:       trimmedStringFromEnv("LINEAR_STATE_NEED_CODE_REVIEW_ID", ""),
			NeedArchiveReviewStateID:    trimmedStringFromEnv("LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID", ""),
			NeedProposalAIReviewStateID: trimmedStringFromEnv("LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID", ""),
			NeedCodeAIReviewStateID:     trimmedStringFromEnv("LINEAR_STATE_NEED_CODE_AI_REVIEW_ID", ""),
			NeedArchiveAIReviewStateID:  trimmedStringFromEnv("LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID", ""),
		},
		Review: ReviewRunnerConfig{
			PrimarySlot:           trimmedStringFromEnv("REVIEW_ROLE_PRIMARY", ""),
			SecondarySlot:         trimmedStringFromEnv("REVIEW_ROLE_SECONDARY", ""),
			PrimaryModel:          trimmedStringFromEnv("REVIEW_PRIMARY_MODEL", ""),
			SecondaryModel:        trimmedStringFromEnv("REVIEW_SECONDARY_MODEL", ""),
			PrimaryExecutorPath:   trimmedStringFromEnv("REVIEW_PRIMARY_EXECUTOR_PATH", ""),
			SecondaryExecutorPath: trimmedStringFromEnv("REVIEW_SECONDARY_EXECUTOR_PATH", ""),
			MaxContextBytes:       reviewMaxContextBytes,
			ParseRepairRetries:    reviewParseRepairRetries,
			PromptDir:             trimmedStringFromEnv("REVIEW_PROMPT_DIR", ""),
		},
		Telegram: telegramCfg,
		Logstash: logstashCfg,
	}, nil
}

func loadTelegramConfig() (TelegramConfig, error) {
	enabled, err := boolFromEnv("TELEGRAM_NOTIFICATIONS_ENABLED", false)
	if err != nil {
		return TelegramConfig{}, err
	}

	timeout, err := durationFromEnv("TELEGRAM_TIMEOUT", 0)
	if err != nil {
		return TelegramConfig{}, err
	}

	cfg := TelegramConfig{
		Enabled:  enabled,
		BotToken: trimmedStringFromEnv("TELEGRAM_BOT_TOKEN", ""),
		ChatID:   trimmedStringFromEnv("TELEGRAM_CHAT_ID", ""),
		APIURL:   strings.TrimRight(trimmedStringFromEnv("TELEGRAM_API_URL", ""), "/"),
		Timeout:  timeout,
	}
	if err := cfg.Validate(); err != nil {
		return TelegramConfig{}, err
	}

	return cfg, nil
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
		"PROPOSAL_GH_PATH":         cfg.GHPath,
	}

	for key, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", key)
		}
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

	aiStates := map[string]string{
		"LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID": cfg.NeedProposalAIReviewStateID,
		"LINEAR_STATE_NEED_CODE_AI_REVIEW_ID":     cfg.NeedCodeAIReviewStateID,
		"LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID":  cfg.NeedArchiveAIReviewStateID,
	}
	emptyAI := 0
	for _, v := range aiStates {
		if strings.TrimSpace(v) == "" {
			emptyAI++
		}
	}
	if emptyAI != 0 && emptyAI != len(aiStates) {
		var missing []string
		for k, v := range aiStates {
			if strings.TrimSpace(v) == "" {
				missing = append(missing, k)
			}
		}
		sort.Strings(missing)
		return fmt.Errorf("AI review configuration is partial; set all three or none. Missing: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (cfg TelegramConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}

	requiredValues := map[string]string{
		"TELEGRAM_BOT_TOKEN": cfg.BotToken,
		"TELEGRAM_CHAT_ID":   cfg.ChatID,
		"TELEGRAM_API_URL":   cfg.APIURL,
	}

	for key, value := range requiredValues {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty when TELEGRAM_NOTIFICATIONS_ENABLED=true", key)
		}
	}
	if cfg.Timeout <= 0 {
		return fmt.Errorf("TELEGRAM_TIMEOUT must be a positive duration when TELEGRAM_NOTIFICATIONS_ENABLED=true, got %s", cfg.Timeout)
	}

	return nil
}

func (cfg LinearTaskManagerConfig) ManagedStateIDs() []string {
	states := []string{
		strings.TrimSpace(cfg.ReadyToProposeStateID),
		strings.TrimSpace(cfg.ReadyToCodeStateID),
		strings.TrimSpace(cfg.ReadyToArchiveStateID),
		strings.TrimSpace(cfg.NeedProposalAIReviewStateID),
		strings.TrimSpace(cfg.NeedCodeAIReviewStateID),
		strings.TrimSpace(cfg.NeedArchiveAIReviewStateID),
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

func (cfg ReviewRunnerConfig) Enabled(tm LinearTaskManagerConfig) bool {
	required := []string{
		tm.NeedProposalAIReviewStateID,
		tm.NeedCodeAIReviewStateID,
		tm.NeedArchiveAIReviewStateID,
		cfg.PrimarySlot,
		cfg.SecondarySlot,
		cfg.PrimaryModel,
		cfg.SecondaryModel,
		cfg.PrimaryExecutorPath,
		cfg.SecondaryExecutorPath,
	}
	for _, v := range required {
		if strings.TrimSpace(v) == "" {
			return false
		}
	}
	return true
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
