## ADDED Requirements

### Requirement: Interactive console highlights error events
The system SHALL render application log events with `type` equal to `error` in red when writing to an interactive console, while preserving raw JSON Lines without ANSI escape sequences for non-interactive outputs and external log sinks.

#### Scenario: Error event is red in interactive console
- **WHEN** the CLI writes a JSON log event whose `type` is `error` to an interactive console writer
- **THEN** the visible console line is wrapped with ANSI red and reset escape sequences

#### Scenario: Info event is not colorized
- **WHEN** the CLI writes a JSON log event whose `type` is `info` to an interactive console writer
- **THEN** the visible console line is written without ANSI color escape sequences

#### Scenario: Non-interactive output remains valid JSON
- **WHEN** the CLI writes error and info events to a non-interactive writer such as a pipe, file, or test buffer
- **THEN** every output line remains a valid JSON object without ANSI color escape sequences

#### Scenario: External log sink receives raw JSON
- **WHEN** console highlighting is active and a secondary log sink is configured
- **THEN** the secondary sink receives the original JSON Lines without ANSI color escape sequences
