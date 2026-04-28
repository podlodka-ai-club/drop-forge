# orchestration-events Specification

## Purpose
TBD - created by archiving change add-telegram-status-notifications. Update Purpose after archive.
## Requirements
### Requirement: Internal events can be published and subscribed to
The system SHALL provide an internal event dispatcher that allows code to publish domain events and allows subscribers to handle events by event type without depending on the publishing component.

#### Scenario: Subscriber receives matching event
- **WHEN** a subscriber is registered for `task.status_changed`
- **AND** the dispatcher publishes a `task.status_changed` event
- **THEN** the dispatcher calls that subscriber with the event payload

#### Scenario: Subscriber does not receive unrelated event
- **WHEN** a subscriber is registered for `task.status_changed`
- **AND** the dispatcher publishes an event with a different type
- **THEN** the dispatcher does not call that subscriber

#### Scenario: Multiple subscribers receive one event
- **WHEN** multiple subscribers are registered for the same event type
- **AND** the dispatcher publishes an event of that type
- **THEN** every registered subscriber is called once for that event

### Requirement: Event publication is testable without external services
The event dispatcher SHALL allow unit tests to register in-memory subscribers and assert published event payloads without network access or external brokers.

#### Scenario: Test subscriber captures event
- **WHEN** a unit test registers an in-memory subscriber
- **AND** application code publishes an event through the dispatcher
- **THEN** the test can inspect the captured event type and payload

#### Scenario: Subscriber failure is returned to publisher
- **WHEN** a registered subscriber returns an error while handling an event
- **THEN** the dispatcher returns a contextual publish error to the caller

### Requirement: Task status change event has stable payload
The system SHALL define a `task.status_changed` event payload that includes the task identity, target workflow state, and event timestamp.

#### Scenario: Status change event contains required fields
- **WHEN** a task status change event is created after a task transition
- **THEN** the payload includes the task ID, target state ID, and occurred-at timestamp

#### Scenario: Status change event supports optional human-readable fields
- **WHEN** the publisher has task identifier, task title, source state, or target state name available
- **THEN** the payload can include those values without changing the event type

