## MODIFIED Requirements

### Requirement: Task state transitions can be applied back to Linear
The system SHALL allow a caller to move a managed task to another configured Linear state, and SHALL publish a task status changed event after Linear confirms a successful transition.

#### Scenario: Managed task is moved to a new state
- **WHEN** a caller requests a state change for a managed task
- **THEN** the task manager updates the task in Linear to the requested target state

#### Scenario: Proposal task is moved to proposal review
- **WHEN** an external orchestration layer completes proposal execution and requests a move to the configured `Need Proposal Review` state
- **THEN** the task manager updates the task in Linear to that review state

#### Scenario: Code task is moved to code review
- **WHEN** an external orchestration layer completes code execution and requests a move to the configured `Need Code Review` state
- **THEN** the task manager updates the task in Linear to that review state

#### Scenario: Archive task is moved to archive review
- **WHEN** an external orchestration layer completes archive execution and requests a move to the configured `Need Archive Review` state
- **THEN** the task manager updates the task in Linear to that review state

#### Scenario: Successful state transition publishes a status change event
- **WHEN** Linear confirms that a managed task was moved to a requested target state
- **THEN** the task manager publishes a task status changed event containing the task identifier available to the transition, the target state ID, and the best available target state name

#### Scenario: State transition failure does not publish a status change event
- **WHEN** Linear rejects a requested state transition for a managed task
- **THEN** the task manager returns an error that identifies the task and the state transition operation
- **AND** no task status changed event is published for that failed transition

#### Scenario: Event handler failure is returned with context
- **WHEN** Linear confirms the state transition but a registered status change event handler fails
- **THEN** the task manager returns an error that identifies the task, target state, and event handling operation
