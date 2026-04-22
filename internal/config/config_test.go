package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var envKeys = []string{
	"APP_ENV",
	"APP_NAME",
	"LOG_LEVEL",
	"HTTP_PORT",
	"OPENAI_API_KEY",
	"PROPOSAL_REPOSITORY_URL",
	"PROPOSAL_BASE_BRANCH",
	"PROPOSAL_REMOTE_NAME",
	"PROPOSAL_BRANCH_PREFIX",
	"PROPOSAL_PR_TITLE_PREFIX",
	"PROPOSAL_CLEANUP_TEMP",
	"PROPOSAL_GIT_PATH",
	"PROPOSAL_CODEX_PATH",
	"PROPOSAL_GH_PATH",
}

func TestLoadUsesDefaults(t *testing.T) {
	isolateEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.AppEnv != defaultAppEnv {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, defaultAppEnv)
	}

	if cfg.AppName != defaultAppName {
		t.Fatalf("AppName = %q, want %q", cfg.AppName, defaultAppName)
	}

	if cfg.LogLevel != defaultLogLevel {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, defaultLogLevel)
	}

	if cfg.HTTPPort != defaultHTTPPort {
		t.Fatalf("HTTPPort = %d, want %d", cfg.HTTPPort, defaultHTTPPort)
	}

	if cfg.ProposalRunner.BaseBranch != defaultProposalBaseBranch {
		t.Fatalf("ProposalRunner.BaseBranch = %q, want %q", cfg.ProposalRunner.BaseBranch, defaultProposalBaseBranch)
	}

	if cfg.ProposalRunner.CleanupTemp {
		t.Fatal("ProposalRunner.CleanupTemp = true, want false")
	}

	if cfg.ProposalRunner.GitPath != defaultProposalGitPath {
		t.Fatalf("ProposalRunner.GitPath = %q, want %q", cfg.ProposalRunner.GitPath, defaultProposalGitPath)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	isolateEnv(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_NAME", "orch-test")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("OPENAI_API_KEY", "secret")
	t.Setenv("PROPOSAL_REPOSITORY_URL", "git@github.com:example/project.git")
	t.Setenv("PROPOSAL_BASE_BRANCH", "develop")
	t.Setenv("PROPOSAL_REMOTE_NAME", "upstream")
	t.Setenv("PROPOSAL_BRANCH_PREFIX", "automation/proposal")
	t.Setenv("PROPOSAL_PR_TITLE_PREFIX", "Spec:")
	t.Setenv("PROPOSAL_CLEANUP_TEMP", "true")
	t.Setenv("PROPOSAL_GIT_PATH", "/usr/local/bin/git")
	t.Setenv("PROPOSAL_CODEX_PATH", "/usr/local/bin/codex")
	t.Setenv("PROPOSAL_GH_PATH", "/usr/local/bin/gh")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.AppEnv != "test" {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, "test")
	}

	if cfg.AppName != "orch-test" {
		t.Fatalf("AppName = %q, want %q", cfg.AppName, "orch-test")
	}

	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}

	if cfg.HTTPPort != 9090 {
		t.Fatalf("HTTPPort = %d, want %d", cfg.HTTPPort, 9090)
	}

	if cfg.OpenAIAPIKey != "secret" {
		t.Fatalf("OpenAIAPIKey = %q, want %q", cfg.OpenAIAPIKey, "secret")
	}

	runnerCfg := cfg.ProposalRunner
	if runnerCfg.RepositoryURL != "git@github.com:example/project.git" {
		t.Fatalf("RepositoryURL = %q", runnerCfg.RepositoryURL)
	}
	if runnerCfg.BaseBranch != "develop" {
		t.Fatalf("BaseBranch = %q", runnerCfg.BaseBranch)
	}
	if runnerCfg.RemoteName != "upstream" {
		t.Fatalf("RemoteName = %q", runnerCfg.RemoteName)
	}
	if runnerCfg.BranchPrefix != "automation/proposal" {
		t.Fatalf("BranchPrefix = %q", runnerCfg.BranchPrefix)
	}
	if runnerCfg.PRTitlePrefix != "Spec:" {
		t.Fatalf("PRTitlePrefix = %q", runnerCfg.PRTitlePrefix)
	}
	if !runnerCfg.CleanupTemp {
		t.Fatal("CleanupTemp = false, want true")
	}
	if runnerCfg.GitPath != "/usr/local/bin/git" {
		t.Fatalf("GitPath = %q", runnerCfg.GitPath)
	}
	if runnerCfg.CodexPath != "/usr/local/bin/codex" {
		t.Fatalf("CodexPath = %q", runnerCfg.CodexPath)
	}
	if runnerCfg.GHPath != "/usr/local/bin/gh" {
		t.Fatalf("GHPath = %q", runnerCfg.GHPath)
	}
}

func TestLoadReturnsErrorForInvalidHTTPPort(t *testing.T) {
	isolateEnv(t)
	t.Setenv("HTTP_PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForInvalidCleanupFlag(t *testing.T) {
	isolateEnv(t)
	t.Setenv("PROPOSAL_CLEANUP_TEMP", "sometimes")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReadsDotEnvWithGodotenvSyntax(t *testing.T) {
	isolateEnv(t)
	writeDotEnv(t, `
APP_NAME="orch app" # inline comment
PROPOSAL_REPOSITORY_URL='git@github.com:example/from-dotenv.git'
PROPOSAL_BASE_BRANCH=feature/base
PROPOSAL_CLEANUP_TEMP=true
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.AppName != "orch app" {
		t.Fatalf("AppName = %q, want %q", cfg.AppName, "orch app")
	}

	if cfg.ProposalRunner.RepositoryURL != "git@github.com:example/from-dotenv.git" {
		t.Fatalf("RepositoryURL = %q", cfg.ProposalRunner.RepositoryURL)
	}

	if cfg.ProposalRunner.BaseBranch != "feature/base" {
		t.Fatalf("BaseBranch = %q", cfg.ProposalRunner.BaseBranch)
	}

	if !cfg.ProposalRunner.CleanupTemp {
		t.Fatal("CleanupTemp = false, want true")
	}

}

func TestLoadProcessEnvironmentOverridesDotEnv(t *testing.T) {
	isolateEnv(t)
	writeDotEnv(t, `
PROPOSAL_REPOSITORY_URL=git@github.com:example/from-dotenv.git
PROPOSAL_BASE_BRANCH=main
`)
	t.Setenv("PROPOSAL_REPOSITORY_URL", "git@github.com:example/from-process.git")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.ProposalRunner.RepositoryURL != "git@github.com:example/from-process.git" {
		t.Fatalf("RepositoryURL = %q, want process env value", cfg.ProposalRunner.RepositoryURL)
	}

	if cfg.ProposalRunner.BaseBranch != "main" {
		t.Fatalf("BaseBranch = %q, want .env value", cfg.ProposalRunner.BaseBranch)
	}
}

func TestProposalRunnerConfigValidate(t *testing.T) {
	valid := ProposalRunnerConfig{
		RepositoryURL: "git@github.com:example/project.git",
		BaseBranch:    "main",
		RemoteName:    "origin",
		BranchPrefix:  "codex/proposal",
		PRTitlePrefix: "OpenSpec proposal:",
		GitPath:       "git",
		CodexPath:     "codex",
		GHPath:        "gh",
	}

	tests := []struct {
		name    string
		mutate  func(*ProposalRunnerConfig)
		wantErr string
	}{
		{
			name: "valid",
		},
		{
			name: "missing repository",
			mutate: func(cfg *ProposalRunnerConfig) {
				cfg.RepositoryURL = " "
			},
			wantErr: "PROPOSAL_REPOSITORY_URL",
		},
		{
			name: "missing git path",
			mutate: func(cfg *ProposalRunnerConfig) {
				cfg.GitPath = " "
			},
			wantErr: "PROPOSAL_GIT_PATH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() returned error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("Validate() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func isolateEnv(t *testing.T) {
	t.Helper()

	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	for _, key := range envKeys {
		oldValue, hadValue := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}

		t.Cleanup(func(key string, oldValue string, hadValue bool) func() {
			return func() {
				if hadValue {
					if err := os.Setenv(key, oldValue); err != nil {
						t.Fatalf("restore %s: %v", key, err)
					}
					return
				}
				if err := os.Unsetenv(key); err != nil {
					t.Fatalf("clear %s: %v", key, err)
				}
			}
		}(key, oldValue, hadValue))
	}
}

func writeDotEnv(t *testing.T, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(".", ".env"), []byte(strings.TrimSpace(content)+"\n"), 0600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
}
