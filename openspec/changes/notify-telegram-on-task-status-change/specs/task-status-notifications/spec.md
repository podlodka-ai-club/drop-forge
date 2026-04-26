## ADDED Requirements

### Requirement: Task status changes can trigger event-driven notifications
The system SHALL expose task status change notification behavior through event handlers subscribed to task status changed events, without requiring orchestration callers to invoke notification channels directly.

#### Scenario: Status change event is handled by subscribed notification handlers
- **WHEN** a task status changed event is published after a successful Linear transition
- **THEN** each registered notification handler receives the event and can react to it

#### Scenario: Orchestration caller does not call Telegram directly
- **WHEN** an orchestration caller requests a task state transition through the task manager
- **THEN** the caller does not need to call Telegram notification code to trigger a status change notification

#### Scenario: Multiple event handlers can subscribe to status changes
- **WHEN** more than one status change event handler is registered
- **THEN** the dispatcher invokes the registered handlers for the same task status changed event

### Requirement: Telegram notification is sent for task status changes
The system SHALL send a Telegram message when the Telegram notification handler receives a task status changed event.

#### Scenario: Telegram message includes task and target status
- **WHEN** the Telegram notification handler receives a task status changed event
- **THEN** it sends a Telegram message that includes the task ID or identifier and the target workflow state name when available

#### Scenario: Telegram message falls back to target state ID
- **WHEN** the Telegram notification handler receives a task status changed event without a known target workflow state name
- **THEN** it sends a Telegram message that includes the target workflow state ID

#### Scenario: Telegram API failure is returned with context
- **WHEN** Telegram rejects the send message request or the HTTP request fails
- **THEN** the notification handler returns an error that identifies the Telegram send operation and the task status changed event

### Requirement: Runtime configuration for Telegram notifications
The system SHALL read Telegram notification runtime parameters from `.env` and environment variables, and the repository SHALL keep `.env.example` synchronized with those keys without committed values.

#### Scenario: Telegram notification configuration is present
- **WHEN** the environment contains the Telegram API URL, bot token, and target chat ID
- **THEN** the system can construct a Telegram notification handler using those values

#### Scenario: Required Telegram notification configuration is missing
- **WHEN** Telegram notification handling is enabled but the Telegram API URL, bot token, or target chat ID is absent
- **THEN** the system returns a configuration error before registering the Telegram notification handler

#### Scenario: Telegram secrets are not committed
- **WHEN** `.env.example` documents Telegram notification variables
- **THEN** it contains only the variable keys without secret values

### Requirement: Telegram notification dependencies are testable
The system SHALL allow tests to replace Telegram HTTP delivery and event handlers so unit tests do not require real Telegram network calls.

#### Scenario: Telegram HTTP delivery is substituted in tests
- **WHEN** a unit test constructs the Telegram notification handler with a fake HTTP client or sender
- **THEN** the test can assert request payloads and error handling without calling the real Telegram API

#### Scenario: Event handlers are substituted in tests
- **WHEN** a unit test constructs the task manager with a fake status change event handler
- **THEN** the test can assert event publication and handler failure behavior without calling Telegram
