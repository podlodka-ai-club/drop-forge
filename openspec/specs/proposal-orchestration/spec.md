# proposal-orchestration Specification

## Purpose
TBD - created by archiving change add-proposal-orchestrator. Update Purpose after archive.
## Requirements
### Requirement: Proposal orchestration uses ready-to-propose tasks

The system SHALL provide a proposal orchestration stage that loads managed tasks through `TaskManager` and processes only tasks whose workflow state ID matches `LINEAR_STATE_READY_TO_PROPOSE_ID`.

#### Scenario: Ready-to-propose task is selected

- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_PROPOSE_ID`
- **THEN** the proposal orchestration stage treats that task as eligible for proposal execution

#### Scenario: Non-proposal managed task is skipped

- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-code or ready-to-archive
- **THEN** the proposal orchestration stage does not call the proposal runner for that task

#### Scenario: No ready-to-propose tasks exist

- **WHEN** `TaskManager` returns no tasks in `LINEAR_STATE_READY_TO_PROPOSE_ID`
- **THEN** the proposal orchestration stage completes without calling the proposal runner and without mutating task state

### Requirement: Proposal input is built from Linear task payload

The system SHALL build the proposal runner input from the selected task's identifier, title, description, and comments so the generated OpenSpec proposal has enough task context.

#### Scenario: Task with description and comments is prepared

- **WHEN** a ready-to-propose task has a title, description, and comments
- **THEN** the proposal runner receives an input string containing the task identifier, title, description, and comments

#### Scenario: Task without description is prepared

- **WHEN** a ready-to-propose task has no description
- **THEN** the proposal runner still receives a non-empty input string containing the task identifier, title, and any available comments

#### Scenario: Task without comments is prepared

- **WHEN** a ready-to-propose task has no comments
- **THEN** the proposal runner input remains valid and explicitly represents that no review comments are available

### Requirement: Existing proposal runner is used as the proposal executor

The system SHALL execute proposals by calling the existing proposal runner contract with one prepared task input at a time and SHALL not change the proposal runner's internal git, Codex, PR, or comment workflow as part of proposal orchestration.

#### Scenario: Proposal runner is called for eligible task

- **WHEN** a ready-to-propose task is processed
- **THEN** the orchestration stage calls the proposal runner with the prepared task input

#### Scenario: Multiple tasks are processed sequentially

- **WHEN** multiple ready-to-propose tasks are returned in one orchestration pass
- **THEN** the orchestration stage runs the proposal runner for one task at a time in the returned order

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

### Requirement: Proposal orchestration emits structured logs

The system SHALL log proposal orchestration decisions and outcomes using the existing structured logger format.

#### Scenario: Task processing is logged

- **WHEN** the orchestration stage starts processing a ready-to-propose task
- **THEN** the logs include a structured event with the orchestration module and task identity

#### Scenario: Task skip is logged

- **WHEN** the orchestration stage skips a managed task because it is not in the ready-to-propose state
- **THEN** the logs include a structured event with the task identity and current state

#### Scenario: Task processing fails

- **WHEN** proposal orchestration fails for a task
- **THEN** the logs include a structured error event with task identity and failure context

### Requirement: Proposal orchestration is available from the CLI

The system SHALL expose a CLI mode that runs one proposal orchestration pass while preserving the existing direct proposal runner mode for task descriptions passed through args or stdin.

#### Scenario: CLI starts proposal orchestration mode

- **WHEN** the user invokes the proposal orchestration CLI mode
- **THEN** the CLI loads config, wires `TaskManager`, proposal runner, and logger, and runs one proposal orchestration pass

#### Scenario: CLI direct proposal mode is preserved

- **WHEN** the user provides a task description through the existing args or stdin path
- **THEN** the CLI runs the existing proposal runner single-run workflow and prints the returned PR URL

### Requirement: Proposal orchestration dependencies are testable

The proposal orchestration stage SHALL allow tests to replace task management and proposal execution dependencies without network access, Codex CLI, GitHub CLI, or Linear API calls.

#### Scenario: Task manager is substituted in tests

- **WHEN** a unit test constructs proposal orchestration with a fake task manager
- **THEN** the test can assert task filtering, PR attachment, and state transition behavior without Linear API calls

#### Scenario: Proposal runner is substituted in tests

- **WHEN** a unit test constructs proposal orchestration with a fake proposal runner
- **THEN** the test can assert proposal execution behavior without Codex CLI, GitHub CLI, git, or network calls
