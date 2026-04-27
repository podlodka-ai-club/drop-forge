## ADDED Requirements

### Requirement: Orchestration pass runs ready tasks concurrently with a limit
The system SHALL execute ready-to-propose, ready-to-code, and ready-to-archive task runs in separate goroutines during one orchestration pass, while enforcing a configured maximum number of concurrently active task runs.

#### Scenario: Two ready tasks run concurrently by default
- **WHEN** one orchestration pass receives at least two runnable tasks across proposal, apply, or archive states
- **THEN** the system starts up to two task runs concurrently when no explicit concurrency limit is configured

#### Scenario: Concurrency limit is enforced
- **WHEN** one orchestration pass receives more runnable tasks than the configured concurrency limit
- **THEN** the system does not run more than the configured number of task runs at the same time
- **AND** remaining runnable tasks wait until a running task completes

#### Scenario: Per-task transition order is preserved
- **WHEN** a proposal, apply, or archive task run executes concurrently with another task run
- **THEN** the system preserves the required state transition and runner execution order inside that individual task run

#### Scenario: Skipped tasks do not consume concurrency slots
- **WHEN** one orchestration pass receives tasks whose states are not ready-to-propose, ready-to-code, or ready-to-archive
- **THEN** the system logs those tasks as skipped
- **AND** the skipped tasks do not occupy task run concurrency capacity

### Requirement: Orchestration pass aggregates concurrent task failures
The system SHALL allow independent task runs in the same orchestration pass to complete even if another task run fails, and SHALL return an aggregated contextual error after all started task runs have finished.

#### Scenario: One task failure does not stop independent task runs
- **WHEN** one task run fails during an orchestration pass that has other runnable tasks
- **THEN** the system records the failing task's contextual error
- **AND** the system continues executing independent task runs within the pass

#### Scenario: Multiple task failures are returned together
- **WHEN** more than one task run fails during the same orchestration pass
- **THEN** the orchestration pass returns an error that preserves the context for each failed task

#### Scenario: Successful concurrent pass returns no error
- **WHEN** every concurrently executed task run succeeds
- **THEN** the orchestration pass completes without returning an error

### Requirement: Concurrent orchestration emits task run logs
The system SHALL log concurrent task run lifecycle events using the existing structured logger format and include enough task identity and route context to distinguish interleaved logs.

#### Scenario: Concurrent task run starts
- **WHEN** a proposal, apply, or archive task run starts in a goroutine
- **THEN** the logs include the orchestration module, task ID, task identifier, and route name

#### Scenario: Concurrent task run completes
- **WHEN** a proposal, apply, or archive task run completes successfully
- **THEN** the logs include the orchestration module, task ID, task identifier, and route name

#### Scenario: Concurrent task run fails
- **WHEN** a proposal, apply, or archive task run returns an error
- **THEN** the logs include the orchestration module, task ID, task identifier, route name, and failure context

### Requirement: Concurrent task limit is runtime configurable
The system SHALL read the maximum number of concurrent orchestration task runs from `.env` and environment variables using `ORCH_MAX_CONCURRENT_TASKS`, validate that it is a positive integer when present, default to `2` when absent, and keep `.env.example` synchronized without a committed value.

#### Scenario: Concurrency limit is configured
- **WHEN** the environment contains `ORCH_MAX_CONCURRENT_TASKS=3`
- **THEN** the orchestration pass allows at most three task runs to execute concurrently

#### Scenario: Concurrency limit uses default
- **WHEN** `ORCH_MAX_CONCURRENT_TASKS` is absent
- **THEN** the orchestration pass allows at most two task runs to execute concurrently

#### Scenario: Concurrency limit is invalid
- **WHEN** `ORCH_MAX_CONCURRENT_TASKS` is not a positive integer
- **THEN** configuration loading returns an error before orchestration monitoring starts

#### Scenario: Environment example lists concurrency limit
- **WHEN** a developer opens `.env.example`
- **THEN** it includes `ORCH_MAX_CONCURRENT_TASKS` without a default value

## MODIFIED Requirements

### Requirement: Existing proposal runner is used as the proposal executor

The system SHALL execute proposals by calling the existing proposal runner contract with one prepared `ProposalInput` per eligible proposal task and SHALL not change the proposal runner's internal git, Codex, PR, or comment workflow as part of proposal orchestration.

#### Scenario: Proposal runner is called for eligible task

- **WHEN** a ready-to-propose task is processed
- **THEN** the orchestration stage calls the proposal runner with the prepared `ProposalInput`

#### Scenario: Multiple proposal tasks are processed concurrently within the task limit

- **WHEN** multiple ready-to-propose tasks are returned in one orchestration pass
- **THEN** the orchestration stage may run proposal task executions concurrently
- **AND** the orchestration stage does not exceed the configured concurrent task limit

### Requirement: Archive orchestration uses ready-to-archive tasks
The system SHALL provide an Archive orchestration stage that loads managed tasks through `TaskManager` and processes tasks whose workflow state ID matches `LINEAR_STATE_READY_TO_ARCHIVE_ID`.

#### Scenario: Ready-to-archive task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_ARCHIVE_ID`
- **THEN** the orchestration stage treats that task as eligible for Archive execution

#### Scenario: Non-archive managed task is skipped by Archive route
- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-propose or ready-to-code
- **THEN** the Archive route does not call the Archive runner for that task

#### Scenario: Proposal Apply and Archive tasks are processed in one pass
- **WHEN** one orchestration pass receives ready-to-propose, ready-to-code, and ready-to-archive tasks
- **THEN** the system routes each task to the executor matching its current state
- **AND** the system may process routed tasks concurrently within the configured task limit
