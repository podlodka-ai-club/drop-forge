## 1. Producer Trailer Foundation

- [x] 1.1 Create `internal/agentmeta/trailer.go` with `Producer` struct (`By`, `Model`, `Stage`), `Stage` enum (`StageProposal`, `StageApply`, `StageArchive`), `AppendTrailer`, `ParseTrailer`, and `ErrTrailerNotFound` exported error.
- [x] 1.2 Add table-driven tests in `internal/agentmeta/trailer_test.go` covering: full round-trip parse, case-insensitive keys, missing trailer, unknown stage, ignoring unrelated trailers, preserving existing trailer block when appending.
- [x] 1.3 Verify `go test ./internal/agentmeta/...` passes.

## 2. Wire Producer Trailer Into Existing Runners

- [x] 2.1 Add `Producer agentmeta.Producer` field to `proposalrunner.Runner`; before `git commit -m`, when `Producer` is non-zero, replace the message with `agentmeta.AppendTrailer(message, Producer)`.
- [x] 2.2 Add a test in `internal/proposalrunner/runner_test.go` that sets `Producer` and asserts the captured `git commit -m <msg>` argument contains `Produced-By:`, `Produced-Model:`, and `Produced-Stage: proposal`.
- [x] 2.3 Repeat 2.1 and 2.2 for `internal/applyrunner/runner.go` and `internal/applyrunner/runner_test.go` with `StageApply`.
- [x] 2.4 Repeat 2.1 and 2.2 for `internal/archiverunner/runner.go` and `internal/archiverunner/runner_test.go` with `StageArchive`.
- [x] 2.5 Verify `go test ./internal/proposalrunner/... ./internal/applyrunner/... ./internal/archiverunner/...` passes.

## 3. Configuration Extensions

- [x] 3.1 Add `NeedProposalAIReviewStateID`, `NeedCodeAIReviewStateID`, `NeedArchiveAIReviewStateID` fields to `config.LinearTaskManagerConfig`.
- [x] 3.2 Add new `config.ReviewRunnerConfig` struct with `PrimarySlot`, `SecondarySlot`, `PrimaryModel`, `SecondaryModel`, `PrimaryExecutorPath`, `SecondaryExecutorPath`, `MaxContextBytes`, `ParseRepairRetries`, `PromptDir`.
- [x] 3.3 Add `Review ReviewRunnerConfig` to top-level `config.Config`; extend `Load()` to parse all new env vars including `REVIEW_MAX_CONTEXT_BYTES` (default 256 KiB) and `REVIEW_PARSE_REPAIR_RETRIES` (default 1).
- [x] 3.4 Implement `(ReviewRunnerConfig).Enabled(LinearTaskManagerConfig) bool` returning true only when all three AI-review state IDs and both reviewer slots are populated.
- [x] 3.5 Extend `LinearTaskManagerConfig.Validate()` with all-or-nothing rule for the three AI-review state IDs (any partial configuration returns a contextual error).
- [x] 3.6 Extend `LinearTaskManagerConfig.ManagedStateIDs()` to also include the three AI-review state IDs when populated, deduplicated.
- [x] 3.7 Add config tests covering: load with all AI-review env vars populated, validate accepts all-empty, validate accepts all-set, validate rejects partial, `ManagedStateIDs` includes AI-review state IDs when set.
- [x] 3.8 Verify `go test ./internal/config/...` passes.

## 4. Task Manager Regression Tests

- [x] 4.1 Add a test in `internal/taskmanager/taskmanager_test.go` asserting that when AI-review state IDs are populated, the fake Linear client receives them in the `stateIDs` argument of `GetTasks`.
- [x] 4.2 Verify `go test ./internal/taskmanager/...` passes.

## 5. ReviewParse Package — JSON Schema And Parser

- [x] 5.1 Create `internal/reviewrunner/reviewparse/parse.go` with `Verdict` enum, `Severity` enum, `Stats`, `Summary`, `Finding` (with nullable `LineStart`/`LineEnd` as `*int`), `Review`, and a `Parse([]byte, agentmeta.Stage) (Review, error)` function.
- [x] 5.2 Create `internal/reviewrunner/reviewparse/categories.go` with stage-to-category-set mapping and `CategoriesForStage` helper.
- [x] 5.3 Add `Parse` validation: unknown verdict, unknown severity, category not in stage enum, mismatched null line range (one set + one null), malformed JSON.
- [x] 5.4 Add table-driven tests in `internal/reviewrunner/reviewparse/parse_test.go` covering valid Proposal review, unknown verdict, unknown severity, out-of-stage category, null line range allowed for general findings, malformed JSON.
- [x] 5.5 Verify `go test ./internal/reviewrunner/reviewparse/...` passes.

## 6. Stage Profiles And Verdict Computation

- [x] 6.1 Create `internal/reviewrunner/stage.go` with `StageProfile { Stage, Categories, PromptName }`, `ProfileFor`, `MustProfile`, and per-stage profiles for Proposal/Apply/Archive listing exact category enums from the spec.
- [x] 6.2 Add `ComputeVerdict([]reviewparse.Finding) reviewparse.Verdict` that returns `blocked` if any blocker, `needs-work` if any major and no blocker, otherwise `ship-ready`.
- [x] 6.3 Add `SeverityIcon(reviewparse.Severity) string` returning `🛑`/`⚠️`/`💡`/`🪶`.
- [x] 6.4 Add tests in `internal/reviewrunner/stage_test.go` for category membership per stage, verdict computation in three cases, and severity icons for all four levels.
- [x] 6.5 Verify `go test ./internal/reviewrunner/...` passes (excluding tests that depend on later tasks).

## 7. Targets Collection

- [x] 7.1 Create `internal/reviewrunner/targets.go` with `Target { Path, Content, Truncated }`, `TargetInput { Stage, CloneDir, MaxBytes, ChangePath, Diff }`, and `CollectTargets` dispatching by stage.
- [x] 7.2 Implement Proposal collection: walk `<CloneDir>/<ChangePath>` for `.md` files, sort, read content; respect `MaxBytes` budget with deterministic truncation, marking truncated entries with `Truncated: true`.
- [x] 7.3 Implement Apply collection: include diff plus the corresponding OpenSpec change context.
- [x] 7.4 Implement Archive collection: include diff plus archived spec files.
- [x] 7.5 Add tests in `internal/reviewrunner/targets_test.go` for: Proposal reads all change files, MaxBytes triggers truncation with marking, Apply combines diff and change context, Archive combines diff and archived files.
- [x] 7.6 Verify `go test ./internal/reviewrunner/...` passes.

## 8. Prompt Templates And Renderer

- [x] 8.1 Create `internal/reviewrunner/prompts/proposal_review.tmpl` with role text in Russian, English machine keywords, embedded JSON schema, fix_prompt rules, and `{{ .Categories }}` / `{{ .Targets }}` Go template loops.
- [x] 8.2 Create `internal/reviewrunner/prompts/apply_review.tmpl` mirroring 8.1 with Apply role text and Apply categories.
- [x] 8.3 Create `internal/reviewrunner/prompts/archive_review.tmpl` mirroring 8.1 with Archive role text and Archive categories.
- [x] 8.4 Create `internal/reviewrunner/prompt.go` with `RenderPrompt(in PromptInput, overrideDir string) (string, error)` using `embed.FS` for default templates and reading override-dir files when present.
- [x] 8.5 Add tests in `internal/reviewrunner/prompt_test.go` covering: rendered prompt contains producer/reviewer/stage/targets/categories; rendering fails for unknown stage.
- [x] 8.6 Verify `go test ./internal/reviewrunner/...` passes.

## 9. PR Commenter

- [x] 9.1 Create `internal/reviewrunner/prcommenter/commenter.go` with `PostReviewInput` struct, `PostReviewResult { Skipped }`, `PRCommenter` interface, and `MarkerFor(reviewer, stage, sha)` helper.
- [x] 9.2 Create `internal/reviewrunner/prcommenter/format.go` implementing `FormatSummaryBody` (with marker as first line, walkthrough, severity-stats line, sorted findings) and `FormatInlineBody` (with severity icon, `[review by ...]` prefix, message, and `<details>🤖 Prompt for AI Agent</details>` block).
- [x] 9.3 Add tests in `internal/reviewrunner/prcommenter/format_test.go` covering: summary body contains marker/verdict/findings/file-line-ref; inline body has details block, fix prompt text, and `[review by ...]` prefix.
- [x] 9.4 Create `internal/reviewrunner/prcommenter/gh.go` implementing `GHPostReviewCommenter` that uses `commandrunner.Runner` to call `gh api` for both `GET /reviews` (idempotency check) and `POST /reviews` (atomic publish).
- [x] 9.5 Add tests in `internal/reviewrunner/prcommenter/gh_test.go` covering: `Skipped=true` when marker already exists, single atomic POST when marker absent (assert payload `event=COMMENT`, `commit_id`, summary body marker, comments array length matches inline-eligible findings).
- [x] 9.6 Verify `go test ./internal/reviewrunner/prcommenter/...` passes.

## 10. Reviewer Slot Selection

- [x] 10.1 Create `internal/reviewrunner/reviewer.go` with `Reviewer { Slot, Model, ExecutorPath, ProducerUnknown }`, `SelectReviewer(config.ReviewRunnerConfig, agentmeta.Producer) (Reviewer, error)`, and exported `ErrUnknownProducerSlot`.
- [x] 10.2 Implement opposite-slot rule: empty `Producer.By` falls back to secondary slot with `ProducerUnknown=true`; producer matching primary returns secondary; producer matching secondary returns primary; unknown slot returns `ErrUnknownProducerSlot`.
- [x] 10.3 Add tests in `internal/reviewrunner/reviewer_test.go` for all four cases.
- [x] 10.4 Verify `go test ./internal/reviewrunner/...` passes.

## 11. Review Agent Executor

- [x] 11.1 Create `internal/reviewrunner/agent_executor.go` with `AgentExecutionInput { Prompt, CloneDir, TempDir, Stdout, Stderr }`, `AgentExecutionResult { FinalMessage }`, and `AgentExecutor` interface mirroring the existing runner pattern.
- [x] 11.2 Create `internal/reviewrunner/codex_executor.go` with `CodexCLIExecutor { Command, CodexPath, Model, Service }` invoking `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone> --model <model> -` with prompt on stdin.
- [x] 11.3 Verify `go build ./internal/reviewrunner/...` succeeds.

## 12. Review Runner Orchestration

- [x] 12.1 Create `internal/reviewrunner/runner.go` with `Runner` struct (`Config`, `ProposalCfg`, `Command`, `Executors map[string]AgentExecutor`, `Commenter`, `Service`, `Stdout`, `Stderr`, `MkdirTemp`, `RemoveAll`) and `ReviewInput { Stage, Identifier, Title, BranchName, PRNumber, RepoOwner, RepoName, PRURL }`.
- [x] 12.2 Implement `(*Runner).Run(ctx, ReviewInput) (Result, error)` performing: validate input, mkdir temp, `git clone --branch`, `git rev-parse HEAD` and `git log -1 --format=%B`, parse trailer, `SelectReviewer`, look up executor by slot, detect change path via `git diff --name-only`, capture full diff, `CollectTargets`, `RenderPrompt`, `executeWithRepair` (one repair retry on parse failure), build tripwires (producer-unknown + truncated targets), call `Commenter.PostReview`, log `review.skipped_idempotent` or `review.publish ok`.
- [x] 12.3 Implement `executeWithRepair`: first call to executor; on parse failure call again with repair prompt that includes the parse error verbatim; on second failure return contextual error.
- [x] 12.4 Add helpers `gitClone`, `readHead`, `gitDiff`, `detectChangePath` using `commandrunner` and pure-Go path manipulation.
- [x] 12.5 Add table-driven tests in `internal/reviewrunner/runner_test.go` using fake `commandrunner`, fake `AgentExecutor`, fake `PRCommenter`, covering: happy path publishes review and selects opposite slot; missing trailer falls back to secondary and adds tripwire; invalid JSON triggers exactly one repair and then succeeds; second invalid JSON returns error and does not call commenter; commenter `Skipped=true` propagates as `Result.Skipped=true`.
- [x] 12.6 Verify `go test ./internal/reviewrunner/...` passes.

## 13. CoreOrch Integration

- [x] 13.1 Add `coreorch.ReviewRunner` interface with `Run(ctx, reviewrunner.ReviewInput) (reviewrunner.Result, error)`.
- [x] 13.2 Extend `coreorch.Config` with `NeedProposalAIReviewStateID`, `NeedCodeAIReviewStateID`, `NeedArchiveAIReviewStateID`, and `AIReviewEnabled bool`.
- [x] 13.3 Add `ReviewRunner ReviewRunner` field to `coreorch.Orchestrator`.
- [x] 13.4 Add `coreorch.BuildReviewInput(task taskmanager.Task, stage agentmeta.Stage) (reviewrunner.ReviewInput, error)` that resolves owner/repo/PR-number from `task.PullRequests[0].URL` (parsing `https://github.com/<owner>/<repo>/pull/<n>`).
- [x] 13.5 Add `processReviewTask` and `routeReview` helpers; extend `RunProposalsOnce` switch with three new cases routing AI-review states to `routeReview` with the matching stage and human-review target.
- [x] 13.6 Update `processProposalTask` / `processApplyTask` / `processArchiveTask` to choose target state via feature flag: when `AIReviewEnabled`, move to `Need*AIReviewStateID`; otherwise keep existing `Need*ReviewStateID`.
- [x] 13.7 Update `(orch).validate()`: when `AIReviewEnabled` is true, require `ReviewRunner != nil` and all three AI-review state IDs to be non-empty; when false, no extra requirements.
- [x] 13.8 Add tests in `internal/coreorch/orchestrator_test.go` for: AI-review proposal task routes to `ReviewRunner` with `StageProposal` and final move target equals `NeedProposalReviewStateID`; AI-review disabled skips the AI-review case and producer routes still move tasks to `Need*Review` directly; producer route with AI-review enabled moves task to `Need*AIReview`.
- [x] 13.9 Verify `go test ./internal/coreorch/...` passes.

## 14. CLI Wiring

- [x] 14.1 Add `singleReviewRunner` interface and extend `appDeps` with `newReviewRunner func(cfg config.Config, logOut io.Writer) singleReviewRunner` and updated `newProposalOrchestrator` signature including the `ReviewRunner` parameter.
- [x] 14.2 Update existing `newProposalRunner`/`newApplyRunner`/`newArchiveRunner` factories to accept `cfg config.Config`; inside each, set `runner.Producer = agentmeta.Producer{ By: cfg.Review.PrimarySlot, Model: cfg.Review.PrimaryModel, Stage: <stage> }` when `cfg.Review.PrimarySlot` is non-empty (today both producers are wired to PRIMARY slot; future producer rotation reuses this same hook).
- [x] 14.3 Implement `defaultDeps().newReviewRunner` that returns nil when `cfg.Review.Enabled(cfg.TaskManager)` is false; otherwise builds an executor map keyed by `PrimarySlot`/`SecondarySlot` (today both are `reviewrunner.CodexCLIExecutor`), constructs `prcommenter.GHPostReviewCommenter`, and returns a wired `*reviewrunner.Runner`.
- [x] 14.4 Update `runWithDeps` to call `deps.newReviewRunner` and pass the result (or nil) to `newProposalOrchestrator` along with `cfg.Review.Enabled(cfg.TaskManager)` value to populate `coreorch.Config.AIReviewEnabled`.
- [x] 14.5 Update existing `cmd/orchv3/main_test.go` to inject a stub `newReviewRunner` returning nil so legacy tests keep passing; add a new test asserting that with AI-review env populated the orchestrator receives a non-nil `ReviewRunner` and `AIReviewEnabled=true`.
- [x] 14.6 Verify `go test ./...` passes.

## 15. Documentation And Configuration Files

- [x] 15.1 Append a "Cross-Agent Review Stage" block to `.env.example` listing the three AI-review state IDs, two reviewer slots (role/model/executor path), and three runtime knobs (`REVIEW_MAX_CONTEXT_BYTES`, `REVIEW_PARSE_REPAIR_RETRIES`, `REVIEW_PROMPT_DIR`) all without default values.
- [x] 15.2 Insert a "Целевой Поток Review-Stage" section into `architecture.md` between the Archive section and "Границы Ответственности", describing the eight-step review flow.
- [x] 15.3 Append `internal/reviewrunner` and `internal/agentmeta` mappings to the "Маппинг На Текущий Код" section of `architecture.md`.
- [x] 15.4 Add a paragraph to `docs/proposal-runner.md` explaining that with the AI-review feature configured, producer-runners hand off to AI review before reaching human review, and apply/archive use the same pattern.
- [x] 15.5 Run `go fmt ./...` and `go test ./...` end-to-end and resolve any remaining issues.
