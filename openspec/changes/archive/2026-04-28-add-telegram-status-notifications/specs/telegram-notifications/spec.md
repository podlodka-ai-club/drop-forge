## ADDED Requirements

### Requirement: Telegram notifications are runtime configurable
The system SHALL read Telegram notification settings from `.env` and environment variables and SHALL keep `.env.example` synchronized with the required keys without committed values.

#### Scenario: Telegram notifications are disabled by default
- **WHEN** `TELEGRAM_NOTIFICATIONS_ENABLED` is absent or false
- **THEN** the system starts without requiring Telegram bot token, chat ID, API URL, or timeout values
- **AND** no Telegram subscriber is registered

#### Scenario: Telegram notifications require delivery settings when enabled
- **WHEN** `TELEGRAM_NOTIFICATIONS_ENABLED=true`
- **THEN** configuration loading requires non-empty `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `TELEGRAM_API_URL`, and a positive `TELEGRAM_TIMEOUT`

#### Scenario: Environment example lists Telegram keys
- **WHEN** a developer opens `.env.example`
- **THEN** it includes `TELEGRAM_NOTIFICATIONS_ENABLED`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `TELEGRAM_API_URL`, and `TELEGRAM_TIMEOUT` without default values

### Requirement: Telegram subscriber sends status change messages
The system SHALL register a Telegram subscriber that listens for `task.status_changed` events and sends a message to the configured Telegram chat through the Telegram Bot API.

#### Scenario: Status change event sends Telegram message
- **WHEN** Telegram notifications are enabled
- **AND** a `task.status_changed` event is published
- **THEN** the Telegram subscriber sends a `sendMessage` request using the configured bot token and chat ID

#### Scenario: Message uses human-readable task fields when available
- **WHEN** the event payload includes task identifier, task title, or target state name
- **THEN** the Telegram message includes those human-readable values

#### Scenario: Message falls back to stable IDs
- **WHEN** the event payload has no task identifier, task title, or target state name
- **THEN** the Telegram message still includes the task ID and target state ID

### Requirement: Telegram delivery failures are observable
Telegram delivery failures SHALL be returned by the subscriber and logged by the publisher or wiring layer without reverting the task status transition that produced the event.

#### Scenario: Telegram API rejects message
- **WHEN** the Telegram API responds with a non-success HTTP status or unsuccessful response body
- **THEN** the Telegram subscriber returns a contextual error that identifies Telegram message delivery

#### Scenario: Telegram request times out
- **WHEN** the Telegram request exceeds the configured timeout
- **THEN** the Telegram subscriber returns a contextual timeout error

#### Scenario: Task transition remains successful when notification fails
- **WHEN** a task status transition in Linear succeeds
- **AND** Telegram delivery fails for the resulting event
- **THEN** the task status transition remains successful from the orchestration flow perspective
- **AND** the notification failure is logged for investigation

### Requirement: Telegram notifier is testable without real Telegram API
The Telegram subscriber SHALL allow tests to replace the HTTP transport or API endpoint so delivery behavior can be verified without calling the real Telegram service.

#### Scenario: Test server receives sendMessage request
- **WHEN** a unit test configures the Telegram subscriber with a local HTTP server
- **AND** the subscriber handles a `task.status_changed` event
- **THEN** the test can assert the request path, chat ID, and message body
