## MODIFIED Requirements

### Requirement: Proposal orchestration runs as a continuous monitor

The system SHALL provide a continuous Drop Forge orchestration monitoring loop that repeatedly executes orchestration for tasks in managed Linear states, including `LINEAR_STATE_READY_TO_PROPOSE_ID`, until its context is cancelled.

#### Scenario: Monitor starts orchestration polling

- **WHEN** the CLI starts without unsupported manual task input
- **THEN** the system loads config, wires `TaskManager`, proposal runner, Apply runner, Archive runner, logger, and orchestration dependencies
- **AND** the system starts a continuous Drop Forge orchestration monitoring loop

#### Scenario: Monitor repeats after successful pass

- **WHEN** an orchestration pass completes successfully
- **THEN** the monitor waits for the configured polling interval
- **AND** the monitor starts the next orchestration pass

#### Scenario: Monitor continues after pass failure

- **WHEN** an orchestration pass returns an error after the monitor has started
- **THEN** the monitor logs a structured error event with the orchestration failure context
- **AND** the monitor waits for the configured polling interval before starting the next pass

#### Scenario: Monitor stops on context cancellation

- **WHEN** the monitor context is cancelled while waiting between passes
- **THEN** the monitor exits without starting another orchestration pass

### Requirement: Proposal polling interval is runtime configurable

The system SHALL read the Drop Forge orchestration monitor polling interval from `.env` and environment variables using `DROP_FORGE_POLL_INTERVAL`, validate that it is a positive duration, and keep `.env.example` synchronized without a committed value.

#### Scenario: Poll interval is configured

- **WHEN** the environment contains `DROP_FORGE_POLL_INTERVAL=1m`
- **THEN** the orchestration monitor waits one minute between orchestration passes

#### Scenario: Poll interval uses default

- **WHEN** `DROP_FORGE_POLL_INTERVAL` is absent
- **THEN** the orchestration monitor uses the code-defined default polling interval

#### Scenario: Poll interval is invalid

- **WHEN** `DROP_FORGE_POLL_INTERVAL` is not a valid positive duration
- **THEN** configuration loading returns an error before orchestration monitoring starts

#### Scenario: Environment example lists poll interval

- **WHEN** a developer opens `.env.example`
- **THEN** it includes `DROP_FORGE_POLL_INTERVAL` without a default value

### Requirement: Proposal orchestration is available from the CLI

The system SHALL expose Drop Forge orchestration as the default CLI runtime by starting a continuous orchestration monitoring loop and SHALL reject the removed direct proposal runner mode for task descriptions passed through args or stdin.

#### Scenario: CLI starts orchestration monitoring mode

- **WHEN** the user invokes the CLI without a manual proposal task description
- **THEN** the CLI loads config, wires `TaskManager`, proposal runner, Apply runner, Archive runner, logger, and orchestration dependencies
- **AND** the CLI starts the continuous Drop Forge orchestration monitoring loop

#### Scenario: CLI direct proposal args are rejected

- **WHEN** the user provides a task description through CLI arguments
- **THEN** the CLI returns a usage error explaining that manual proposal execution is unsupported in Drop Forge
- **AND** the CLI does not call the proposal runner

#### Scenario: CLI direct proposal stdin is rejected

- **WHEN** the user provides a task description through stdin
- **THEN** the CLI returns a usage error explaining that manual proposal execution is unsupported in Drop Forge
- **AND** the CLI does not call the proposal runner

#### Scenario: Legacy one-pass command is not the public runtime

- **WHEN** the user invokes the previous one-pass proposal orchestration command
- **THEN** the CLI rejects the command as unsupported or treats it as invalid manual input
- **AND** the CLI does not run a one-pass orchestration mode as public behavior
