## 1. Configuration

- [ ] 1.1 Add a proposal polling interval field to centralized config with `time.Duration` parsing and validation.
- [ ] 1.2 Add the polling interval key to `.env.example` without a default value or secret.
- [ ] 1.3 Extend config tests for valid interval, omitted interval default, and invalid interval failure.

## 2. CoreOrch Polling Loop

- [ ] 2.1 Add a long-running proposal polling loop API that reuses the existing one-pass proposal processing.
- [ ] 2.2 Make the loop wait for the configured interval between iterations and exit cleanly when context is cancelled.
- [ ] 2.3 Log loop startup, iteration outcomes, iteration errors, and cancellation using the existing structured logger.
- [ ] 2.4 Ensure non-cancellation iteration errors are logged and do not stop the loop.

## 3. CLI Behavior

- [ ] 3.1 Wire default CLI startup to build `TaskManager`, proposal runner, logger, and run the proposal polling loop.
- [ ] 3.2 Remove direct proposal runner execution from arbitrary CLI arguments.
- [ ] 3.3 Remove direct proposal runner execution from piped `stdin`.
- [ ] 3.4 Remove the public one-shot `orchestrate-proposals` CLI command path or make it return a usage error.
- [ ] 3.5 Add signal-aware context cancellation for graceful shutdown of the polling loop.

## 4. Documentation and Architecture

- [ ] 4.1 Update `README.md` to document Linear-driven monitoring as the primary run mode and remove args/stdin manual proposal examples.
- [ ] 4.2 Update detailed proposal/orchestration docs so they no longer describe manual task-description execution as supported behavior.
- [ ] 4.3 Update `architecture.md` mapping to describe the long-running `CoreOrch` polling loop and changed CLI boundary.

## 5. Tests and Verification

- [ ] 5.1 Add `internal/coreorch` tests for repeated iterations, interval wait, cancellation, and continuing after iteration errors.
- [ ] 5.2 Update CLI tests for default polling loop wiring, args/stdin usage errors, removed one-shot command behavior, and graceful cancellation.
- [ ] 5.3 Keep existing proposal processing tests passing for one-pass task filtering, PR attachment, and state transition behavior.
- [ ] 5.4 Run `go fmt ./...`.
- [ ] 5.5 Run `go test ./...`.
