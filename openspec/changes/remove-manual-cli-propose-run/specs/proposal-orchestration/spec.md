## ADDED Requirements

### Requirement: Proposal orchestration runs as a continuous monitor

The system SHALL provide a continuous proposal monitoring loop that repeatedly executes proposal orchestration for tasks in `LINEAR_STATE_READY_TO_PROPOSE_ID` until its context is cancelled.

#### Scenario: Monitor starts proposal polling

- **WHEN** the CLI starts without unsupported manual task input
- **THEN** the system loads config, wires `TaskManager`, proposal runner, logger, and proposal orchestration dependencies
- **AND** the system starts a continuous proposal monitoring loop

#### Scenario: Monitor repeats after successful pass

- **WHEN** a proposal orchestration pass completes successfully
- **THEN** the monitor waits for the configured polling interval
- **AND** the monitor starts the next proposal orchestration pass

#### Scenario: Monitor continues after pass failure

- **WHEN** a proposal orchestration pass returns an error after the monitor has started
- **THEN** the monitor logs a structured error event with the orchestration failure context
- **AND** the monitor waits for the configured polling interval before starting the next pass

#### Scenario: Monitor stops on context cancellation

- **WHEN** the monitor context is cancelled while waiting between passes
- **THEN** the monitor exits without starting another proposal orchestration pass

### Requirement: Proposal polling interval is runtime configurable

The system SHALL read the proposal monitor polling interval from `.env` and environment variables using `PROPOSAL_POLL_INTERVAL`, validate that it is a positive duration, and keep `.env.example` synchronized without a committed value.

#### Scenario: Poll interval is configured

- **WHEN** the environment contains `PROPOSAL_POLL_INTERVAL=1m`
- **THEN** the proposal monitor waits one minute between orchestration passes

#### Scenario: Poll interval uses default

- **WHEN** `PROPOSAL_POLL_INTERVAL` is absent
- **THEN** the proposal monitor uses the code-defined default polling interval

#### Scenario: Poll interval is invalid

- **WHEN** `PROPOSAL_POLL_INTERVAL` is not a valid positive duration
- **THEN** configuration loading returns an error before proposal monitoring starts

#### Scenario: Environment example lists poll interval

- **WHEN** a developer opens `.env.example`
- **THEN** it includes `PROPOSAL_POLL_INTERVAL` without a default value

## MODIFIED Requirements

### Requirement: Proposal orchestration is available from the CLI

The system SHALL expose proposal orchestration as the default CLI runtime by starting a continuous proposal monitoring loop and SHALL reject the removed direct proposal runner mode for task descriptions passed through args or stdin.

#### Scenario: CLI starts proposal monitoring mode

- **WHEN** the user invokes the CLI without a manual proposal task description
- **THEN** the CLI loads config, wires `TaskManager`, proposal runner, logger, and proposal orchestration dependencies
- **AND** the CLI starts the continuous proposal monitoring loop

#### Scenario: CLI direct proposal args are rejected

- **WHEN** the user provides a task description through CLI arguments
- **THEN** the CLI returns a usage error explaining that manual proposal execution was removed
- **AND** the CLI does not call the proposal runner

#### Scenario: CLI direct proposal stdin is rejected

- **WHEN** the user provides a task description through stdin
- **THEN** the CLI returns a usage error explaining that manual proposal execution was removed
- **AND** the CLI does not call the proposal runner

#### Scenario: Legacy one-pass command is not the public runtime

- **WHEN** the user invokes the previous one-pass proposal orchestration command
- **THEN** the CLI rejects the command as unsupported or treats it as invalid manual input
- **AND** the CLI does not run a one-pass orchestration mode as public behavior
