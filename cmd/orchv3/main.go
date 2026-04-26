package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/coreorch"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
	"orchv3/internal/taskmanager"
)

const orchestrateProposalsCommand = "orchestrate-proposals"

type singleProposalRunner interface {
	Run(ctx context.Context, taskDescription string) (string, error)
}

type proposalOrchestrator interface {
	RunProposalsOnce(ctx context.Context) error
}

type appDeps struct {
	loadConfig              func() (config.Config, error)
	buildLogger             func(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error)
	newProposalRunner       func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner
	newTaskManager          func(cfg config.LinearTaskManagerConfig, logOut io.Writer) coreorch.TaskManager
	newProposalOrchestrator func(cfg config.Config, tasks coreorch.TaskManager, runner coreorch.ProposalRunner, logOut io.Writer) proposalOrchestrator
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer) int {
	return runWithDeps(args, stdin, stdout, stderr, defaultDeps())
}

func runWithDeps(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer, deps appDeps) int {
	cfg, err := deps.loadConfig()
	if err != nil {
		steplog.New(stderr).Errorf("cli", "load config: %v", err)
		return 1
	}

	logger, logOut, closer, err := deps.buildLogger(stderr, cfg, os.Stderr)
	if err != nil {
		steplog.New(stderr).Errorf("cli", "build logger: %v", err)
		return 1
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	if isOrchestrateProposalsCommand(args) {
		taskManager := deps.newTaskManager(cfg.TaskManager, logOut)
		runner := deps.newProposalRunner(cfg.ProposalRunner, cfg.AppName, logOut)
		orchestrator := deps.newProposalOrchestrator(cfg, taskManager, runner, logOut)
		if err := orchestrator.RunProposalsOnce(context.Background()); err != nil {
			logger.Errorf("cli", "run proposal orchestration: %v", err)
			return 1
		}

		return 0
	}

	taskDescription, err := readTaskDescription(args, stdin)
	if err != nil {
		logger.Errorf("cli", "read task description: %v", err)
		return 1
	}

	if taskDescription != "" {
		runner := deps.newProposalRunner(cfg.ProposalRunner, cfg.AppName, logOut)
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

func isOrchestrateProposalsCommand(args []string) bool {
	return len(args) == 1 && args[0] == orchestrateProposalsCommand
}

func defaultDeps() appDeps {
	return appDeps{
		loadConfig:  config.Load,
		buildLogger: buildLogger,
		newProposalRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner {
			runner := proposalrunner.New(cfg)
			runner.Service = service
			runner.Stdout = logOut
			runner.Stderr = logOut
			runner.Command = commandrunner.ExecRunner{LogWriter: logOut}
			return runner
		},
		newTaskManager: func(cfg config.LinearTaskManagerConfig, logOut io.Writer) coreorch.TaskManager {
			manager := taskmanager.New(cfg)
			manager.LogWriter = logOut
			return manager
		},
		newProposalOrchestrator: func(cfg config.Config, tasks coreorch.TaskManager, runner coreorch.ProposalRunner, logOut io.Writer) proposalOrchestrator {
			return &coreorch.Orchestrator{
				Config: coreorch.Config{
					ReadyToProposeStateID:     cfg.TaskManager.ReadyToProposeStateID,
					NeedProposalReviewStateID: cfg.TaskManager.NeedProposalReviewStateID,
				},
				TaskManager:    tasks,
				ProposalRunner: runner,
				Service:        cfg.AppName,
				LogWriter:      logOut,
			}
		},
	}
}
