package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/proposalrunner"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	taskDescription, err := readTaskDescription(os.Args[1:], os.Stdin)
	if err != nil {
		log.Fatalf("read task description: %v", err)
	}

	if taskDescription != "" {
		runner := proposalrunner.New(cfg.ProposalRunner)
		runner.Stdout = os.Stderr
		runner.Stderr = os.Stderr
		runner.Command = commandrunner.ExecRunner{LogWriter: os.Stderr}

		prURL, err := runner.Run(context.Background(), taskDescription)
		if err != nil {
			log.Fatalf("run proposal workflow: %v", err)
		}

		fmt.Fprintln(os.Stdout, prURL)
		return
	}

	log.Printf(
		"%s starting in %s on port %d",
		cfg.AppName,
		cfg.AppEnv,
		cfg.HTTPPort,
	)
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
