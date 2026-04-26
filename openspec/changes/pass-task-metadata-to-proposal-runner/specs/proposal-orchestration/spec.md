## MODIFIED Requirements

### Requirement: Proposal input is built from Linear task payload

The system SHALL build the proposal runner input as a structured `ProposalInput { Title, Identifier, AgentPrompt }` value derived from the selected task's identifier, title, description, and comments. The `Title` SHALL be the task's title (or a non-empty fallback when the task has no title), the `Identifier` SHALL be the task's Linear identifier (or empty if absent), and the `AgentPrompt` SHALL be a multi-line block containing the task identifier, title, description, and comments so the generated OpenSpec proposal has enough task context.

#### Scenario: Task with description and comments is prepared

- **WHEN** a ready-to-propose task has a title, description, and comments
- **THEN** the proposal runner receives a `ProposalInput` whose `Title` equals the task title, `Identifier` equals the task identifier, and `AgentPrompt` contains the task identifier, title, description, and comments

#### Scenario: Task without description is prepared

- **WHEN** a ready-to-propose task has no description
- **THEN** the proposal runner still receives a `ProposalInput` whose `AgentPrompt` is non-empty and contains the task identifier, title, and any available comments

#### Scenario: Task without comments is prepared

- **WHEN** a ready-to-propose task has no comments
- **THEN** the proposal runner input remains valid and the `AgentPrompt` explicitly represents that no review comments are available

#### Scenario: Task without title falls back to placeholder

- **WHEN** a ready-to-propose task has an empty title
- **THEN** the proposal runner receives a `ProposalInput` whose `Title` is set to a non-empty fallback so the runner does not reject the input

### Requirement: Existing proposal runner is used as the proposal executor

The system SHALL execute proposals by calling the existing proposal runner contract with one prepared `ProposalInput` at a time and SHALL not change the proposal runner's internal git, Codex, PR, or comment workflow as part of proposal orchestration.

#### Scenario: Proposal runner is called for eligible task

- **WHEN** a ready-to-propose task is processed
- **THEN** the orchestration stage calls the proposal runner with the prepared `ProposalInput`

#### Scenario: Multiple tasks are processed sequentially

- **WHEN** multiple ready-to-propose tasks are returned in one orchestration pass
- **THEN** the orchestration stage runs the proposal runner for one task at a time in the returned order

### Requirement: Proposal orchestration is available from the CLI

The system SHALL expose a CLI mode that runs one proposal orchestration pass while preserving the existing direct proposal runner mode for task descriptions passed through args or stdin. In the direct mode the CLI SHALL build a `ProposalInput` whose `Title` and `AgentPrompt` are both set to the user-provided text and whose `Identifier` is empty.

#### Scenario: CLI starts proposal orchestration mode

- **WHEN** the user invokes the proposal orchestration CLI mode
- **THEN** the CLI loads config, wires `TaskManager`, proposal runner, and logger, and runs one proposal orchestration pass

#### Scenario: CLI direct proposal mode is preserved

- **WHEN** the user provides a task description through the existing args or stdin path
- **THEN** the CLI builds a `ProposalInput` with `Title` and `AgentPrompt` equal to that text and `Identifier` empty, runs the proposal runner single-run workflow, and prints the returned PR URL
