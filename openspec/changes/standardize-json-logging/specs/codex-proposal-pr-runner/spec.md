## MODIFIED Requirements

### Requirement: Console logging of workflow steps
The system SHALL log all workflow steps and Codex CLI interaction to the console as JSON Lines application log events, including prompt text, command output, PR creation progress, and final PR URL.

#### Scenario: Workflow emits structured step logs
- **WHEN** the proposal runner executes a workflow
- **THEN** the console output includes JSON log events for temp directory creation, git clone, Codex prompt, Codex output, git commit/push, PR creation, and final PR URL

#### Scenario: Workflow logs include required fields
- **WHEN** the proposal runner writes a workflow log event
- **THEN** the event contains `time`, `module`, `type`, and `message` fields

#### Scenario: Codex emits reasoning or agent output
- **WHEN** Codex CLI writes reasoning, progress, stderr, or final output to its process streams
- **THEN** the system forwards that output to the console as JSON log events without filtering it out

#### Scenario: Workflow failure emits error log
- **WHEN** a workflow step fails after logging has been initialized
- **THEN** the system writes a JSON log event with `type` set to `error` and a `message` that identifies the failed step

#### Scenario: CLI startup emits structured log
- **WHEN** the CLI starts without a proposal task description
- **THEN** the startup message is written as a JSON log event with `module` set to `cli` and `type` set to `info`

#### Scenario: CLI fatal error emits structured log
- **WHEN** the CLI cannot load configuration, read the task description, or run the proposal workflow
- **THEN** the failure is written as a JSON log event with `module` set to `cli` and `type` set to `error` before the process exits
