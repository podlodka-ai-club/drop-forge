## ADDED Requirements

### Requirement: AI review state IDs are conditionally required as a unit
The task manager configuration SHALL accept three AI review workflow state IDs (`LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`) on an all-or-nothing basis. Configuration validation SHALL accept either all three populated together or all three empty together. Any partial combination SHALL be a configuration error.

#### Scenario: All three AI review state IDs populated
- **WHEN** the environment populates all three AI review state IDs
- **THEN** task manager configuration validation succeeds
- **AND** the AI review state IDs become managed input states for the Review orchestration stage

#### Scenario: All three AI review state IDs empty
- **WHEN** the environment leaves all three AI review state IDs empty
- **THEN** task manager configuration validation succeeds
- **AND** producer routes transition tasks directly to their human review state

#### Scenario: Partial AI review configuration is rejected
- **WHEN** at least one AI review state ID is populated and at least one is empty
- **THEN** task manager configuration validation returns an error identifying the partial AI review configuration
- **AND** the orchestrator does not start

### Requirement: AI review state IDs are managed input queues for the Review orchestration stage
When all three AI review state IDs are configured, the task manager SHALL include them in the list of managed state IDs used to load tasks from Linear so the orchestration layer can route those tasks to the Review runner.

#### Scenario: AI review state IDs are loaded as managed tasks
- **WHEN** all three AI review state IDs are configured and an orchestration layer requests managed tasks
- **THEN** the task manager includes the three AI review state IDs in the state IDs used to load tasks

#### Scenario: AI review state IDs are not loaded when feature disabled
- **WHEN** all three AI review state IDs are empty and an orchestration layer requests managed tasks
- **THEN** the task manager does not include any AI review state ID in the state IDs used to load tasks

### Requirement: AI review review-state transitions are applied to Linear
The task manager SHALL allow callers to move a managed task into any of the three AI review state IDs and from any of them into the corresponding human review state.

#### Scenario: Task is moved to need-proposal-ai-review
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Task is moved to need-code-ai-review
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Task is moved to need-archive-ai-review
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Task is moved from AI review to human review
- **WHEN** an orchestration layer completes Review execution and requests a move from an AI review state to the matching human review state
- **THEN** the task manager applies that state transition to Linear

## MODIFIED Requirements

### Requirement: Runtime configuration for Linear task management
The system SHALL read Linear task manager runtime parameters from `.env` and environment variables, including the target Linear project, managed workflow state IDs (ready-to-propose, ready-to-code, ready-to-archive), in-progress target state IDs, configured human review target state IDs, and the optional AI review target state IDs (`LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`). The repository SHALL keep `.env.example` synchronized with all configured keys without committed values.

#### Scenario: Linear task manager configuration is present
- **WHEN** the environment contains the required Linear connection, project filter, managed state IDs, in-progress target state IDs, and human review target state IDs
- **THEN** the task manager uses those values to select tasks and apply workflow transitions

#### Scenario: Required Linear task manager configuration is missing
- **WHEN** a required Linear connection, project filter, managed state ID, in-progress target state ID, or human review target state ID is absent
- **THEN** the system returns a configuration error before starting task processing

#### Scenario: Environment example lists in-progress workflow states
- **WHEN** a developer opens `.env.example`
- **THEN** it includes keys for `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` without default values

#### Scenario: Environment example lists AI review workflow states
- **WHEN** a developer opens `.env.example`
- **THEN** it includes keys for `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, and `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID` without default values

### Requirement: In-progress state IDs are transition targets only
The task manager SHALL keep configured in-progress workflow state IDs available as transition targets without treating those states as managed input queues for task selection. The task manager SHALL also keep human review state IDs as transition targets only, while the AI review state IDs (when configured) SHALL be both managed input queues and transition targets.

#### Scenario: Managed task selection excludes in-progress states
- **WHEN** the task manager builds the list of managed state IDs for loading tasks from Linear
- **THEN** the list includes ready-to-propose, ready-to-code, and ready-to-archive state IDs
- **AND** the list does not include proposing-in-progress, code-in-progress, or archiving-in-progress state IDs

#### Scenario: Managed task selection excludes human review states
- **WHEN** the task manager builds the list of managed state IDs for loading tasks from Linear
- **THEN** the list does not include need-proposal-review, need-code-review, or need-archive-review state IDs

#### Scenario: Managed task selection includes AI review states when configured
- **WHEN** all three AI review state IDs are configured and the task manager builds the list of managed state IDs
- **THEN** the list includes need-proposal-ai-review, need-code-ai-review, and need-archive-ai-review state IDs
