## 1. Configuration

- [ ] 1.1 Add `ProposingInProgressStateID`, `CodeInProgressStateID`, and `ArchivingInProgressStateID` fields to `LinearTaskManagerConfig`.
- [ ] 1.2 Load `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` in `config.Load`.
- [ ] 1.3 Require the new in-progress state IDs in `LinearTaskManagerConfig.Validate`.
- [ ] 1.4 Update `.env.example` with the new Linear state keys without default values.
- [ ] 1.5 Update config tests to cover successful loading and missing required in-progress state IDs.

## 2. Task Manager Behavior

- [ ] 2.1 Keep `ManagedStateIDs()` limited to ready-to-propose, ready-to-code, and ready-to-archive states.
- [ ] 2.2 Add or update task manager/config tests proving in-progress states are not included in managed task selection.

## 3. Proposal Orchestration

- [ ] 3.1 Add `ProposingInProgressStateID` to `coreorch.Config` and validate it.
- [ ] 3.2 Wire `ProposingInProgressStateID` from loaded config into the CLI orchestration mode.
- [ ] 3.3 Move each selected ready-to-propose task to `Proposing in Progress` before building/running the proposal.
- [ ] 3.4 Preserve the existing success flow after proposal execution: attach PR, then move to `Need Proposal Review`.
- [ ] 3.5 Return contextual errors when the initial in-progress transition fails and do not run the proposal runner in that case.
- [ ] 3.6 Update failure handling tests so runner/PR failures leave the task out of proposal review after the initial in-progress move.

## 4. Verification

- [ ] 4.1 Run `go fmt ./...`.
- [ ] 4.2 Run `go test ./...`.
- [ ] 4.3 Run `openspec validate update-taskmanager-progress-columns --strict`.
