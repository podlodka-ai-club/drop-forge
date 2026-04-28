## MODIFIED Requirements

### Requirement: Task status change event has stable payload
The system SHALL define a `task.status_changed` event payload that includes the task identity, target workflow state, and event timestamp, and SHALL support optional human-readable task and pull request context.

#### Scenario: Status change event contains required fields
- **WHEN** a task status change event is created after a task transition
- **THEN** the payload includes the task ID, target state ID, and occurred-at timestamp

#### Scenario: Status change event supports optional human-readable fields
- **WHEN** the publisher has task identifier, task title, source state, or target state name available
- **THEN** the payload can include those values without changing the event type

#### Scenario: Status change event supports optional pull request context
- **WHEN** the publisher has a pull request URL or branch source associated with the task transition
- **THEN** the payload can include that pull request context without changing the event type
