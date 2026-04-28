## 1. Event Payload

- [x] 1.1 Extend `events.TaskStatusChanged` with optional `PullRequestURL` and `PullRequestBranch` fields.
- [x] 1.2 Add or update event tests to verify required fields remain stable and optional PR context can be carried without changing the event type.

## 2. Task Manager Publishing

- [x] 2.1 Add an expanded task status transition input or method in `taskmanager` that accepts task identifier, title, state names, PR URL, and PR branch while preserving existing `MoveTask` behavior.
- [x] 2.2 Publish the expanded context in `task.status_changed` only after Linear accepts the transition.
- [x] 2.3 Add task manager tests for basic `MoveTask`, expanded review move context, failed move without event, and publish failure without transition rollback.

## 3. Orchestration Context

- [x] 3.1 Update `coreorch` proposal review transition to pass task identifier, title, target review state, and the newly created proposal PR URL.
- [x] 3.2 Update `coreorch` code review and archive review transitions to pass task identifier, title, target review state, and deterministic PR URL or branch source from the task payload.
- [x] 3.3 Add orchestration tests proving review transitions publish enough context for Telegram and in-progress transitions do not require PR context.

## 4. Telegram Filtering and Message Format

- [x] 4.1 Wire Telegram notifier with the configured review state IDs without adding new environment variables.
- [x] 4.2 Filter `task.status_changed` events in the Telegram subscriber so only proposal/code/archive review target states send messages.
- [x] 4.3 Update Telegram message formatting to include readable task reference, title, target state, and PR URL when available, with stable ID fallbacks.
- [x] 4.4 Add Telegram notifier tests for review-state delivery, non-review suppression, PR URL rendering, branch-only fallback, and legacy ID fallback.

## 5. Verification

- [x] 5.1 Run `go fmt ./...`.
- [x] 5.2 Run `go test ./...`.
- [x] 5.3 Run `openspec status --change improve-telegram-review-notifications` and confirm the change is apply-ready.
