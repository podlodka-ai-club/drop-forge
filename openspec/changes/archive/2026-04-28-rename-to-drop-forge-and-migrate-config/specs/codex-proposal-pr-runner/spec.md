## MODIFIED Requirements

### Requirement: Runtime configuration from environment files
The system SHALL read runtime configuration from `.env` with `github.com/joho/godotenv` and environment variables, including shared Drop Forge repository settings, branch settings, and external command paths. Repository URL, base branch, remote name, cleanup behavior, and external command paths SHALL use the shared `DROP_FORGE_*` keys. Proposal-only PR metadata settings SHALL remain under proposal-specific keys.

#### Scenario: Repository is configured in env
- **WHEN** `.env` or the process environment contains `DROP_FORGE_REPOSITORY_URL`
- **THEN** the system uses that repository for `git clone`

#### Scenario: Required repository setting is missing
- **WHEN** `DROP_FORGE_REPOSITORY_URL` is absent
- **THEN** the system returns a configuration error before running the proposal workflow

#### Scenario: Environment overrides dot env
- **WHEN** the same configuration key exists in `.env` and in the process environment
- **THEN** the system uses the process environment value

#### Scenario: Dot env syntax is parsed by godotenv
- **WHEN** `.env` contains values that rely on supported godotenv syntax such as quoted strings or inline comments
- **THEN** the system loads those values using godotenv-compatible parsing

#### Scenario: Proposal metadata remains proposal-specific
- **WHEN** `.env` or the process environment contains `PROPOSAL_BRANCH_PREFIX` and `PROPOSAL_PR_TITLE_PREFIX`
- **THEN** the proposal runner uses those values for proposal PR branch names and titles

### Requirement: Environment variable template
The repository SHALL keep `.env.example` synchronized with all supported configuration keys without secrets or default values, using `DROP_FORGE_*` keys for shared runtime settings and proposal-specific keys only for proposal PR metadata.

#### Scenario: Template lists runtime keys
- **WHEN** a developer needs to configure the proposal runner locally
- **THEN** `.env.example` lists the required shared `DROP_FORGE_*` keys and proposal-specific metadata keys without committed values

#### Scenario: Template excludes removed shared proposal keys
- **WHEN** a developer opens `.env.example`
- **THEN** it does not list removed shared keys such as `PROPOSAL_REPOSITORY_URL`, `PROPOSAL_BASE_BRANCH`, `PROPOSAL_REMOTE_NAME`, `PROPOSAL_CLEANUP_TEMP`, `PROPOSAL_GIT_PATH`, `PROPOSAL_CODEX_PATH`, or `PROPOSAL_GH_PATH`

### Requirement: Console logging of workflow steps
The system SHALL log all workflow steps and agent execution interaction to the console as JSON Lines application log events under the Drop Forge application identity, including prompt text, command output, PR creation progress, and final PR URL.

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
- **THEN** the startup message is written as a JSON log event with `module` set to `cli`, `type` set to `info`, and Drop Forge as the default application identity

#### Scenario: CLI fatal error emits structured log
- **WHEN** the CLI cannot load configuration, read the task description, or run the proposal workflow
- **THEN** the failure is written as a JSON log event with `module` set to `cli` and `type` set to `error` before the process exits

### Requirement: PR metadata is derived from task Title and Identifier
The system SHALL derive the PR title, branch name, and commit message from the `ProposalInput`'s `Title` and `Identifier` fields, not from the `AgentPrompt`. When `Identifier` is non-empty, the human-readable display name used for these metadata SHALL be `"<Identifier>: <Title>"`; otherwise it SHALL be `<Title>` alone. The `AgentPrompt` field SHALL NOT influence PR title, branch name, or commit message. The PR body SHALL identify Drop Forge as the generating application.

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

#### Scenario: PR body names Drop Forge
- **WHEN** the proposal runner creates a pull request body
- **THEN** the body identifies the generated proposal as created by Drop Forge
