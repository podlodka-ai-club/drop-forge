## 1. CoreOrch Package

- [x] 1.1 Create `internal/coreorch` with `TaskManager` and `ProposalRunner` interfaces matching the existing manager/runner methods needed by proposal-stage.
- [x] 1.2 Implement `RunProposalsOnce(ctx)` to load tasks through `TaskManager`, filter by `ReadyToProposeStateID`, and process eligible tasks sequentially.
- [x] 1.3 Implement deterministic proposal input formatting from task identifier, title, description, and comments.
- [x] 1.4 Implement success flow: run proposal, attach returned PR URL, then move task to `NeedProposalReviewStateID`.
- [x] 1.5 Implement failure handling with contextual errors and structured logs without moving failed tasks to review state.

## 2. CLI Integration

- [x] 2.1 Add an explicit CLI mode for one proposal orchestration pass, preserving the existing direct task-description proposal runner mode.
- [x] 2.2 Wire config, `taskmanager.Manager`, `proposalrunner.Runner`, command runner, and logger output for the orchestration CLI mode.
- [x] 2.3 Keep existing stdout behavior for direct proposal runner mode and avoid printing orchestration-only data as plain text.

## 3. Tests

- [x] 3.1 Add `internal/coreorch` unit tests for ready-state filtering, no-ready-task behavior, sequential processing, and proposal input formatting.
- [x] 3.2 Add `internal/coreorch` unit tests for runner failure, PR attachment failure, and proposal review transition failure.
- [x] 3.3 Add CLI tests that verify orchestration mode wiring and that direct proposal runner behavior remains compatible.
- [x] 3.4 Confirm task manager payload fields used by orchestration are covered by existing tests or add focused tests if coverage is missing.

## 4. Documentation And Verification

- [x] 4.1 Update `architecture.md` mapping to mark `CoreOrch` as implemented in the new package and CLI mode.
- [x] 4.2 Run `go fmt ./...`.
- [x] 4.3 Run `go test ./...`.
- [x] 4.4 Run `openspec validate add-proposal-orchestrator --strict`.
