## ADDED Requirements

### Requirement: CLI handles shutdown signals gracefully

The system SHALL start the orchestration monitor with a root context that is cancelled when the process receives `SIGINT` or `SIGTERM`.

#### Scenario: SIGINT requests monitor shutdown

- **WHEN** the CLI receives `SIGINT` after the orchestration monitor has started
- **THEN** the root context passed to the monitor is cancelled
- **AND** the CLI logs a structured shutdown request event

#### Scenario: SIGTERM requests monitor shutdown

- **WHEN** the CLI receives `SIGTERM` after the orchestration monitor has started
- **THEN** the root context passed to the monitor is cancelled
- **AND** the CLI logs a structured shutdown request event

#### Scenario: Shutdown completes successfully

- **WHEN** the orchestration monitor exits because the root context was cancelled by a shutdown signal
- **THEN** the CLI logs a structured shutdown completion event
- **AND** the CLI exits with success.

### Requirement: Orchestration monitor does not start work after cancellation

The system SHALL stop starting new orchestration monitor iterations after its context is cancelled.

#### Scenario: Context is cancelled before next iteration

- **WHEN** the orchestration monitor context is cancelled while waiting between polling iterations
- **THEN** the monitor exits without starting another orchestration pass

#### Scenario: Context is cancelled before first iteration

- **WHEN** the orchestration monitor starts with an already cancelled context
- **THEN** the monitor exits without loading managed tasks

### Requirement: In-flight orchestration pass completes under cancellation

The system SHALL pass context cancellation to already-started task processing and wait for all task goroutines in the current orchestration pass to finish before returning from that pass.

#### Scenario: Cancellation reaches active runners

- **WHEN** the orchestration pass has already started proposal, apply, or archive runner work and the context is cancelled
- **THEN** each active runner receives the cancelled context through its existing `Run` call

#### Scenario: Current pass waits for active tasks

- **WHEN** the context is cancelled while multiple task goroutines are active
- **THEN** the orchestration pass waits for every started goroutine to finish
- **AND** the pass returns any aggregated errors reported by those goroutines
