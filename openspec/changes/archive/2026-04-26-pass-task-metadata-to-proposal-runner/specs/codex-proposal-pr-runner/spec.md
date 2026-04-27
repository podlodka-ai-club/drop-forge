## MODIFIED Requirements

### Requirement: Task description input
The system SHALL expose a proposal runner module that accepts a structured `ProposalInput { Title, Identifier, AgentPrompt }` value as the primary input. The system SHALL reject inputs whose `Title` or `AgentPrompt` is empty or whitespace-only after trimming. `Identifier` is optional.

#### Scenario: Valid proposal input is accepted
- **WHEN** the caller starts the proposal runner with a `ProposalInput` whose `Title` and `AgentPrompt` are non-empty
- **THEN** the system starts the proposal PR workflow for that input

#### Scenario: Empty title is rejected
- **WHEN** the caller starts the proposal runner with a `ProposalInput` whose `Title` is empty or whitespace-only
- **THEN** the system returns an error before creating a temp directory or running external commands

#### Scenario: Empty agent prompt is rejected
- **WHEN** the caller starts the proposal runner with a `ProposalInput` whose `AgentPrompt` is empty or whitespace-only
- **THEN** the system returns an error before creating a temp directory or running external commands

#### Scenario: Identifier is optional
- **WHEN** the caller starts the proposal runner with a `ProposalInput` whose `Identifier` is empty
- **THEN** the system proceeds with the workflow and derives PR metadata from `Title` alone

### Requirement: Codex CLI openspec propose execution
The system SHALL execute the OpenSpec proposal generation step through an internal `AgentExecutor` contract. The default implementation SHALL remain Codex CLI and SHALL preserve the current local non-interactive command format `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`, with the prompt passed through stdin. The prompt SHALL be the `AgentPrompt` field of the `ProposalInput`, which already contains the `openspec-propose` skill instruction plus the original task context.

#### Scenario: Agent executor receives proposal task
- **WHEN** the workflow reaches the agent proposal step
- **THEN** the proposal runner invokes its configured `AgentExecutor` with the `AgentPrompt` from the `ProposalInput` and the clone workspace path

#### Scenario: Codex executor receives prompt
- **WHEN** the configured `AgentExecutor` is the default Codex CLI implementation
- **THEN** the Codex executor logs the prompt and invokes `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -` with the `AgentPrompt` on stdin

#### Scenario: Agent executor succeeds
- **WHEN** the `AgentExecutor` exits successfully after creating OpenSpec artifacts
- **THEN** the system continues to git status, commit, push, and PR creation

#### Scenario: Agent executor fails
- **WHEN** the `AgentExecutor` returns an error
- **THEN** the system logs agent output and returns an error that identifies the agent proposal step

## ADDED Requirements

### Requirement: PR metadata is derived from task Title and Identifier
The system SHALL derive the PR title, branch name, and commit message from the `ProposalInput`'s `Title` and `Identifier` fields, not from the `AgentPrompt`. When `Identifier` is non-empty, the human-readable display name used for these metadata SHALL be `"<Identifier>: <Title>"`; otherwise it SHALL be `<Title>` alone. The `AgentPrompt` field SHALL NOT influence PR title, branch name, or commit message.

#### Scenario: Identifier and Title produce combined PR title
- **WHEN** the proposal runner receives a `ProposalInput` with `Identifier="ZIM-42"` and `Title="Add export feature"`
- **THEN** the resulting PR title contains `"ZIM-42: Add export feature"` (with the configured PR title prefix prepended if any)
- **AND** the branch name is built from a slug of `"ZIM-42 Add export feature"`
- **AND** the git commit message equals the PR title

#### Scenario: Empty Identifier falls back to Title only
- **WHEN** the proposal runner receives a `ProposalInput` with empty `Identifier` and `Title="Refactor payments module"`
- **THEN** the resulting PR title contains `"Refactor payments module"` (with the configured prefix prepended if any) and does not contain a leading colon

#### Scenario: AgentPrompt content does not appear in PR title
- **WHEN** the proposal runner receives a `ProposalInput` whose `AgentPrompt` begins with the literal `"Linear task:"` and whose `Title` is `"Add export feature"`
- **THEN** the resulting PR title does not contain `"Linear task:"` and is derived from `Title`

#### Scenario: Title with embedded newlines is normalized
- **WHEN** the proposal runner receives a `ProposalInput` whose `Title` contains a newline character
- **THEN** the resulting PR title contains the title text with newlines replaced by spaces and is truncated to the configured maximum length

### Requirement: Integration test covers orchestrator-to-runner contract
The repository SHALL include a test that exercises `coreorch.BuildProposalInput` together with the proposal runner's PR-title derivation, and that fails if PR title, branch name, or commit message stop reflecting the source task's `Title` (and `Identifier` when present).

#### Scenario: Contract test fails on regression
- **WHEN** a developer changes either `BuildProposalInput` or the runner's metadata-derivation logic in a way that drops `Title` or `Identifier` from the PR title
- **THEN** the test reports a failure that names both the produced and expected PR title
