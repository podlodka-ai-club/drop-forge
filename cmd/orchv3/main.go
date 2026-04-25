package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer) int {
	cfg, err := config.Load()
	if err != nil {
		steplog.New(stderr).Errorf("cli", "load config: %v", err)
		return 1
	}

	logger, logOut, closer, err := buildLogger(stderr, cfg, os.Stderr)
	if err != nil {
		steplog.New(stderr).Errorf("cli", "build logger: %v", err)
		return 1
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	taskDescription, err := readTaskDescription(args, stdin)
	if err != nil {
		logger.Errorf("cli", "read task description: %v", err)
		return 1
	}

	if taskDescription != "" {
		runner := proposalrunner.New(cfg.ProposalRunner)
		runner.Service = cfg.AppName
		runner.Stdout = logOut
		runner.Stderr = logOut
		runner.Command = commandrunner.ExecRunner{LogWriter: logOut}

		prURL, err := runner.Run(context.Background(), taskDescription)
		if err != nil {
			logger.Errorf("cli", "run proposal workflow: %v", err)
			return 1
		}

		fmt.Fprintln(stdout, prURL)
		return 0
	}

	logger.Infof(
		"cli",
		"%s starting in %s on port %d",
		cfg.AppName,
		cfg.AppEnv,
		cfg.HTTPPort,
	)
	return 0
}

func readTaskDescription(args []string, stdin *os.File) (string, error) {
	if len(args) > 0 {
		return strings.TrimSpace(strings.Join(args, " ")), nil
	}

	stat, err := stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("stat stdin: %w", err)
	}

	if stat.Mode()&os.ModeCharDevice != 0 {
		return "", nil
	}

	content, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}
