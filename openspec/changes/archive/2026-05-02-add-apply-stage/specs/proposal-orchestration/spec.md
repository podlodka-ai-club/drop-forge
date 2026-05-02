## ADDED Requirements

### Requirement: Apply orchestration uses ready-to-code tasks
The system SHALL provide an Apply orchestration stage that loads managed tasks through `TaskManager` and processes tasks whose workflow state ID matches `LINEAR_STATE_READY_TO_CODE_ID`.

#### Scenario: Ready-to-code task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_CODE_ID`
- **THEN** the orchestration stage treats that task as eligible for Apply execution

#### Scenario: Non-code managed task is skipped by Apply route
- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-propose or ready-to-archive
- **THEN** the Apply route does not call the Apply runner for that task

#### Scenario: Proposal and Apply tasks are processed in one pass
- **WHEN** one orchestration pass receives both ready-to-propose and ready-to-code tasks
- **THEN** the system routes each task to the executor matching its current state
- **AND** the system processes tasks sequentially in the returned order

### Requirement: Apply input is built from Linear task payload
The system SHALL build Apply runner input from the selected task's identity, title, description, comments, and associated task branch source. The task branch source SHALL be either a concrete branch name or a pull request URL from which the Apply runner can resolve the branch.

#### Scenario: Ready-to-code task with PR URL is prepared
- **WHEN** a ready-to-code task includes an associated pull request URL
- **THEN** the Apply runner receives input containing the task identity, task context, and pull request URL

#### Scenario: Ready-to-code task with branch is prepared
- **WHEN** a ready-to-code task includes a concrete branch name
- **THEN** the Apply runner receives input containing the task identity, task context, and branch name

#### Scenario: Ready-to-code task without branch source is rejected
- **WHEN** a ready-to-code task has no branch name and no associated pull request URL
- **THEN** the orchestration stage returns a contextual error for that task
- **AND** the orchestration stage does not call the Apply runner
- **AND** the orchestration stage does not move the task to code review

### Requirement: Apply runner implements code changes on the task branch
The system SHALL execute Apply by cloning the configured repository into a temporary directory, checking out the task branch, running implementation through the OpenSpec Apply skill, committing produced changes, and pushing the task branch.

#### Scenario: Apply runner uses isolated temporary clone
- **WHEN** the Apply runner starts for a valid ready-to-code task
- **THEN** it creates a temporary workspace
- **AND** it clones the configured repository into that workspace
- **AND** it checks out the task branch before running implementation

#### Scenario: Apply runner pushes implementation changes
- **WHEN** OpenSpec Apply produces repository changes
- **THEN** the Apply runner stages the changes
- **AND** commits them with a task-specific commit message
- **AND** pushes the commit to the task branch

#### Scenario: Apply runner does not create a new pull request
- **WHEN** Apply execution succeeds
- **THEN** the Apply runner returns success without creating a new pull request

#### Scenario: Apply runner fails when implementation produces no changes
- **WHEN** OpenSpec Apply completes but `git status --short` shows no repository changes
- **THEN** the Apply runner returns an error that identifies the no-change condition
- **AND** it does not commit or push

### Requirement: Successful Apply updates the Linear task
The system SHALL move a ready-to-code task to `LINEAR_STATE_CODE_IN_PROGRESS_ID` before executing the Apply runner and move it to `LINEAR_STATE_NEED_CODE_REVIEW_ID` after the Apply runner succeeds.

#### Scenario: Code task enters in-progress state before execution
- **WHEN** a ready-to-code task is selected for Apply processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_CODE_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the Apply runner until that transition succeeds

#### Scenario: Code task reaches review state after push
- **WHEN** the Apply runner succeeds for a task moved to code-in-progress
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_CODE_REVIEW_ID`

#### Scenario: Apply transitions happen in order
- **WHEN** a ready-to-code task is successfully processed
- **THEN** the orchestration stage moves the task to code-in-progress before running the Apply runner
- **AND** the orchestration stage moves the task to code review only after the Apply runner succeeds

### Requirement: Apply orchestration preserves task state on failure
The system SHALL return contextual errors, avoid running Apply work when the initial in-progress transition fails, and avoid moving a task to code review when Apply execution fails.

#### Scenario: Code in-progress transition fails
- **WHEN** `TaskManager` fails to move a ready-to-code task to `LINEAR_STATE_CODE_IN_PROGRESS_ID`
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation
- **AND** the orchestration stage does not call the Apply runner or move the task to code review

#### Scenario: Apply runner fails
- **WHEN** the Apply runner returns an error for a task already moved to code-in-progress
- **THEN** the orchestration stage returns an error that identifies the task and Apply operation
- **AND** the orchestration stage does not move the task to code review

#### Scenario: Code review transition fails
- **WHEN** the Apply runner succeeds but `TaskManager` fails to move the task to code review
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation

### Requirement: Apply orchestration emits structured logs
The system SHALL log Apply orchestration decisions and outcomes using the existing structured logger format.

#### Scenario: Apply task processing is logged
- **WHEN** the orchestration stage starts processing a ready-to-code task
- **THEN** the logs include a structured event with the orchestration module and task identity

#### Scenario: Apply task processing fails
- **WHEN** Apply orchestration fails for a task
- **THEN** the logs include a structured error event with task identity and failure context

### Requirement: Orchestration dependencies support Apply tests
The Apply orchestration stage SHALL allow tests to replace task management and Apply execution dependencies without network access, Codex CLI, GitHub CLI, git, or Linear API calls.

#### Scenario: Apply runner is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake Apply runner
- **THEN** the test can assert Apply execution behavior without Codex CLI, GitHub CLI, git, or network calls

#### Scenario: Apply task manager is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake task manager
- **THEN** the test can assert Apply task filtering and state transition behavior without Linear API calls
