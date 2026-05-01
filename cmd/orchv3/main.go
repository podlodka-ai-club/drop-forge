package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"orchv3/internal/agentmeta"
	"orchv3/internal/applyrunner"
	"orchv3/internal/archiverunner"
	"orchv3/internal/commandrunner"
	"orchv3/internal/config"
	"orchv3/internal/coreorch"
	"orchv3/internal/events"
	telegramnotifications "orchv3/internal/notifications/telegram"
	"orchv3/internal/proposalrunner"
	"orchv3/internal/reviewrunner"
	"orchv3/internal/reviewrunner/prcommenter"
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

type singleReviewRunner interface {
	Run(ctx context.Context, input reviewrunner.ReviewInput) (reviewrunner.Result, error)
}

type proposalMonitor interface {
	RunProposalsLoop(ctx context.Context, interval time.Duration) error
}

type appDeps struct {
	loadConfig              func() (config.Config, error)
	buildLogger             func(stderr io.Writer, cfg config.Config, warnOut io.Writer) (steplog.Logger, io.Writer, io.Closer, error)
	newProposalRunner       func(cfg config.Config, logOut io.Writer) singleProposalRunner
	newApplyRunner          func(cfg config.Config, logOut io.Writer) singleApplyRunner
	newArchiveRunner        func(cfg config.Config, logOut io.Writer) singleArchiveRunner
	newReviewRunner         func(cfg config.Config, logOut io.Writer) singleReviewRunner
	newEventPublisher       func(telegramCfg config.TelegramConfig, taskManagerCfg config.LinearTaskManagerConfig, logOut io.Writer) (events.Publisher, error)
	newTaskManager          func(cfg config.LinearTaskManagerConfig, logOut io.Writer, publisher events.Publisher) coreorch.TaskManager
	newProposalOrchestrator func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, reviewRunner coreorch.ReviewRunner, logOut io.Writer) proposalMonitor
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer) int {
	return runWithDeps(args, stdin, stdout, stderr, defaultDeps())
}

func runWithDeps(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer, deps appDeps) int {
	_ = stdout
	earlyLogger := steplog.New(consoleLogWriter(stderr, isTerminalWriter))

	if err := rejectManualProposalInput(args, stdin); err != nil {
		earlyLogger.Errorf("cli", "%v", err)
		return 1
	}

	cfg, err := deps.loadConfig()
	if err != nil {
		earlyLogger.Errorf("cli", "load config: %v", err)
		return 1
	}

	logger, logOut, closer, err := deps.buildLogger(stderr, cfg, os.Stderr)
	if err != nil {
		earlyLogger.Errorf("cli", "build logger: %v", err)
		return 1
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	publisher, err := deps.newEventPublisher(cfg.Telegram, cfg.TaskManager, logOut)
	if err != nil {
		logger.Errorf("cli", "build event publisher: %v", err)
		return 1
	}

	taskManager := deps.newTaskManager(cfg.TaskManager, logOut, publisher)
	proposalRunner := deps.newProposalRunner(cfg, logOut)
	applyRunner := deps.newApplyRunner(cfg, logOut)
	archiveRunner := deps.newArchiveRunner(cfg, logOut)
	reviewRunnerSingle := deps.newReviewRunner(cfg, logOut)

	// Keep the orchestrator's interface field nil-valued when the factory
	// returned no review runner; assigning a typed-nil singleReviewRunner to
	// coreorch.ReviewRunner would produce a non-nil interface holding a nil
	// pointer, defeating the orchestrator's nil check.
	var reviewRunner coreorch.ReviewRunner
	if reviewRunnerSingle != nil {
		reviewRunner = reviewRunnerSingle
	}

	orchestrator := deps.newProposalOrchestrator(cfg, taskManager, proposalRunner, applyRunner, archiveRunner, reviewRunner, logOut)
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
		newProposalRunner: func(cfg config.Config, logOut io.Writer) singleProposalRunner {
			runner := proposalrunner.New(cfg.ProposalRunner)
			runner.Service = cfg.AppName
			runner.Stdout = logOut
			runner.Stderr = logOut
			runner.Command = commandrunner.ExecRunner{LogWriter: logOut}
			if cfg.Review.PrimarySlot != "" {
				runner.Producer = agentmeta.Producer{
					By:    cfg.Review.PrimarySlot,
					Model: cfg.Review.PrimaryModel,
					Stage: agentmeta.StageProposal,
				}
			}
			return runner
		},
		newApplyRunner: func(cfg config.Config, logOut io.Writer) singleApplyRunner {
			runner := applyrunner.New(cfg.ProposalRunner)
			runner.Service = cfg.AppName
			runner.Stdout = logOut
			runner.Stderr = logOut
			runner.Command = commandrunner.ExecRunner{LogWriter: logOut}
			if cfg.Review.PrimarySlot != "" {
				runner.Producer = agentmeta.Producer{
					By:    cfg.Review.PrimarySlot,
					Model: cfg.Review.PrimaryModel,
					Stage: agentmeta.StageApply,
				}
			}
			return runner
		},
		newArchiveRunner: func(cfg config.Config, logOut io.Writer) singleArchiveRunner {
			runner := archiverunner.New(cfg.ProposalRunner)
			runner.Service = cfg.AppName
			runner.Stdout = logOut
			runner.Stderr = logOut
			runner.Command = commandrunner.ExecRunner{LogWriter: logOut}
			if cfg.Review.PrimarySlot != "" {
				runner.Producer = agentmeta.Producer{
					By:    cfg.Review.PrimarySlot,
					Model: cfg.Review.PrimaryModel,
					Stage: agentmeta.StageArchive,
				}
			}
			return runner
		},
		newReviewRunner: func(cfg config.Config, logOut io.Writer) singleReviewRunner {
			if !cfg.Review.Enabled(cfg.TaskManager) {
				return nil
			}
			cmd := commandrunner.ExecRunner{LogWriter: logOut}
			// When PrimarySlot == SecondarySlot (single-Codex deployment), the
			// map collapses to one entry. SelectReviewer still routes correctly
			// because the slot key is shared; the secondary model/path silently
			// overwrites the primary in the map. Once a true second agent is
			// registered (e.g., Claude), the slots will differ and the map will
			// hold two distinct executors.
			executors := map[string]reviewrunner.AgentExecutor{
				cfg.Review.PrimarySlot: reviewrunner.CodexCLIExecutor{
					Command:   cmd,
					CodexPath: cfg.Review.PrimaryExecutorPath,
					Model:     cfg.Review.PrimaryModel,
					Service:   cfg.AppName,
				},
				cfg.Review.SecondarySlot: reviewrunner.CodexCLIExecutor{
					Command:   cmd,
					CodexPath: cfg.Review.SecondaryExecutorPath,
					Model:     cfg.Review.SecondaryModel,
					Service:   cfg.AppName,
				},
			}
			commenter := prcommenter.GHPostReviewCommenter{
				Command: cmd,
				GHPath:  cfg.ProposalRunner.GHPath,
				Service: cfg.AppName,
				Stdout:  logOut,
				Stderr:  logOut,
			}
			return &reviewrunner.Runner{
				Config:      cfg.Review,
				ProposalCfg: cfg.ProposalRunner,
				Command:     cmd,
				Executors:   executors,
				Commenter:   commenter,
				Service:     cfg.AppName,
				Stdout:      logOut,
				Stderr:      logOut,
			}
		},
		newEventPublisher: buildEventPublisher,
		newTaskManager: func(cfg config.LinearTaskManagerConfig, logOut io.Writer, publisher events.Publisher) coreorch.TaskManager {
			manager := taskmanager.New(cfg)
			manager.LogWriter = logOut
			manager.Publisher = publisher
			return manager
		},
		newProposalOrchestrator: func(cfg config.Config, tasks coreorch.TaskManager, proposalRunner coreorch.ProposalRunner, applyRunner coreorch.ApplyRunner, archiveRunner coreorch.ArchiveRunner, reviewRunner coreorch.ReviewRunner, logOut io.Writer) proposalMonitor {
			return &coreorch.Orchestrator{
				Config: coreorch.Config{
					ReadyToProposeStateID:       cfg.TaskManager.ReadyToProposeStateID,
					ProposingInProgressStateID:  cfg.TaskManager.ProposingInProgressStateID,
					NeedProposalReviewStateID:   cfg.TaskManager.NeedProposalReviewStateID,
					NeedProposalAIReviewStateID: cfg.TaskManager.NeedProposalAIReviewStateID,
					ReadyToCodeStateID:          cfg.TaskManager.ReadyToCodeStateID,
					CodeInProgressStateID:       cfg.TaskManager.CodeInProgressStateID,
					NeedCodeReviewStateID:       cfg.TaskManager.NeedCodeReviewStateID,
					NeedCodeAIReviewStateID:     cfg.TaskManager.NeedCodeAIReviewStateID,
					ReadyToArchiveStateID:       cfg.TaskManager.ReadyToArchiveStateID,
					ArchivingInProgressStateID:  cfg.TaskManager.ArchivingInProgressStateID,
					NeedArchiveReviewStateID:    cfg.TaskManager.NeedArchiveReviewStateID,
					NeedArchiveAIReviewStateID:  cfg.TaskManager.NeedArchiveAIReviewStateID,
					AIReviewEnabled:             cfg.Review.Enabled(cfg.TaskManager),
				},
				TaskManager:    tasks,
				ProposalRunner: proposalRunner,
				ApplyRunner:    applyRunner,
				ArchiveRunner:  archiveRunner,
				ReviewRunner:   reviewRunner,
				Service:        cfg.AppName,
				LogWriter:      logOut,
			}
		},
	}
}

func buildEventPublisher(cfg config.TelegramConfig, taskManagerCfg config.LinearTaskManagerConfig, logOut io.Writer) (events.Publisher, error) {
	dispatcher := events.NewDispatcher()
	if !cfg.Enabled {
		return dispatcher, nil
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	dispatcher.Subscribe(events.TaskStatusChangedType, telegramnotifications.NewNotifier(
		cfg,
		taskManagerCfg.NeedProposalReviewStateID,
		taskManagerCfg.NeedCodeReviewStateID,
		taskManagerCfg.NeedArchiveReviewStateID,
	))
	steplog.New(writerOrDiscard(logOut)).Infof("cli", "telegram notifications enabled")
	return dispatcher, nil
}

func writerOrDiscard(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
