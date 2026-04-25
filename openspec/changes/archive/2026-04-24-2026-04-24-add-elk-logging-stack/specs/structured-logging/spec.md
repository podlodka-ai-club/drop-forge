## MODIFIED Requirements

### Requirement: JSON log event format
The system SHALL emit application log events as JSON Lines where each line is one valid JSON object containing at least the fields `time`, `module`, `type`, and `message`, and MAY include an optional `service` field identifying the emitting application.

#### Scenario: Info event is written as JSON
- **WHEN** a module writes an informational log message
- **THEN** the output line is valid JSON and contains `type` set to `info`, a non-empty `module`, a non-empty `time`, and the formatted text in `message`

#### Scenario: Error event is written as JSON
- **WHEN** a module writes an error log message
- **THEN** the output line is valid JSON and contains `type` set to `error`, a non-empty `module`, a non-empty `time`, and the formatted text in `message`

#### Scenario: Multiple events are stream friendly
- **WHEN** multiple log events are written to the same writer
- **THEN** each event is written as a separate newline-terminated JSON object

#### Scenario: Service field is included when configured
- **WHEN** the logger is constructed with a non-empty service name
- **THEN** every emitted event contains a `service` field set to that name

#### Scenario: Service field is omitted when not configured
- **WHEN** the logger is constructed without a service name
- **THEN** emitted events do not contain a `service` field

## ADDED Requirements

### Requirement: Service-aware logger constructor
The `internal/steplog` package SHALL provide a constructor that binds a service name to every event emitted by the returned logger, while keeping the existing service-less constructor fully backward compatible.

#### Scenario: NewWithService sets service field
- **WHEN** code constructs a logger with `NewWithService(out, "orchv3")`
- **THEN** every event produced by that logger has `service` equal to `"orchv3"`

#### Scenario: New preserves existing behavior
- **WHEN** code constructs a logger with `New(out)`
- **THEN** emitted events do not contain a `service` field
- **AND** all previously documented behaviors of `New` continue to hold
