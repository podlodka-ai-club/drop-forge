package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var envKeys = []string{
	"APP_ENV",
	"APP_NAME",
	"LOG_LEVEL",
	"HTTP_PORT",
	"OPENAI_API_KEY",
	"DROP_FORGE_REPOSITORY_URL",
	"DROP_FORGE_BASE_BRANCH",
	"DROP_FORGE_REMOTE_NAME",
	"PROPOSAL_BRANCH_PREFIX",
	"PROPOSAL_PR_TITLE_PREFIX",
	"DROP_FORGE_CLEANUP_TEMP",
	"DROP_FORGE_POLL_INTERVAL",
	"DROP_FORGE_GIT_PATH",
	"DROP_FORGE_CODEX_PATH",
	"DROP_FORGE_GH_PATH",
	"PROPOSAL_REPOSITORY_URL",
	"PROPOSAL_BASE_BRANCH",
	"PROPOSAL_REMOTE_NAME",
	"PROPOSAL_CLEANUP_TEMP",
	"PROPOSAL_POLL_INTERVAL",
	"PROPOSAL_GIT_PATH",
	"PROPOSAL_CODEX_PATH",
	"PROPOSAL_GH_PATH",
	"LINEAR_API_URL",
	"LINEAR_API_TOKEN",
	"LINEAR_PROJECT_ID",
	"LINEAR_STATE_READY_TO_PROPOSE_ID",
	"LINEAR_STATE_READY_TO_CODE_ID",
	"LINEAR_STATE_READY_TO_ARCHIVE_ID",
	"LINEAR_STATE_PROPOSING_IN_PROGRESS_ID",
	"LINEAR_STATE_CODE_IN_PROGRESS_ID",
	"LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID",
	"LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID",
	"LINEAR_STATE_NEED_CODE_REVIEW_ID",
	"LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID",
	"LOGSTASH_ADDR",
	"LOGSTASH_BUFFER_SIZE",
	"LOGSTASH_DIAL_TIMEOUT",
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

	if cfg.ProposalRunner.BaseBranch != defaultDropForgeBaseBranch {
		t.Fatalf("ProposalRunner.BaseBranch = %q, want %q", cfg.ProposalRunner.BaseBranch, defaultDropForgeBaseBranch)
	}

	if cfg.ProposalRunner.CleanupTemp {
		t.Fatal("ProposalRunner.CleanupTemp = true, want false")
	}

	if cfg.ProposalRunner.GitPath != defaultDropForgeGitPath {
		t.Fatalf("ProposalRunner.GitPath = %q, want %q", cfg.ProposalRunner.GitPath, defaultDropForgeGitPath)
	}

	if cfg.TaskManager.APIURL != defaultLinearAPIURL {
		t.Fatalf("TaskManager.APIURL = %q, want %q", cfg.TaskManager.APIURL, defaultLinearAPIURL)
	}
	if cfg.ProposalPollInterval != defaultDropForgePollInterval {
		t.Fatalf("ProposalPollInterval = %v, want %v", cfg.ProposalPollInterval, defaultDropForgePollInterval)
	}
	if cfg.Logstash.Addr != "" {
		t.Fatalf("Logstash.Addr = %q, want empty by default", cfg.Logstash.Addr)
	}
	if cfg.Logstash.BufferSize != defaultLogstashBufferSize {
		t.Fatalf("Logstash.BufferSize = %d, want %d", cfg.Logstash.BufferSize, defaultLogstashBufferSize)
	}
	if cfg.Logstash.DialTimeout != defaultLogstashDialTimeout {
		t.Fatalf("Logstash.DialTimeout = %v, want %v", cfg.Logstash.DialTimeout, defaultLogstashDialTimeout)
	}
}

func TestLoadReadsLogstashEnvironment(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_ADDR", "127.0.0.1:5000")
	t.Setenv("LOGSTASH_BUFFER_SIZE", "2048")
	t.Setenv("LOGSTASH_DIAL_TIMEOUT", "750ms")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Logstash.Addr != "127.0.0.1:5000" {
		t.Fatalf("Addr = %q", cfg.Logstash.Addr)
	}
	if cfg.Logstash.BufferSize != 2048 {
		t.Fatalf("BufferSize = %d", cfg.Logstash.BufferSize)
	}
	if cfg.Logstash.DialTimeout != 750*time.Millisecond {
		t.Fatalf("DialTimeout = %v", cfg.Logstash.DialTimeout)
	}
}

func TestLoadReturnsErrorForInvalidLogstashBufferSize(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_BUFFER_SIZE", "abc")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForNonPositiveLogstashBufferSize(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_BUFFER_SIZE", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForInvalidLogstashDialTimeout(t *testing.T) {
	isolateEnv(t)
	t.Setenv("LOGSTASH_DIAL_TIMEOUT", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	isolateEnv(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_NAME", "orch-test")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("OPENAI_API_KEY", "secret")
	t.Setenv("DROP_FORGE_REPOSITORY_URL", "git@github.com:example/project.git")
	t.Setenv("DROP_FORGE_BASE_BRANCH", "develop")
	t.Setenv("DROP_FORGE_REMOTE_NAME", "upstream")
	t.Setenv("PROPOSAL_BRANCH_PREFIX", "automation/proposal")
	t.Setenv("PROPOSAL_PR_TITLE_PREFIX", "Spec:")
	t.Setenv("DROP_FORGE_CLEANUP_TEMP", "true")
	t.Setenv("DROP_FORGE_POLL_INTERVAL", "1m")
	t.Setenv("DROP_FORGE_GIT_PATH", "/usr/local/bin/git")
	t.Setenv("DROP_FORGE_CODEX_PATH", "/usr/local/bin/codex")
	t.Setenv("DROP_FORGE_GH_PATH", "/usr/local/bin/gh")
	t.Setenv("LINEAR_API_URL", "https://linear.example/graphql")
	t.Setenv("LINEAR_API_TOKEN", "linear-token")
	t.Setenv("LINEAR_PROJECT_ID", "project-123")
	t.Setenv("LINEAR_STATE_READY_TO_PROPOSE_ID", "state-propose")
	t.Setenv("LINEAR_STATE_READY_TO_CODE_ID", "state-code")
	t.Setenv("LINEAR_STATE_READY_TO_ARCHIVE_ID", "state-archive")
	t.Setenv("LINEAR_STATE_PROPOSING_IN_PROGRESS_ID", "state-proposing-progress")
	t.Setenv("LINEAR_STATE_CODE_IN_PROGRESS_ID", "state-code-progress")
	t.Setenv("LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID", "state-archiving-progress")
	t.Setenv("LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID", "state-proposal-review")
	t.Setenv("LINEAR_STATE_NEED_CODE_REVIEW_ID", "state-code-review")
	t.Setenv("LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID", "state-archive-review")

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
	if cfg.ProposalPollInterval != time.Minute {
		t.Fatalf("ProposalPollInterval = %v, want %v", cfg.ProposalPollInterval, time.Minute)
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

	taskManagerCfg := cfg.TaskManager
	if taskManagerCfg.APIURL != "https://linear.example/graphql" {
		t.Fatalf("APIURL = %q", taskManagerCfg.APIURL)
	}
	if taskManagerCfg.APIToken != "linear-token" {
		t.Fatalf("APIToken = %q", taskManagerCfg.APIToken)
	}
	if taskManagerCfg.ProjectID != "project-123" {
		t.Fatalf("ProjectID = %q", taskManagerCfg.ProjectID)
	}
	if taskManagerCfg.ReadyToProposeStateID != "state-propose" {
		t.Fatalf("ReadyToProposeStateID = %q", taskManagerCfg.ReadyToProposeStateID)
	}
	if taskManagerCfg.ReadyToCodeStateID != "state-code" {
		t.Fatalf("ReadyToCodeStateID = %q", taskManagerCfg.ReadyToCodeStateID)
	}
	if taskManagerCfg.ReadyToArchiveStateID != "state-archive" {
		t.Fatalf("ReadyToArchiveStateID = %q", taskManagerCfg.ReadyToArchiveStateID)
	}
	if taskManagerCfg.ProposingInProgressStateID != "state-proposing-progress" {
		t.Fatalf("ProposingInProgressStateID = %q", taskManagerCfg.ProposingInProgressStateID)
	}
	if taskManagerCfg.CodeInProgressStateID != "state-code-progress" {
		t.Fatalf("CodeInProgressStateID = %q", taskManagerCfg.CodeInProgressStateID)
	}
	if taskManagerCfg.ArchivingInProgressStateID != "state-archiving-progress" {
		t.Fatalf("ArchivingInProgressStateID = %q", taskManagerCfg.ArchivingInProgressStateID)
	}
	if taskManagerCfg.NeedProposalReviewStateID != "state-proposal-review" {
		t.Fatalf("NeedProposalReviewStateID = %q", taskManagerCfg.NeedProposalReviewStateID)
	}
	if taskManagerCfg.NeedCodeReviewStateID != "state-code-review" {
		t.Fatalf("NeedCodeReviewStateID = %q", taskManagerCfg.NeedCodeReviewStateID)
	}
	if taskManagerCfg.NeedArchiveReviewStateID != "state-archive-review" {
		t.Fatalf("NeedArchiveReviewStateID = %q", taskManagerCfg.NeedArchiveReviewStateID)
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
	t.Setenv("DROP_FORGE_CLEANUP_TEMP", "sometimes")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForInvalidProposalPollInterval(t *testing.T) {
	isolateEnv(t)
	t.Setenv("DROP_FORGE_POLL_INTERVAL", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadReturnsErrorForNonPositiveProposalPollInterval(t *testing.T) {
	isolateEnv(t)
	t.Setenv("DROP_FORGE_POLL_INTERVAL", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}

func TestLoadRejectsLegacySharedProposalKeys(t *testing.T) {
	tests := []struct {
		legacy string
		value  string
		want   string
	}{
		{legacy: "PROPOSAL_REPOSITORY_URL", value: "git@github.com:example/legacy.git", want: "DROP_FORGE_REPOSITORY_URL"},
		{legacy: "PROPOSAL_POLL_INTERVAL", value: "1m", want: "DROP_FORGE_POLL_INTERVAL"},
		{legacy: "PROPOSAL_GIT_PATH", value: "/usr/bin/git", want: "DROP_FORGE_GIT_PATH"},
		{legacy: "PROPOSAL_CODEX_PATH", value: "/usr/bin/codex", want: "DROP_FORGE_CODEX_PATH"},
		{legacy: "PROPOSAL_GH_PATH", value: "/usr/bin/gh", want: "DROP_FORGE_GH_PATH"},
	}

	for _, tt := range tests {
		t.Run(tt.legacy, func(t *testing.T) {
			isolateEnv(t)
			t.Setenv(tt.legacy, tt.value)

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want non-nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %q, want replacement %q", err.Error(), tt.want)
			}
		})
	}
}

func TestLoadReadsDotEnvWithGodotenvSyntax(t *testing.T) {
	isolateEnv(t)
	writeDotEnv(t, `
APP_NAME="orch app" # inline comment
DROP_FORGE_REPOSITORY_URL='git@github.com:example/from-dotenv.git'
DROP_FORGE_BASE_BRANCH=feature/base
DROP_FORGE_CLEANUP_TEMP=true
LINEAR_PROJECT_ID='project-from-dotenv'
LINEAR_STATE_READY_TO_PROPOSE_ID="state-propose"
LINEAR_STATE_READY_TO_CODE_ID='state-code'
LINEAR_STATE_READY_TO_ARCHIVE_ID="state-archive"
LINEAR_STATE_PROPOSING_IN_PROGRESS_ID="state-proposing-progress"
LINEAR_STATE_CODE_IN_PROGRESS_ID='state-code-progress'
LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID="state-archiving-progress"
LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID="state-proposal-review"
LINEAR_STATE_NEED_CODE_REVIEW_ID="state-code-review"
LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID="state-archive-review"
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

	if cfg.TaskManager.ProjectID != "project-from-dotenv" {
		t.Fatalf("ProjectID = %q", cfg.TaskManager.ProjectID)
	}
	if cfg.TaskManager.ReadyToArchiveStateID != "state-archive" {
		t.Fatalf("ReadyToArchiveStateID = %q", cfg.TaskManager.ReadyToArchiveStateID)
	}
	if cfg.TaskManager.ProposingInProgressStateID != "state-proposing-progress" {
		t.Fatalf("ProposingInProgressStateID = %q", cfg.TaskManager.ProposingInProgressStateID)
	}
	if cfg.TaskManager.NeedCodeReviewStateID != "state-code-review" {
		t.Fatalf("NeedCodeReviewStateID = %q", cfg.TaskManager.NeedCodeReviewStateID)
	}

}

func TestLoadReadsDotEnvFromParentDirectory(t *testing.T) {
	isolateEnv(t)
	writeDotEnv(t, `
APP_NAME=orch-parent-env
LINEAR_PROJECT_ID=project-from-parent-dotenv
LINEAR_STATE_READY_TO_CODE_ID=state-code-from-parent
`)

	if err := os.MkdirAll(filepath.Join("internal", "taskmanager"), 0755); err != nil {
		t.Fatalf("create nested package dir: %v", err)
	}
	if err := os.Chdir(filepath.Join("internal", "taskmanager")); err != nil {
		t.Fatalf("chdir nested package dir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.AppName != "orch-parent-env" {
		t.Fatalf("AppName = %q, want parent .env value", cfg.AppName)
	}
	if cfg.TaskManager.ProjectID != "project-from-parent-dotenv" {
		t.Fatalf("ProjectID = %q, want parent .env value", cfg.TaskManager.ProjectID)
	}
	if cfg.TaskManager.ReadyToCodeStateID != "state-code-from-parent" {
		t.Fatalf("ReadyToCodeStateID = %q, want parent .env value", cfg.TaskManager.ReadyToCodeStateID)
	}
}

func TestLoadProcessEnvironmentOverridesDotEnv(t *testing.T) {
	isolateEnv(t)
	writeDotEnv(t, `
DROP_FORGE_REPOSITORY_URL=git@github.com:example/from-dotenv.git
DROP_FORGE_BASE_BRANCH=main
LINEAR_PROJECT_ID=project-from-dotenv
`)
	t.Setenv("DROP_FORGE_REPOSITORY_URL", "git@github.com:example/from-process.git")
	t.Setenv("LINEAR_PROJECT_ID", "project-from-process")

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

	if cfg.TaskManager.ProjectID != "project-from-process" {
		t.Fatalf("ProjectID = %q, want process env value", cfg.TaskManager.ProjectID)
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
			wantErr: "DROP_FORGE_REPOSITORY_URL",
		},
		{
			name: "missing git path",
			mutate: func(cfg *ProposalRunnerConfig) {
				cfg.GitPath = " "
			},
			wantErr: "DROP_FORGE_GIT_PATH",
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

func TestLinearTaskManagerConfigValidate(t *testing.T) {
	valid := LinearTaskManagerConfig{
		APIURL:                     defaultLinearAPIURL,
		APIToken:                   "linear-token",
		ProjectID:                  "project-123",
		ReadyToProposeStateID:      "state-propose",
		ReadyToCodeStateID:         "state-code",
		ReadyToArchiveStateID:      "state-archive",
		ProposingInProgressStateID: "state-proposing-progress",
		CodeInProgressStateID:      "state-code-progress",
		ArchivingInProgressStateID: "state-archiving-progress",
		NeedProposalReviewStateID:  "state-proposal-review",
		NeedCodeReviewStateID:      "state-code-review",
		NeedArchiveReviewStateID:   "state-archive-review",
	}

	tests := []struct {
		name    string
		mutate  func(*LinearTaskManagerConfig)
		wantErr string
	}{
		{
			name: "valid",
		},
		{
			name: "missing project",
			mutate: func(cfg *LinearTaskManagerConfig) {
				cfg.ProjectID = " "
			},
			wantErr: "LINEAR_PROJECT_ID",
		},
		{
			name: "missing ready to code state",
			mutate: func(cfg *LinearTaskManagerConfig) {
				cfg.ReadyToCodeStateID = " "
			},
			wantErr: "LINEAR_STATE_READY_TO_CODE_ID",
		},
		{
			name: "missing proposing in progress state",
			mutate: func(cfg *LinearTaskManagerConfig) {
				cfg.ProposingInProgressStateID = " "
			},
			wantErr: "LINEAR_STATE_PROPOSING_IN_PROGRESS_ID",
		},
		{
			name: "missing code in progress state",
			mutate: func(cfg *LinearTaskManagerConfig) {
				cfg.CodeInProgressStateID = " "
			},
			wantErr: "LINEAR_STATE_CODE_IN_PROGRESS_ID",
		},
		{
			name: "missing archiving in progress state",
			mutate: func(cfg *LinearTaskManagerConfig) {
				cfg.ArchivingInProgressStateID = " "
			},
			wantErr: "LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID",
		},
		{
			name: "missing proposal review state",
			mutate: func(cfg *LinearTaskManagerConfig) {
				cfg.NeedProposalReviewStateID = " "
			},
			wantErr: "LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID",
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

func TestLinearTaskManagerConfigManagedStatesDeduplicatesAndTrims(t *testing.T) {
	cfg := LinearTaskManagerConfig{
		ReadyToProposeStateID:      " state-propose ",
		ReadyToCodeStateID:         "state-code",
		ReadyToArchiveStateID:      "state-propose",
		ProposingInProgressStateID: "state-proposing-progress",
		CodeInProgressStateID:      "state-code-progress",
		ArchivingInProgressStateID: "state-archiving-progress",
	}

	got := cfg.ManagedStateIDs()
	want := []string{"state-propose", "state-code"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ManagedStateIDs() = %#v, want %#v", got, want)
	}
}

func TestLinearTaskManagerConfigManagedStatesExcludeInProgressStates(t *testing.T) {
	cfg := LinearTaskManagerConfig{
		ReadyToProposeStateID:      "state-propose",
		ReadyToCodeStateID:         "state-code",
		ReadyToArchiveStateID:      "state-archive",
		ProposingInProgressStateID: "state-proposing-progress",
		CodeInProgressStateID:      "state-code-progress",
		ArchivingInProgressStateID: "state-archiving-progress",
	}

	got := cfg.ManagedStateIDs()
	want := []string{"state-propose", "state-code", "state-archive"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ManagedStateIDs() = %#v, want %#v", got, want)
	}

	for _, forbidden := range []string{
		cfg.ProposingInProgressStateID,
		cfg.CodeInProgressStateID,
		cfg.ArchivingInProgressStateID,
	} {
		for _, stateID := range got {
			if stateID == forbidden {
				t.Fatalf("ManagedStateIDs() includes in-progress state %q in %#v", forbidden, got)
			}
		}
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
