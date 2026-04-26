## 1. CoreOrch Package

- [ ] 1.1 Create `internal/coreorch` with `TaskManager` and `ProposalRunner` interfaces matching the existing manager/runner methods needed by proposal-stage.
- [ ] 1.2 Implement `RunProposalsOnce(ctx)` to load tasks through `TaskManager`, filter by `ReadyToProposeStateID`, and process eligible tasks sequentially.
- [ ] 1.3 Implement deterministic proposal input formatting from task identifier, title, description, and comments.
- [ ] 1.4 Implement success flow: run proposal, attach returned PR URL, then move task to `NeedProposalReviewStateID`.
- [ ] 1.5 Implement failure handling with contextual errors and structured logs without moving failed tasks to review state.

## 2. CLI Integration

- [ ] 2.1 Add an explicit CLI mode for one proposal orchestration pass, preserving the existing direct task-description proposal runner mode.
- [ ] 2.2 Wire config, `taskmanager.Manager`, `proposalrunner.Runner`, command runner, and logger output for the orchestration CLI mode.
- [ ] 2.3 Keep existing stdout behavior for direct proposal runner mode and avoid printing orchestration-only data as plain text.

## 3. Tests

- [ ] 3.1 Add `internal/coreorch` unit tests for ready-state filtering, no-ready-task behavior, sequential processing, and proposal input formatting.
- [ ] 3.2 Add `internal/coreorch` unit tests for runner failure, PR attachment failure, and proposal review transition failure.
- [ ] 3.3 Add CLI tests that verify orchestration mode wiring and that direct proposal runner behavior remains compatible.
- [ ] 3.4 Confirm task manager payload fields used by orchestration are covered by existing tests or add focused tests if coverage is missing.

## 4. Documentation And Verification

- [ ] 4.1 Update `architecture.md` mapping to mark `CoreOrch` as implemented in the new package and CLI mode.
- [ ] 4.2 Run `go fmt ./...`.
- [ ] 4.3 Run `go test ./...`.
- [ ] 4.4 Run `openspec validate add-proposal-orchestrator --strict`.
