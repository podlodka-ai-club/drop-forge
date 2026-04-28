## 1. CLI Shutdown Wiring

- [ ] 1.1 Add a testable root context factory in `cmd/orchv3` that uses standard-library signal cancellation for `SIGINT` and `SIGTERM`.
- [ ] 1.2 Pass the signal-aware root context into `RunProposalsLoop` instead of `context.Background()`.
- [ ] 1.3 Add structured CLI logs for shutdown request and shutdown completion.

## 2. Orchestration Cancellation Behavior

- [ ] 2.1 Add or adjust `coreorch` tests proving an already cancelled loop exits without loading managed tasks.
- [ ] 2.2 Add or adjust `coreorch` tests proving cancellation during the polling wait exits without starting another pass.
- [ ] 2.3 Add or adjust `coreorch` tests proving cancellation during an active pass is passed to runners and the pass waits for all active goroutines.

## 3. CLI Tests

- [ ] 3.1 Add `cmd/orchv3` tests that trigger the injected shutdown cancel path without sending real OS signals.
- [ ] 3.2 Assert that shutdown caused by the root context returns success and writes JSON shutdown log events.
- [ ] 3.3 Assert that non-cancellation monitor errors still return a non-zero exit code and structured error log.

## 4. Verification

- [ ] 4.1 Run `go fmt ./...`.
- [ ] 4.2 Run `go test ./...`.
- [ ] 4.3 Update `architecture.md` only if implementation changes component boundaries or orchestration responsibilities beyond local CLI/runtime wiring.
