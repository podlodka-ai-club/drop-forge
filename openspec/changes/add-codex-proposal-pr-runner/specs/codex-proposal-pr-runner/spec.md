## ADDED Requirements

### Requirement: Task description input
The system SHALL expose a proposal runner module that accepts a task description string as the primary input and rejects empty or whitespace-only descriptions.

#### Scenario: Valid task description is accepted
- **WHEN** the caller starts the proposal runner with a non-empty task description
- **THEN** the system starts the proposal PR workflow for that description

#### Scenario: Empty task description is rejected
- **WHEN** the caller starts the proposal runner with an empty or whitespace-only task description
- **THEN** the system returns an error before creating a temp directory or running external commands

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

### Requirement: Environment variable templates
The repository SHALL include `env.dist` with all required configuration keys and SHALL keep `.env.example` synchronized without secrets or default values.

#### Scenario: Templates list runtime keys
- **WHEN** a developer needs to configure the proposal runner locally
- **THEN** `env.dist` and `.env.example` list the required keys without committed values

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
The system SHALL run Codex CLI using the current local non-interactive command format `codex exec --cd <clone-dir> -`, with the prompt passed through stdin and containing the `openspec-propose` skill instruction plus the original task description.

#### Scenario: Codex CLI receives prompt
- **WHEN** the workflow reaches the Codex step
- **THEN** the system logs the prompt and invokes `codex exec --cd <clone-dir> -` with that prompt on stdin

#### Scenario: Codex CLI succeeds
- **WHEN** Codex CLI exits successfully after creating OpenSpec artifacts
- **THEN** the system continues to git status, commit, push, and PR creation

#### Scenario: Codex CLI fails
- **WHEN** Codex CLI exits with a non-zero status
- **THEN** the system logs Codex output and returns an error that identifies the Codex step

### Requirement: Console logging of workflow steps
The system SHALL log all workflow steps and Codex CLI interaction to the console, including prompt text, command output, and PR creation progress.

#### Scenario: Workflow emits step logs
- **WHEN** the proposal runner executes a workflow
- **THEN** the console output includes logs for temp directory creation, git clone, Codex prompt, Codex output, git commit/push, PR creation, and final PR URL

#### Scenario: Codex emits reasoning or agent output
- **WHEN** Codex CLI writes reasoning, progress, stderr, or final output to its process streams
- **THEN** the system forwards that output to the console without filtering it out

### Requirement: Pull request creation
The system SHALL create a pull request through the authenticated `gh` CLI in the target GitHub repository after Codex CLI produces changes and SHALL return the pull request URL.

#### Scenario: Pull request is created
- **WHEN** Codex CLI succeeds and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes a branch, creates a PR through `gh`, logs the PR URL, and returns that URL to the caller

#### Scenario: GitHub CLI is unavailable or unauthenticated
- **WHEN** PR creation requires `gh` but `gh` is missing or not authenticated
- **THEN** the system returns an error that identifies the GitHub CLI prerequisite

#### Scenario: No changes were produced
- **WHEN** Codex CLI succeeds but git status shows no changes
- **THEN** the system returns an error and does not create an empty pull request

#### Scenario: PR creation fails
- **WHEN** the PR creation command exits with an error
- **THEN** the system logs the PR creation output and returns an error that identifies the PR step

### Requirement: Open questions PR comment
The system SHALL add unresolved implementation questions as a separate comment on the created pull request when such questions are known at PR creation time.

#### Scenario: Open questions are present
- **WHEN** the workflow creates a pull request and the runner has one or more open questions
- **THEN** the system publishes those questions as a pull request comment and logs the comment creation step

#### Scenario: No open questions are present
- **WHEN** the workflow creates a pull request and the runner has no open questions
- **THEN** the system does not create an empty questions comment and still returns the pull request URL

#### Scenario: Open questions comment fails
- **WHEN** the pull request is created but publishing the open questions comment fails
- **THEN** the system returns an error that identifies the comment step and logs the comment creation output

### Requirement: Testable command execution
The proposal runner module SHALL allow tests to replace external command execution so unit tests do not require real GitHub access, Codex CLI, or network calls.

#### Scenario: Command runner is substituted in tests
- **WHEN** a unit test constructs the proposal runner with a fake command runner
- **THEN** the test can assert the ordered git, Codex, and PR commands without executing external programs
