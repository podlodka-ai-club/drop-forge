# codex-proposal-pr-runner Specification

## Purpose
Описывает запуск Codex для подготовки OpenSpec proposal в отдельном clone workspace и публикацию результата в pull request.
## Requirements
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

### Requirement: Runtime configuration from environment files
The system SHALL read runtime configuration from `.env` with `github.com/joho/godotenv` and environment variables, including the target GitHub repository, branch settings, and external command paths.

#### Scenario: Repository is configured in env
- **WHEN** `.env` or the process environment contains the target repository setting
- **THEN** the system uses that repository for `git clone`

#### Scenario: Required repository setting is missing
- **WHEN** the target repository setting is absent
- **THEN** the system returns a configuration error before running the proposal workflow

#### Scenario: Environment overrides dot env
- **WHEN** the same configuration key exists in `.env` and in the process environment
- **THEN** the system uses the process environment value

#### Scenario: Dot env syntax is parsed by godotenv
- **WHEN** `.env` contains values that rely on supported godotenv syntax such as quoted strings or inline comments
- **THEN** the system loads those values using godotenv-compatible parsing

### Requirement: Environment variable template
The repository SHALL keep `.env.example` synchronized with all supported configuration keys without secrets or default values.

#### Scenario: Template lists runtime keys
- **WHEN** a developer needs to configure the proposal runner locally
- **THEN** `.env.example` lists the required keys without committed values

### Requirement: Temporary clone workspace
The system SHALL create a unique temporary directory for each run, clone the configured GitHub repository into that directory using `git clone`, and preserve the temporary directory by default for diagnostics.

#### Scenario: Repository clone succeeds
- **WHEN** the configured repository is reachable
- **THEN** the system creates a temporary directory, logs its path, clones the repository into it, and continues the workflow in the clone root

#### Scenario: Default temp directory retention
- **WHEN** the workflow finishes without an explicit cleanup setting
- **THEN** the system leaves the temporary directory on disk and logs its path for diagnostics

#### Scenario: Explicit temp cleanup
- **WHEN** the workflow finishes with cleanup enabled by configuration
- **THEN** the system removes the temporary directory and logs the cleanup result

#### Scenario: Repository clone fails
- **WHEN** `git clone` exits with an error
- **THEN** the system logs the clone output and returns an error that identifies the clone step

### Requirement: Codex CLI openspec propose execution
The system SHALL execute the OpenSpec proposal generation step through an internal `AgentExecutor` contract. The default implementation SHALL remain Codex CLI and SHALL preserve the current local non-interactive command format `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`, with the prompt passed through stdin. The proposal runner SHALL pass the `AgentPrompt` field of the `ProposalInput` to the configured `AgentExecutor`; the default Codex CLI implementation SHALL wrap that task context in the `openspec-propose` skill instruction before invoking Codex.

#### Scenario: Agent executor receives proposal task
- **WHEN** the workflow reaches the agent proposal step
- **THEN** the proposal runner invokes its configured `AgentExecutor` with the `AgentPrompt` from the `ProposalInput` and the clone workspace path

#### Scenario: Codex executor receives prompt
- **WHEN** the configured `AgentExecutor` is the default Codex CLI implementation
- **THEN** the Codex executor logs the prompt and invokes `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -` with a prompt that contains the `openspec-propose` instruction and the `AgentPrompt` task context on stdin

#### Scenario: Agent executor succeeds
- **WHEN** the `AgentExecutor` exits successfully after creating OpenSpec artifacts
- **THEN** the system continues to git status, commit, push, and PR creation

#### Scenario: Agent executor fails
- **WHEN** the `AgentExecutor` returns an error
- **THEN** the system logs agent output and returns an error that identifies the agent proposal step

### Requirement: Console logging of workflow steps
The system SHALL log all workflow steps and agent execution interaction to the console as JSON Lines application log events, including prompt text, command output, PR creation progress, and final PR URL.

#### Scenario: Workflow emits structured step logs
- **WHEN** the proposal runner executes a workflow
- **THEN** the console output includes JSON log events for temp directory creation, git clone, agent prompt or execution start, agent output, git commit/push, PR creation, and final PR URL

#### Scenario: Workflow logs include required fields
- **WHEN** the proposal runner writes a workflow log event
- **THEN** the event contains `time`, `module`, `type`, and `message` fields

#### Scenario: Agent emits reasoning or output
- **WHEN** the configured agent runtime writes reasoning, progress, stderr, or final output to its process streams
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

### Requirement: Pull request creation
The system SHALL create a pull request through the authenticated `gh` CLI in the target GitHub repository after the agent executor produces changes and SHALL return the pull request URL.

#### Scenario: Pull request is created
- **WHEN** the agent executor succeeds and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes a branch, creates a PR through `gh`, logs the PR URL, and returns that URL to the caller

#### Scenario: GitHub CLI is unavailable or unauthenticated
- **WHEN** PR creation requires `gh` but `gh` is missing or not authenticated
- **THEN** the system returns an error that identifies the GitHub CLI prerequisite

#### Scenario: No changes were produced
- **WHEN** the agent executor succeeds but git status shows no changes
- **THEN** the system returns an error and does not create an empty pull request

#### Scenario: PR creation fails
- **WHEN** the PR creation command exits with an error
- **THEN** the system logs the PR creation output and returns an error that identifies the PR step

### Requirement: Testable command execution
The proposal runner module SHALL allow tests to replace agent execution and external command execution so unit tests do not require real GitHub access, Codex CLI, or network calls.

#### Scenario: Agent executor is substituted in tests
- **WHEN** a unit test constructs the proposal runner with a fake agent executor
- **THEN** the test can assert the workflow around clone, git status, branch, commit, push, PR creation, and PR comment without executing a real agent runtime

#### Scenario: Command runner is substituted in Codex executor tests
- **WHEN** a unit test constructs the Codex CLI executor with a fake command runner
- **THEN** the test can assert the Codex command, arguments, stdin prompt, output forwarding, and last-message capture without executing Codex CLI

### Requirement: Codex final response PR comment
The system SHALL publish the last non-empty agent response as a separate comment on the created pull request. The default Codex CLI implementation SHALL obtain that response from `codex exec --output-last-message`.

#### Scenario: Final agent response is present
- **WHEN** the workflow creates a pull request and the agent executor returns a non-empty final message
- **THEN** the system publishes that message as a pull request comment and logs the comment creation step

#### Scenario: Final agent response is empty
- **WHEN** the workflow creates a pull request and the agent executor returns an empty or whitespace-only final message
- **THEN** the system does not create an empty pull request comment and still returns the pull request URL

#### Scenario: Agent response comment fails
- **WHEN** the pull request is created but publishing the last agent message as a comment fails
- **THEN** the system returns an error that identifies the comment step and logs the comment creation output

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
