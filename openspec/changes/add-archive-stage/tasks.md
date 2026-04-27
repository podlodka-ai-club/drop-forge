## 1. Archive Input and Contracts

- [ ] 1.1 Add an Archive runner input type with task identity, title, agent prompt, PR URL, and optional branch name.
- [ ] 1.2 Add an Archive runner interface to `internal/coreorch`.
- [ ] 1.3 Add `BuildArchiveInput` that derives branch source from task pull request metadata and returns a contextual error when branch source is missing.
- [ ] 1.4 Add unit tests for Archive input title fallback, prompt content, PR URL source, branch source, and missing branch source.

## 2. Archive Runner

- [ ] 2.1 Create `internal/archiverunner` using the Apply runner lifecycle as the baseline.
- [ ] 2.2 Implement Archive input validation and repository command config validation.
- [ ] 2.3 Implement temporary clone, branch resolution from branch name or PR URL, checkout, OpenSpec Archive agent execution, git status validation, commit, and push.
- [ ] 2.4 Build an Archive-specific Codex prompt that requires the `openspec-archive-change` skill and fails on ambiguous active changes.
- [ ] 2.5 Add unit tests for command order, temp cleanup/preserve behavior, PR URL branch resolution, empty git status failure, commit message, and command error wrapping.
- [ ] 2.6 Keep Archive runner tests isolated with fake command runner and fake agent executor, without network, git, GitHub CLI, or Codex CLI.

## 3. Orchestration Flow

- [ ] 3.1 Extend `coreorch.Config` and `Orchestrator` wiring with ready-to-archive, archiving-in-progress, and need-archive-review state IDs.
- [ ] 3.2 Route tasks in `LINEAR_STATE_READY_TO_ARCHIVE_ID` to the Archive runner in the same orchestration pass as proposal and Apply tasks.
- [ ] 3.3 Move Archive tasks to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` before runner execution.
- [ ] 3.4 Move Archive tasks to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` only after Archive runner success.
- [ ] 3.5 Add failure-path tests for missing branch source, archiving-in-progress transition failure, Archive runner failure, and archive-review transition failure.
- [ ] 3.6 Add structured log coverage for Archive task processing, skips, success, and failure context.

## 4. Runtime Wiring and Docs

- [ ] 4.1 Wire the real Archive runner into `cmd/orchv3` default runtime.
- [ ] 4.2 Verify existing Linear archive state config is loaded and listed in `.env.example`; add new keys only if implementation introduces new runtime parameters.
- [ ] 4.3 Update `README.md`, `docs/proposal-runner.md`, `docs/linear-task-manager.md`, and `architecture.md` where they describe the end-to-end orchestration lifecycle.
- [ ] 4.4 Keep public CLI behavior as continuous orchestration monitoring, without adding a manual archive command.

## 5. Verification

- [ ] 5.1 Run `go fmt ./...`.
- [ ] 5.2 Run `go test ./...`.
- [ ] 5.3 Run `openspec status --change add-archive-stage`.
- [ ] 5.4 Confirm the change is apply-ready and all OpenSpec artifacts exist.
