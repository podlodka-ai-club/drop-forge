## ADDED Requirements

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
- **AND** the system processes tasks sequentially in the returned order

### Requirement: Archive input is built from Linear task payload
The system SHALL build Archive runner input from the selected task's identity, title, description, comments, and associated task branch source. The task branch source SHALL be either a concrete branch name or a pull request URL from which the Archive runner can resolve the branch.

#### Scenario: Ready-to-archive task with PR URL is prepared
- **WHEN** a ready-to-archive task includes an associated pull request URL
- **THEN** the Archive runner receives input containing the task identity, task context, and pull request URL

#### Scenario: Ready-to-archive task with branch is prepared
- **WHEN** a ready-to-archive task includes a concrete branch name
- **THEN** the Archive runner receives input containing the task identity, task context, and branch name

#### Scenario: Ready-to-archive task without branch source is rejected
- **WHEN** a ready-to-archive task has no branch name and no associated pull request URL
- **THEN** the orchestration stage returns a contextual error for that task
- **AND** the orchestration stage does not call the Archive runner
- **AND** the orchestration stage does not move the task to archive review

### Requirement: Archive runner archives the OpenSpec change on the task branch
The system SHALL execute Archive by cloning the configured repository into a temporary directory, checking out the task branch, running archival through the OpenSpec Archive skill, committing produced changes, and pushing the task branch.

#### Scenario: Archive runner uses isolated temporary clone
- **WHEN** the Archive runner starts for a valid ready-to-archive task
- **THEN** it creates a temporary workspace separate from the operator checkout
- **AND** it clones the configured repository into that workspace
- **AND** it checks out the task branch before running archival

#### Scenario: Archive runner pushes archival changes
- **WHEN** OpenSpec Archive produces repository changes
- **THEN** the Archive runner stages the changes
- **AND** commits them with a task-specific commit message
- **AND** pushes the commit to the task branch

#### Scenario: Archive runner does not create a new pull request
- **WHEN** Archive execution succeeds
- **THEN** the Archive runner returns success without creating a new pull request

#### Scenario: Archive runner fails when archival produces no changes
- **WHEN** OpenSpec Archive completes but `git status --short` shows no repository changes
- **THEN** the Archive runner returns an error that identifies the no-change condition
- **AND** it does not commit or push

### Requirement: Successful Archive updates the Linear task
The system SHALL move a ready-to-archive task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` before executing the Archive runner and move it to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` after the Archive runner succeeds.

#### Scenario: Archive task enters in-progress state before execution
- **WHEN** a ready-to-archive task is selected for Archive processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the Archive runner until that transition succeeds

#### Scenario: Archive task reaches review state after push
- **WHEN** the Archive runner succeeds for a task moved to archiving-in-progress
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`

#### Scenario: Archive transitions happen in order
- **WHEN** a ready-to-archive task is successfully processed
- **THEN** the orchestration stage moves the task to archiving-in-progress before running the Archive runner
- **AND** the orchestration stage moves the task to archive review only after the Archive runner succeeds

### Requirement: Archive orchestration preserves task state on failure
The system SHALL return contextual errors, avoid running Archive work when the initial in-progress transition fails, and avoid moving a task to archive review when Archive execution fails.

#### Scenario: Archiving in-progress transition fails
- **WHEN** `TaskManager` fails to move a ready-to-archive task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation
- **AND** the orchestration stage does not call the Archive runner or move the task to archive review

#### Scenario: Archive runner fails
- **WHEN** the Archive runner returns an error for a task already moved to archiving-in-progress
- **THEN** the orchestration stage returns an error that identifies the task and Archive operation
- **AND** the orchestration stage does not move the task to archive review

#### Scenario: Archive review transition fails
- **WHEN** the Archive runner succeeds but `TaskManager` fails to move the task to archive review
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation

### Requirement: Archive orchestration emits structured logs
The system SHALL log Archive orchestration decisions and outcomes using the existing structured logger format.

#### Scenario: Archive task processing is logged
- **WHEN** the orchestration stage starts processing a ready-to-archive task
- **THEN** the logs include a structured event with the orchestration module and task identity

#### Scenario: Archive task processing fails
- **WHEN** Archive orchestration fails for a task
- **THEN** the logs include a structured error event with task identity and failure context

### Requirement: Orchestration dependencies support Archive tests
The Archive orchestration stage SHALL allow tests to replace task management and Archive execution dependencies without network access, Codex CLI, GitHub CLI, git, or Linear API calls.

#### Scenario: Archive runner is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake Archive runner
- **THEN** the test can assert Archive execution behavior without Codex CLI, GitHub CLI, git, or network calls

#### Scenario: Archive task manager is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake task manager
- **THEN** the test can assert Archive task filtering and state transition behavior without Linear API calls
