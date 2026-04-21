package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_NAME", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("OPENAI_API_KEY", "")

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
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_NAME", "orch-test")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("OPENAI_API_KEY", "secret")

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
}

func TestLoadReturnsErrorForInvalidHTTPPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
}
