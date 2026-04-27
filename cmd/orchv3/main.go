package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"orchv3/internal/applyrunner"
	"orchv3/internal/archiverunner"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/coreorch"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/steplog"
	"orchv3/internal/taskmanager"
)

type singleProposalRunner interface {
	Run(ctx context.Context, input proposalrunner.ProposalInput) (string, error)
}

type singleApplyRunner interface {
	Run(ctx context.Context, input applyrunner.ApplyInput) error
}

type singleArchiveRunner interface {
	Run(ctx context.Context, input archiverunner.ArchiveInput) error
}

type proposalMonitor interface {
	RunProposalsLoop(ctx context.Context, interval time.Duration) error
}

type appDeps struct {
	loadConfig              func() (config.Config, error)
	buildLogger             func(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error)
	newProposalRunner       func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleProposalRunner
	newApplyRunner          func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleApplyRunner
	newArchiveRunner        func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleArchiveRunner
	newTaskManager          func(cfg config.LinearTaskManagerConfig, logOut io.Writer) coreorch.TaskManager
	newProposalOrchestrator func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, logOut io.Writer) proposalMonitor
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer) int {
	return runWithDeps(args, stdin, stdout, stderr, defaultDeps())
}

func runWithDeps(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer, deps appDeps) int {
	_ = stdout

	if err := rejectManualProposalInput(args, stdin); err != nil {
		steplog.New(stderr).Errorf("cli", "%v", err)
		return 1
	}

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

	taskManager := deps.newTaskManager(cfg.TaskManager, logOut)
	proposalRunner := deps.newProposalRunner(cfg.ProposalRunner, cfg.AppName, logOut)
	applyRunner := deps.newApplyRunner(cfg.ProposalRunner, cfg.AppName, logOut)
	archiveRunner := deps.newArchiveRunner(cfg.ProposalRunner, cfg.AppName, logOut)
	orchestrator := deps.newProposalOrchestrator(cfg, taskManager, proposalRunner, applyRunner, archiveRunner, logOut)
	logger.Infof(
		"cli",
		"%s starting orchestration monitor in %s on port %d interval=%s",
		cfg.AppName,
		cfg.AppEnv,
		cfg.HTTPPort,
		cfg.ProposalPollInterval,
	)
	if err := orchestrator.RunProposalsLoop(context.Background(), cfg.ProposalPollInterval); err != nil {
		logger.Errorf("cli", "run orchestration monitor: %v", err)
		return 1
	}

	return 0
}

func rejectManualProposalInput(args []string, stdin *os.File) error {
	if len(args) > 0 {
		return fmt.Errorf("usage error: manual proposal execution was removed; run without arguments to start the proposal monitor")
	}

	stat, err := stdin.Stat()
	if err != nil {
		return fmt.Errorf("stat stdin: %w", err)
	}

	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil
	}

	content, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	if strings.TrimSpace(string(content)) != "" {
		return fmt.Errorf("usage error: manual proposal execution was removed; run without stdin input to start the proposal monitor")
	}

	return nil
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
		newApplyRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleApplyRunner {
			runner := applyrunner.New(cfg)
			runner.Service = service
			runner.Stdout = logOut
			runner.Stderr = logOut
			runner.Command = commandrunner.ExecRunner{LogWriter: logOut}
			return runner
		},
		newArchiveRunner: func(cfg config.ProposalRunnerConfig, service string, logOut io.Writer) singleArchiveRunner {
			runner := archiverunner.New(cfg)
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
		newProposalOrchestrator: func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, logOut io.Writer) proposalMonitor {
			return &coreorch.Orchestrator{
				Config: coreorch.Config{
					ReadyToProposeStateID:      cfg.TaskManager.ReadyToProposeStateID,
					ProposingInProgressStateID: cfg.TaskManager.ProposingInProgressStateID,
					NeedProposalReviewStateID:  cfg.TaskManager.NeedProposalReviewStateID,
					ReadyToCodeStateID:         cfg.TaskManager.ReadyToCodeStateID,
					CodeInProgressStateID:      cfg.TaskManager.CodeInProgressStateID,
					NeedCodeReviewStateID:      cfg.TaskManager.NeedCodeReviewStateID,
					ReadyToArchiveStateID:      cfg.TaskManager.ReadyToArchiveStateID,
					ArchivingInProgressStateID: cfg.TaskManager.ArchivingInProgressStateID,
					NeedArchiveReviewStateID:   cfg.TaskManager.NeedArchiveReviewStateID,
				},
				TaskManager:    tasks,
				ProposalRunner: proposalRunner,
				ApplyRunner:    applyRunner,
				ArchiveRunner:  archiveRunner,
				Service:        cfg.AppName,
				LogWriter:      logOut,
			}
		},
	}
}
