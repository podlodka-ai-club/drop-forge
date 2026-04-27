## 1. CoreOrch Concurrency Tests

- [ ] 1.1 Add a unit test proving a ready-to-propose task and a ready-to-code task start runner execution concurrently in one `RunProposalsOnce` pass.
- [ ] 1.2 Add a unit test proving multiple ready-to-propose tasks remain sequential within the proposal stage.
- [ ] 1.3 Add a unit test proving multiple ready-to-code tasks remain sequential within the Apply stage.
- [ ] 1.4 Add a unit test proving failure in one stage does not cancel an already running peer stage and the pass returns the failed stage error after both goroutines finish.
- [ ] 1.5 Add a unit test proving proposal and Apply failures from the same pass are both represented in the returned error.

## 2. CoreOrch Implementation

- [ ] 2.1 Refactor `RunProposalsOnce` to group ready-to-propose, ready-to-code, ready-to-archive, and skipped tasks without changing task input loading.
- [ ] 2.2 Implement a small internal helper that runs one stage queue sequentially and reports a contextual stage error.
- [ ] 2.3 Start proposal and Apply stage queues in separate goroutines when they have eligible tasks, and wait for both before returning from the pass.
- [ ] 2.4 Keep Archive task processing sequential and run it outside the proposal/apply concurrent section.
- [ ] 2.5 Aggregate stage errors without dropping context when both proposal and Apply fail.
- [ ] 2.6 Ensure task-level lifecycle ordering inside `processProposalTask` and `processApplyTask` remains unchanged.

## 3. Logging And Test Doubles

- [ ] 3.1 Ensure concurrent stage logs write complete structured events without interleaving partial JSON lines.
- [ ] 3.2 Make `internal/coreorch` test doubles concurrency-safe where concurrent tests share fake task manager or runner state.
- [ ] 3.3 Update existing sequential routing assertions so they verify per-stage ordering instead of global proposal-before-apply ordering.

## 4. Documentation And Verification

- [ ] 4.1 Update `architecture.md` to describe that `CoreOrch` can run proposal and Apply stage executors concurrently while keeping Archive sequential.
- [ ] 4.2 Run `go fmt ./...`.
- [ ] 4.3 Run `go test ./...`.
- [ ] 4.4 Run `openspec status --change add-concurrent-proposal-apply` and confirm the change is apply-ready.
