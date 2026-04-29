## ADDED Requirements

### Requirement: Producer trailer is written into producer-runner commit messages
The system SHALL ensure that `ProposalRunner`, `ApplyRunner`, and `ArchiveRunner` append a producer trailer to the commit message before pushing changes when the AI review feature is configured. The trailer SHALL include `Produced-By` (slot identifier), `Produced-Model` (concrete model identifier), and `Produced-Stage` (`proposal`, `apply`, or `archive`). The trailer SHALL be in canonical git trailer format readable by `git interpret-trailers --parse`.

#### Scenario: Proposal commit carries producer trailer
- **WHEN** the proposal runner commits agent-produced changes with the AI review feature configured
- **THEN** the commit message contains `Produced-By`, `Produced-Model`, and `Produced-Stage: proposal` trailers

#### Scenario: Apply commit carries producer trailer
- **WHEN** the apply runner commits agent-produced changes with the AI review feature configured
- **THEN** the commit message contains `Produced-By`, `Produced-Model`, and `Produced-Stage: apply` trailers

#### Scenario: Archive commit carries producer trailer
- **WHEN** the archive runner commits agent-produced changes with the AI review feature configured
- **THEN** the commit message contains `Produced-By`, `Produced-Model`, and `Produced-Stage: archive` trailers

#### Scenario: Trailer is omitted when no producer is configured
- **WHEN** a producer runner commits agent-produced changes and the AI review feature is not configured
- **THEN** the commit message does not contain producer trailers
- **AND** the existing commit message format is preserved

## MODIFIED Requirements

### Requirement: Successful proposal updates the Linear task
The system SHALL move a ready-to-propose task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID` before executing the proposal runner, then attach the proposal PR URL to the task. After the proposal runner succeeds, the system SHALL move the task to `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID` when the AI review feature is enabled, otherwise to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.

#### Scenario: Proposal task enters in-progress state before execution

- **WHEN** a ready-to-propose task is selected for proposal processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the proposal runner until that transition succeeds

#### Scenario: Proposal task reaches AI review state when feature enabled

- **WHEN** the proposal runner returns a PR URL for a task moved to proposing-in-progress and the AI review feature is enabled
- **THEN** the orchestration stage asks `TaskManager` to attach that PR URL to the task
- **AND** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`

#### Scenario: Proposal task reaches human review state when feature disabled

- **WHEN** the proposal runner returns a PR URL for a task moved to proposing-in-progress and the AI review feature is disabled
- **THEN** the orchestration stage asks `TaskManager` to attach that PR URL to the task
- **AND** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`

#### Scenario: Proposal transitions happen in order

- **WHEN** a ready-to-propose task is successfully processed
- **THEN** the orchestration stage moves the task to proposing-in-progress before running the proposal runner
- **AND** the orchestration stage attaches the PR URL before moving the task to the proposal review state (AI or human)

### Requirement: Successful Apply updates the Linear task
The system SHALL move a ready-to-code task to `LINEAR_STATE_CODE_IN_PROGRESS_ID` before executing the Apply runner. After the Apply runner succeeds, the system SHALL move the task to `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID` when the AI review feature is enabled, otherwise to `LINEAR_STATE_NEED_CODE_REVIEW_ID`.

#### Scenario: Code task enters in-progress state before execution
- **WHEN** a ready-to-code task is selected for Apply processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_CODE_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the Apply runner until that transition succeeds

#### Scenario: Code task reaches AI review state after push when feature enabled
- **WHEN** the Apply runner succeeds for a task moved to code-in-progress and the AI review feature is enabled
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`

#### Scenario: Code task reaches human review state after push when feature disabled
- **WHEN** the Apply runner succeeds for a task moved to code-in-progress and the AI review feature is disabled
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_CODE_REVIEW_ID`

#### Scenario: Apply transitions happen in order
- **WHEN** a ready-to-code task is successfully processed
- **THEN** the orchestration stage moves the task to code-in-progress before running the Apply runner
- **AND** the orchestration stage moves the task to the code review state (AI or human) only after the Apply runner succeeds

### Requirement: Successful Archive updates the Linear task
The system SHALL move a ready-to-archive task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` before executing the Archive runner. After the Archive runner succeeds, the system SHALL move the task to `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID` when the AI review feature is enabled, otherwise to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`.

#### Scenario: Archive task enters in-progress state before execution
- **WHEN** a ready-to-archive task is selected for Archive processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the Archive runner until that transition succeeds

#### Scenario: Archive task reaches AI review state after push when feature enabled
- **WHEN** the Archive runner succeeds for a task moved to archiving-in-progress and the AI review feature is enabled
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`

#### Scenario: Archive task reaches human review state after push when feature disabled
- **WHEN** the Archive runner succeeds for a task moved to archiving-in-progress and the AI review feature is disabled
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`

#### Scenario: Archive transitions happen in order
- **WHEN** a ready-to-archive task is successfully processed
- **THEN** the orchestration stage moves the task to archiving-in-progress before running the Archive runner
- **AND** the orchestration stage moves the task to the archive review state (AI or human) only after the Archive runner succeeds
