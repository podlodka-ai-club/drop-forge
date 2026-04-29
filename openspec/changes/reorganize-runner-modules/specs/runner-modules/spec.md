## ADDED Requirements

### Requirement: Runner packages are grouped under a dedicated module area

The system SHALL place proposal, apply, archive, and shared runner implementation packages under a dedicated internal runner directory so runner ownership is visible from the package structure.

#### Scenario: Developer locates stage runners

- **WHEN** a developer inspects the internal package tree
- **THEN** proposal, apply, and archive runner packages are located under the same dedicated runner directory
- **AND** unrelated internal packages such as task management, notifications, events, and git management remain outside that runner directory

#### Scenario: Core orchestration imports stage runners

- **WHEN** `CoreOrch` or CLI wiring needs to construct proposal, apply, or archive runner dependencies
- **THEN** it imports the stage runner packages from the dedicated runner directory
- **AND** the orchestration behavior remains equivalent to the previous proposal/apply/archive flow

### Requirement: Shared runner components remove duplicated files

The system SHALL provide shared runner components for common agent execution types, Codex CLI command execution, command output logging, writer fallback behavior, and runner metadata helpers.

#### Scenario: Agent execution contract is shared

- **WHEN** proposal, apply, or archive runner invokes an agent runtime
- **THEN** it uses the shared `AgentExecutor` contract and shared execution input/result types
- **AND** stage packages do not define duplicate `AgentExecutionInput`, `AgentExecutionResult`, or `AgentExecutor` types

#### Scenario: Logged command execution is shared

- **WHEN** a runner executes Codex CLI through a command runner
- **THEN** stdout and stderr are wrapped through one shared logged-command helper
- **AND** stage packages do not keep duplicate `logged_command.go` implementations

#### Scenario: Metadata helpers are shared

- **WHEN** proposal, apply, or archive runner builds display names, titles, branch slugs, or commit messages from task identifier/title
- **THEN** it uses shared runner metadata helpers
- **AND** apply and archive runner packages do not import proposal runner only to reuse metadata functions

### Requirement: Stage-specific runner behavior remains explicit

The system SHALL keep proposal, apply, and archive stage-specific behavior explicit in their stage runner packages or stage profiles.

#### Scenario: Proposal stage keeps PR workflow

- **WHEN** proposal runner completes agent execution with repository changes
- **THEN** it creates a new branch, commits, pushes, creates a pull request, and comments with the final agent response when present

#### Scenario: Apply stage keeps existing branch workflow

- **WHEN** apply runner completes agent execution with repository changes
- **THEN** it commits and pushes to the resolved existing branch without creating a new pull request

#### Scenario: Archive stage keeps existing branch workflow

- **WHEN** archive runner completes agent execution with repository changes
- **THEN** it commits and pushes to the resolved existing branch without creating a new pull request

#### Scenario: Stage prompts remain distinct

- **WHEN** Codex CLI is invoked for proposal, apply, or archive
- **THEN** the prompt uses the stage-specific OpenSpec skill instruction for that stage
- **AND** no stage uses another stage's prompt text

### Requirement: Runner refactor preserves test substitution points

The system SHALL preserve the ability to substitute command execution, agent execution, git management, filesystem cleanup, clock, stdout, and stderr in runner tests.

#### Scenario: Unit test replaces external dependencies

- **WHEN** a unit test constructs a stage runner with fake agent, command, or git dependencies
- **THEN** the test can exercise runner behavior without real Codex CLI, GitHub CLI, Git commands, network access, or Linear API calls

#### Scenario: Codex executor is tested without Codex CLI

- **WHEN** a unit test constructs the shared Codex CLI executor with a fake command runner
- **THEN** the test can assert command name, arguments, working directory, stdin prompt, stdout/stderr forwarding, and optional final-message capture

### Requirement: Architecture documentation reflects runner module boundaries

The system SHALL update `architecture.md` when runner modules move or shared runner components change the mapping between architecture actors and code packages.

#### Scenario: Architecture mapping is current

- **WHEN** a developer reads the architecture mapping after the runner refactor
- **THEN** it identifies the new dedicated runner directory
- **AND** it describes where stage runner packages and shared runner components are implemented
