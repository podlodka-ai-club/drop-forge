## ADDED Requirements

### Requirement: Proposal runner is invoked only through orchestration runtime

The system SHALL keep the proposal runner module available for internal orchestration calls while preventing the CLI from exposing a direct manual proposal runner mode.

#### Scenario: Orchestration invokes proposal runner

- **WHEN** proposal orchestration processes a task from `Ready to propose`
- **THEN** it can call the proposal runner with the prepared Linear task input

#### Scenario: CLI arguments do not invoke proposal runner directly

- **WHEN** a user passes a task description as CLI arguments
- **THEN** the CLI does not call the proposal runner directly

#### Scenario: CLI stdin does not invoke proposal runner directly

- **WHEN** a user pipes a task description into the CLI
- **THEN** the CLI does not call the proposal runner directly
