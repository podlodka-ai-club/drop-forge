## ADDED Requirements

### Requirement: Orchestration routes run concurrently by stage
The system SHALL run ready-to-propose, ready-to-code, and ready-to-archive route processing in separate goroutines within a single orchestration pass when tasks for those stages are present.

#### Scenario: Different ready columns start without waiting for each other
- **WHEN** one orchestration pass receives at least one ready-to-propose task and at least one ready-to-code or ready-to-archive task
- **THEN** the system starts the proposal route in its own goroutine
- **AND** the system starts each other non-empty route in its own goroutine without waiting for the proposal route to finish

#### Scenario: Empty route does not start worker
- **WHEN** one orchestration pass receives no tasks for a route
- **THEN** the system does not start a goroutine for that empty route
- **AND** the system logs the existing no-ready-tasks message for that route

#### Scenario: One stage keeps its internal order
- **WHEN** one orchestration pass receives multiple tasks for the same ready state
- **THEN** the route goroutine for that state processes those tasks sequentially in the order returned by `TaskManager`

### Requirement: Orchestration pass waits for concurrent stage routes
The system SHALL wait for all stage route goroutines started by an orchestration pass before the pass returns to the monitor loop.

#### Scenario: Pass completes after all started routes finish
- **WHEN** proposal, Apply, and Archive routes are started in one pass
- **THEN** `RunProposalsOnce` does not return until all started route goroutines have finished

#### Scenario: Monitor waits after full concurrent pass
- **WHEN** a concurrent orchestration pass completes
- **THEN** the monitor waits for the configured polling interval only after every started route goroutine from that pass has finished

### Requirement: Concurrent stage route errors are isolated and aggregated
The system SHALL let independent stage route goroutines finish after another stage returns an error, and SHALL return an aggregated pass error that identifies the failed stage routes and task context.

#### Scenario: Apply failure does not stop Archive route already running
- **WHEN** Apply and Archive routes are started in the same pass
- **AND** the Apply route returns an error for one task
- **THEN** the Archive route continues processing its current task with the original context unless that context is externally cancelled
- **AND** the pass returns an error that includes Apply failure context

#### Scenario: Multiple stage failures are reported
- **WHEN** more than one stage route returns an error during the same pass
- **THEN** the pass returns an error that includes each failed stage name and its contextual error

#### Scenario: Successful stage mutations are not rolled back
- **WHEN** one stage route succeeds and another stage route fails during the same pass
- **THEN** the successful stage keeps its completed task transitions and artifacts
- **AND** the pass still returns an error for the failed stage

### Requirement: Concurrent orchestration logs remain structured
The system SHALL keep orchestration logs in the existing structured logger format when stage routes run concurrently.

#### Scenario: Concurrent route logs identify stage and task
- **WHEN** multiple stage route goroutines emit logs during the same pass
- **THEN** each task processing log includes enough stage and task identity to identify the route that emitted it

#### Scenario: Concurrent route log writes remain valid events
- **WHEN** multiple stage route goroutines write logs at the same time
- **THEN** each emitted log entry remains a valid structured event in the existing logger format

## MODIFIED Requirements

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
- **AND** the system may process tasks from different ready states concurrently through their stage route goroutines
- **AND** each stage route processes its own tasks sequentially in the returned order
