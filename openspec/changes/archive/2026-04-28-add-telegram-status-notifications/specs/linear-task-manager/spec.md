## ADDED Requirements

### Requirement: Task status transitions publish events
The task manager SHALL publish a `task.status_changed` event after a managed task is successfully moved to another Linear workflow state.

#### Scenario: Successful move publishes status change event
- **WHEN** a caller requests a task state change through `TaskManager`
- **AND** Linear accepts the state transition
- **THEN** the task manager publishes a `task.status_changed` event containing the task ID and target state ID

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

### Requirement: Task status event publishing is optional and testable
The task manager SHALL allow tests and application wiring to provide an event publisher, and SHALL keep existing task management behavior valid when no publisher is configured.

#### Scenario: Task manager runs without publisher
- **WHEN** a task manager has no event publisher configured
- **AND** Linear accepts a requested state transition
- **THEN** the task manager completes the state transition without failing because of missing event wiring

#### Scenario: Fake publisher captures transition event
- **WHEN** a unit test configures the task manager with a fake event publisher
- **AND** a task state transition succeeds
- **THEN** the test can assert that exactly one `task.status_changed` event was published with the expected task ID and target state ID
