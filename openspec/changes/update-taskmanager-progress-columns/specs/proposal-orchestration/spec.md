## MODIFIED Requirements

### Requirement: Successful proposal updates the Linear task
The system SHALL move a ready-to-propose task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID` before executing the proposal runner, then attach the proposal PR URL to the task and move the task to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID` after the proposal runner succeeds.

#### Scenario: Proposal task enters in-progress state before execution

- **WHEN** a ready-to-propose task is selected for proposal processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the proposal runner until that transition succeeds

#### Scenario: Proposal task reaches review state

- **WHEN** the proposal runner returns a PR URL for a task moved to proposing-in-progress
- **THEN** the orchestration stage asks `TaskManager` to attach that PR URL to the task
- **AND** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`

#### Scenario: Proposal transitions happen in order

- **WHEN** a ready-to-propose task is successfully processed
- **THEN** the orchestration stage moves the task to proposing-in-progress before running the proposal runner
- **AND** the orchestration stage attaches the PR URL before moving the task to the proposal review state

### Requirement: Proposal orchestration preserves task state on failure
The system SHALL return contextual errors, avoid running proposal work when the initial in-progress transition fails, and avoid moving a task to proposal review when proposal execution or PR attachment fails.

#### Scenario: Proposing in-progress transition fails

- **WHEN** `TaskManager` fails to move a ready-to-propose task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation
- **AND** the orchestration stage does not call the proposal runner, attach a PR URL, or move the task to proposal review

#### Scenario: Proposal runner fails

- **WHEN** the proposal runner returns an error for a task already moved to proposing-in-progress
- **THEN** the orchestration stage returns an error that identifies the task
- **AND** the orchestration stage does not attach a PR URL or move the task to proposal review

#### Scenario: PR attachment fails

- **WHEN** the proposal runner returns a PR URL but `TaskManager` fails to attach it to the task
- **THEN** the orchestration stage returns an error that identifies the task and PR attachment operation
- **AND** the orchestration stage does not move the task to proposal review

#### Scenario: Proposal review transition fails

- **WHEN** the PR URL is attached but `TaskManager` fails to move the task to proposal review
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation
