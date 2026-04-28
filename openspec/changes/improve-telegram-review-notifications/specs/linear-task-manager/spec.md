## MODIFIED Requirements

### Requirement: Task status transitions publish events
The task manager SHALL publish a `task.status_changed` event after a managed task is successfully moved to another Linear workflow state and SHALL include expanded task context when the caller provides it for that transition.

#### Scenario: Successful move publishes status change event
- **WHEN** a caller requests a task state change through `TaskManager`
- **AND** Linear accepts the state transition
- **THEN** the task manager publishes a `task.status_changed` event containing the task ID and target state ID

#### Scenario: Successful review move publishes task and PR context
- **WHEN** an orchestration stage requests a task move to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, or `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`
- **AND** the orchestration stage provides the task identifier, task title, target state name, and pull request URL or branch source
- **AND** Linear accepts the state transition
- **THEN** the task manager publishes a `task.status_changed` event containing those provided context fields

#### Scenario: Failed move does not publish status change event
- **WHEN** a caller requests a task state change through `TaskManager`
- **AND** Linear rejects the state transition
- **THEN** the task manager returns the state transition error
- **AND** the task manager does not publish a `task.status_changed` event

#### Scenario: Event publish failure does not revert successful move
- **WHEN** Linear accepts a task state transition
- **AND** publishing the resulting `task.status_changed` event fails
- **THEN** the task manager logs the event publication failure
- **AND** the task manager still reports the state transition as successful to the caller
