## 1. Configuration

- [x] 1.1 Add `ProposingInProgressStateID`, `CodeInProgressStateID`, and `ArchivingInProgressStateID` fields to `LinearTaskManagerConfig`.
- [x] 1.2 Load `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` in `config.Load`.
- [x] 1.3 Require the new in-progress state IDs in `LinearTaskManagerConfig.Validate`.
- [x] 1.4 Update `.env.example` with the new Linear state keys without default values.
- [x] 1.5 Update config tests to cover successful loading and missing required in-progress state IDs.

## 2. Task Manager Behavior

- [x] 2.1 Keep `ManagedStateIDs()` limited to ready-to-propose, ready-to-code, and ready-to-archive states.
- [x] 2.2 Add or update task manager/config tests proving in-progress states are not included in managed task selection.

## 3. Proposal Orchestration

- [x] 3.1 Add `ProposingInProgressStateID` to `coreorch.Config` and validate it.
- [x] 3.2 Wire `ProposingInProgressStateID` from loaded config into the CLI orchestration mode.
- [x] 3.3 Move each selected ready-to-propose task to `Proposing in Progress` before building/running the proposal.
- [x] 3.4 Preserve the existing success flow after proposal execution: attach PR, then move to `Need Proposal Review`.
- [x] 3.5 Return contextual errors when the initial in-progress transition fails and do not run the proposal runner in that case.
- [x] 3.6 Update failure handling tests so runner/PR failures leave the task out of proposal review after the initial in-progress move.

## 4. Verification

- [x] 4.1 Run `go fmt ./...`.
- [x] 4.2 Run `go test ./...`.
- [x] 4.3 Run `openspec validate update-taskmanager-progress-columns --strict`.
