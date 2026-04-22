## ADDED Requirements

### Requirement: JSON log event format
The system SHALL emit application log events as JSON Lines where each line is one valid JSON object containing at least the fields `time`, `module`, `type`, and `message`.

#### Scenario: Info event is written as JSON
- **WHEN** a module writes an informational log message
- **THEN** the output line is valid JSON and contains `type` set to `info`, a non-empty `module`, a non-empty `time`, and the formatted text in `message`

#### Scenario: Error event is written as JSON
- **WHEN** a module writes an error log message
- **THEN** the output line is valid JSON and contains `type` set to `error`, a non-empty `module`, a non-empty `time`, and the formatted text in `message`

#### Scenario: Multiple events are stream friendly
- **WHEN** multiple log events are written to the same writer
- **THEN** each event is written as a separate newline-terminated JSON object

### Requirement: Log timestamp standard
The system SHALL write log event timestamps in UTC using RFC3339Nano-compatible formatting.

#### Scenario: Timestamp can be parsed
- **WHEN** a log event is decoded by a consumer
- **THEN** the `time` field can be parsed as an RFC3339Nano timestamp

#### Scenario: Timestamp uses UTC
- **WHEN** a log event is written
- **THEN** the `time` field represents the event time in UTC

### Requirement: Log module standard
The system SHALL include the logical source of every application log event in the `module` field.

#### Scenario: Module identifies source
- **WHEN** `proposalrunner` writes a workflow log
- **THEN** the `module` field identifies the workflow source such as `proposalrunner`, `temp`, `git`, `codex`, or `github`

#### Scenario: Empty module is normalized
- **WHEN** a caller writes a log event with an empty or whitespace-only module name
- **THEN** the logger writes `unknown` in the `module` field

### Requirement: Supported log types
The system SHALL support exactly two application log types: `info` and `error`.

#### Scenario: Informational helper writes info type
- **WHEN** code writes through the informational logging helper
- **THEN** the emitted event contains `type` set to `info`

#### Scenario: Error helper writes error type
- **WHEN** code writes through the error logging helper
- **THEN** the emitted event contains `type` set to `error`

### Requirement: Safe JSON message encoding
The system SHALL encode log message text through a JSON encoder so special characters and multiline messages do not break the JSON Lines stream.

#### Scenario: Multiline message is encoded
- **WHEN** a module writes a message containing newline characters
- **THEN** the output remains one valid JSON object and the text is preserved in the `message` field after JSON decoding

#### Scenario: Quoted message is encoded
- **WHEN** a module writes a message containing quotes or backslashes
- **THEN** the output remains valid JSON and the text is preserved in the `message` field after JSON decoding
