package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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
)

type Config struct {
	AppEnv         string
	AppName        string
	LogLevel       string
	HTTPPort       int
	OpenAIAPIKey   string
	ProposalRunner ProposalRunnerConfig
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

	return Config{
		AppEnv:       trimmedStringFromEnv("APP_ENV", defaultAppEnv),
		AppName:      trimmedStringFromEnv("APP_NAME", defaultAppName),
		LogLevel:     trimmedStringFromEnv("LOG_LEVEL", defaultLogLevel),
		HTTPPort:     httpPort,
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
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
	}, nil
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

func loadDotEnv() error {
	err := godotenv.Load()
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return fmt.Errorf("load .env: %w", err)
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
