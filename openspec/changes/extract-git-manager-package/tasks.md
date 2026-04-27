## 1. GitManager Package

- [x] 1.1 Create `internal/gitmanager` with config, dependency hooks, workspace model, and default construction from existing proposal runner config values.
- [x] 1.2 Implement logged command execution in `gitmanager` using `internal/commandrunner` and `internal/steplog` with `git` and `github` modules.
- [x] 1.3 Implement workspace lifecycle: temp directory creation, clone into `<temp-dir>/repo`, preserve-by-default cleanup, configured removal, and contextual clone/cleanup errors.
- [x] 1.4 Implement reusable git operations: short status, existing branch checkout, new branch checkout, stage all, commit, and push to configured remote.
- [x] 1.5 Implement reusable GitHub CLI operations: resolve PR head branch, create PR and parse URL, skip empty PR comment, and publish non-empty PR comment.
- [x] 1.6 Add focused `internal/gitmanager` unit tests with fake command runner and fake temp filesystem hooks for success, command failures, PR URL parsing, branch resolution, comment skipping, and cleanup behavior.

## 2. Runner Integration

- [x] 2.1 Update `proposalrunner.Runner` to accept an injectable GitManager dependency while preserving existing `New(cfg)` defaults.
- [x] 2.2 Refactor `proposalrunner.Run` to delegate workspace clone, status, branch checkout, commit/push, PR creation, and final response comment to `gitmanager`.
- [x] 2.3 Update `applyrunner.Runner` to accept an injectable GitManager dependency while preserving existing `New(cfg)` defaults.
- [x] 2.4 Refactor `applyrunner.Run` to delegate workspace clone, optional PR branch resolution, checkout, status, commit, and push to `gitmanager`.
- [x] 2.5 Update `archiverunner.Runner` to accept an injectable GitManager dependency while preserving existing `New(cfg)` defaults.
- [x] 2.6 Refactor `archiverunner.Run` to delegate workspace clone, optional PR branch resolution, checkout, status, commit, and push to `gitmanager`.
- [x] 2.7 Remove duplicated git/gh helper code from runner packages once all callers use `gitmanager`.

## 3. Tests And Documentation

- [x] 3.1 Adapt proposal runner tests to assert runner workflow through fake GitManager and keep coverage for no-change, PR comment, cleanup, and error paths.
- [x] 3.2 Adapt apply runner tests to assert branch-source handling, no-change behavior, commit/push decisions, cleanup, and error paths through fake GitManager.
- [x] 3.3 Adapt archive runner tests to assert branch-source handling, no-change behavior, commit/push decisions, cleanup, and error paths through fake GitManager.
- [x] 3.4 Keep Codex executor tests on `commandrunner` because agent execution remains outside `gitmanager`.
- [x] 3.5 Update `architecture.md` so the current code mapping says `GitManager` is implemented in `internal/gitmanager` and reused by proposal, apply, and archive runner packages.

## 4. Verification

- [x] 4.1 Run `go fmt ./...`.
- [x] 4.2 Run `go test ./...`.
- [x] 4.3 Run OpenSpec validation/status for `extract-git-manager-package` and fix any spec/task formatting issues.
