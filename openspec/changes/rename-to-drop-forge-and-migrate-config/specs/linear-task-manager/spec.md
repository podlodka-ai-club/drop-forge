## MODIFIED Requirements

### Requirement: Runtime configuration for Linear task management
The system SHALL read Linear task manager runtime parameters from `.env` and environment variables, including the target Linear project, managed workflow state IDs, in-progress target state IDs, and configured review target state IDs. Shared Drop Forge runtime settings such as repository, polling, cleanup, and command paths SHALL NOT be configured through Linear-specific keys, and the repository SHALL keep `.env.example` synchronized with supported Linear keys without committed values.

#### Scenario: Linear task manager configuration is present
- **WHEN** the environment contains the required Linear connection, project filter, managed state IDs, in-progress target state IDs, and review target state IDs
- **THEN** the task manager uses those values to select tasks and apply workflow transitions

#### Scenario: Required Linear task manager configuration is missing
- **WHEN** a required Linear connection, project filter, managed state ID, in-progress target state ID, or review target state ID is absent
- **THEN** the system returns a configuration error before starting task processing

#### Scenario: Environment example lists in-progress workflow states
- **WHEN** a developer opens `.env.example`
- **THEN** it includes keys for `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` without default values

#### Scenario: Environment example keeps Linear keys separate
- **WHEN** a developer opens `.env.example`
- **THEN** Linear connection and workflow state keys are grouped separately from shared `DROP_FORGE_*` runtime keys

#### Scenario: Shared runtime keys are not duplicated under Linear
- **WHEN** configuration loading reads repository URL, polling interval, cleanup behavior, or external command paths
- **THEN** it reads those values from `DROP_FORGE_*` keys and not from Linear-specific keys
