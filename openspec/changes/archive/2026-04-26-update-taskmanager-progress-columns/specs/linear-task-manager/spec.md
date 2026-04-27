## MODIFIED Requirements

### Requirement: Runtime configuration for Linear task management
The system SHALL read Linear task manager runtime parameters from `.env` and environment variables, including the target Linear project, managed workflow state IDs, in-progress target state IDs, and configured review target state IDs, and the repository SHALL keep `.env.example` synchronized with those keys without committed values.

#### Scenario: Linear task manager configuration is present
- **WHEN** the environment contains the required Linear connection, project filter, managed state IDs, in-progress target state IDs, and review target state IDs
- **THEN** the task manager uses those values to select tasks and apply workflow transitions

#### Scenario: Required Linear task manager configuration is missing
- **WHEN** a required Linear connection, project filter, managed state ID, in-progress target state ID, or review target state ID is absent
- **THEN** the system returns a configuration error before starting task processing

#### Scenario: Environment example lists in-progress workflow states
- **WHEN** a developer opens `.env.example`
- **THEN** it includes keys for `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` without default values

## ADDED Requirements

### Requirement: In-progress state IDs are transition targets only
The task manager SHALL keep configured in-progress workflow state IDs available as transition targets without treating those states as managed input queues for task selection.

#### Scenario: Managed task selection excludes in-progress states
- **WHEN** the task manager builds the list of managed state IDs for loading tasks from Linear
- **THEN** the list includes ready-to-propose, ready-to-code, and ready-to-archive state IDs
- **AND** the list does not include proposing-in-progress, code-in-progress, or archiving-in-progress state IDs
