## 1. TaskManager Payload

- [x] 1.1 Extend `taskmanager.Task` with deterministic pull request branch source data for Apply.
- [x] 1.2 Update Linear managed issue queries to return Pull Request attachment URLs for managed tasks.
- [x] 1.3 Add Linear client tests for ready-to-code tasks with PR attachment, without PR attachment, and with deterministic multiple PR handling.
- [x] 1.4 Update task manager unit tests to cover the extended payload while preserving existing state selection behavior.

## 2. Apply Runner

- [x] 2.1 Add an Apply runner input type with task identity, title, agent prompt, PR URL, and optional branch name.
- [x] 2.2 Implement Apply runner validation for repository config and required branch source.
- [x] 2.3 Implement temporary clone, branch resolution from PR URL or branch name, checkout, OpenSpec Apply agent execution, git status validation, commit, and push.
- [x] 2.4 Add unit tests for command order, temp cleanup/preserve behavior, PR URL branch resolution, empty git status failure, and command error wrapping.
- [x] 2.5 Keep Apply runner testable through fake command runner and fake agent executor without network, git, GitHub CLI, or Codex CLI.

## 3. Core Orchestration

- [x] 3.1 Add Apply runner interface and Apply input builder to `internal/coreorch`.
- [x] 3.2 Extend orchestrator config with ready-to-code, code-in-progress, and need-code-review state IDs.
- [x] 3.3 Route managed tasks by state in one orchestration pass: proposal tasks to proposal runner, code tasks to Apply runner, other managed states to skip logs.
- [x] 3.4 Implement Apply state transitions: move to code-in-progress before execution and need-code-review only after successful Apply.
- [x] 3.5 Add failure-path tests for missing branch source, code-in-progress transition failure, Apply runner failure, and code-review transition failure.
- [x] 3.6 Update monitor naming/log expectations where needed so the runtime is no longer proposal-only in behavior.

## 4. CLI Wiring And Documentation

- [x] 4.1 Wire the real Apply runner into `cmd/orchv3` default runtime.
- [x] 4.2 Update CLI tests to assert apply-related config is passed into the orchestrator.
- [x] 4.3 Update `architecture.md` to document Apply-stage interactions and current code mapping.
- [x] 4.4 Update relevant docs if runtime wording currently says the monitor only executes proposal tasks.

## 5. Verification

- [x] 5.1 Run `go fmt ./...`.
- [x] 5.2 Run `go test ./...`.
- [x] 5.3 Run `openspec status --change add-apply-stage` and verify the change is apply-ready.
