## ADDED Requirements

### Requirement: Proposal orchestration runs as a polling loop

The system SHALL provide a long-running proposal orchestration loop that repeatedly monitors Linear tasks in `LINEAR_STATE_READY_TO_PROPOSE_ID` and launches proposal processing for eligible tasks.

#### Scenario: Loop processes ready-to-propose tasks
- **WHEN** the polling loop runs an iteration and `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_PROPOSE_ID`
- **THEN** the system runs the existing proposal orchestration processing for that task

#### Scenario: Loop waits between iterations
- **WHEN** a polling iteration completes
- **THEN** the system waits for the configured polling interval before loading tasks again

#### Scenario: Loop continues after iteration failure
- **WHEN** a polling iteration returns an error that is not caused by context cancellation
- **THEN** the system logs the error and continues with the next iteration after the configured polling interval

#### Scenario: Loop exits on cancellation
- **WHEN** the process context is cancelled or an operating system shutdown signal is received
- **THEN** the polling loop exits without starting another proposal processing iteration

### Requirement: Proposal polling interval is runtime-configurable

The system SHALL read the proposal polling interval from centralized runtime configuration and the repository SHALL keep `.env.example` synchronized with the corresponding key without committed values.

#### Scenario: Polling interval is configured
- **WHEN** the environment contains a valid proposal polling interval
- **THEN** the proposal orchestration loop uses that interval between task loading iterations

#### Scenario: Polling interval is omitted
- **WHEN** the environment does not contain a proposal polling interval
- **THEN** the system uses a safe default interval and starts normally

#### Scenario: Polling interval is invalid
- **WHEN** the environment contains an invalid proposal polling interval
- **THEN** configuration loading fails before the polling loop starts

### Requirement: Proposal polling emits structured logs

The system SHALL log polling loop lifecycle, iteration start, iteration completion, iteration failures, and cancellation using the existing structured logger format.

#### Scenario: Loop startup is logged
- **WHEN** the proposal polling loop starts
- **THEN** the logs include a structured event with the orchestration module and configured polling interval

#### Scenario: Iteration outcome is logged
- **WHEN** a polling iteration completes with or without ready-to-propose tasks
- **THEN** the logs include a structured event describing the iteration outcome

#### Scenario: Loop cancellation is logged
- **WHEN** the proposal polling loop exits because the context is cancelled
- **THEN** the logs include a structured event describing the shutdown

## MODIFIED Requirements

### Requirement: Proposal orchestration is available from the CLI

The system SHALL expose the proposal orchestration polling loop as the primary CLI behavior and SHALL reject manual direct proposal execution from task descriptions passed through args or stdin.

#### Scenario: CLI starts proposal polling mode

- **WHEN** the user invokes the CLI without a task description
- **THEN** the CLI loads config, wires `TaskManager`, proposal runner, logger, and proposal polling loop
- **AND** the CLI runs the proposal polling loop until cancellation or process shutdown

#### Scenario: CLI rejects direct proposal arguments

- **WHEN** the user invokes the CLI with arbitrary task description arguments
- **THEN** the CLI returns a usage error
- **AND** the CLI does not call the proposal runner directly

#### Scenario: CLI rejects direct proposal stdin

- **WHEN** the user pipes a task description into the CLI
- **THEN** the CLI returns a usage error
- **AND** the CLI does not call the proposal runner directly

#### Scenario: CLI no longer exposes one-shot proposal orchestration command

- **WHEN** the user invokes the previous one-shot proposal orchestration command
- **THEN** the CLI returns a usage error
- **AND** the CLI does not run a single orchestration pass as a public command
