package config

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultAppEnv   = "development"
	defaultAppName  = "orchv3"
	defaultLogLevel = "debug"
	defaultHTTPPort = 8080
)

type Config struct {
	AppEnv       string
	AppName      string
	LogLevel     string
	HTTPPort     int
	OpenAIAPIKey string
}

func Load() (Config, error) {
	httpPort, err := intFromEnv("HTTP_PORT", defaultHTTPPort)
	if err != nil {
		return Config{}, err
	}

	return Config{
		AppEnv:       stringFromEnv("APP_ENV", defaultAppEnv),
		AppName:      stringFromEnv("APP_NAME", defaultAppName),
		LogLevel:     stringFromEnv("LOG_LEVEL", defaultLogLevel),
		HTTPPort:     httpPort,
		OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
	}, nil
}

func stringFromEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
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
