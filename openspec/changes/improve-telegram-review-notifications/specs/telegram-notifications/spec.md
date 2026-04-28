## MODIFIED Requirements

### Requirement: Telegram subscriber sends status change messages
The system SHALL register a Telegram subscriber that listens for `task.status_changed` events and sends a message to the configured Telegram chat through the Telegram Bot API only when the target state is one of the configured human-review states.

#### Scenario: Review status change event sends Telegram message
- **WHEN** Telegram notifications are enabled
- **AND** a `task.status_changed` event is published with target state `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, or `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`
- **THEN** the Telegram subscriber sends a `sendMessage` request using the configured bot token and chat ID

#### Scenario: Non-review status change event is ignored
- **WHEN** Telegram notifications are enabled
- **AND** a `task.status_changed` event is published with a target state other than `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, or `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`
- **THEN** the Telegram subscriber does not send a `sendMessage` request
- **AND** the event handling completes without error

#### Scenario: Message uses human-readable task fields when available
- **WHEN** the event payload includes task identifier, task title, or target state name
- **THEN** the Telegram message includes those human-readable values

#### Scenario: Message includes pull request URL when available
- **WHEN** the event payload includes a pull request URL
- **THEN** the Telegram message includes that pull request URL as the review link

#### Scenario: Message falls back to stable IDs
- **WHEN** the event payload has no task identifier, task title, target state name, or pull request URL
- **THEN** the Telegram message still includes the task ID and target state ID
