## ADDED Requirements

### Requirement: Proposal branch input
The system SHALL expose an apply runner that accepts a proposal branch name string as the primary input and rejects empty or whitespace-only branch names before creating a temp directory or running external commands.

#### Scenario: Valid proposal branch is accepted
- **WHEN** the caller starts the apply runner with a non-empty proposal branch name
- **THEN** the system starts the OpenSpec apply PR workflow for that branch

#### Scenario: Empty proposal branch is rejected
- **WHEN** the caller starts the apply runner with an empty or whitespace-only proposal branch name
- **THEN** the system returns an error before creating a temp directory or running external commands

### Requirement: Apply CLI mode
The system SHALL provide an explicit CLI mode for apply while preserving the existing proposal workflow invocation.

#### Scenario: Apply subcommand runs apply workflow
- **WHEN** the user runs `orchv3 apply <proposal-branch>`
- **THEN** the system runs the apply workflow using `<proposal-branch>` as input

#### Scenario: Proposal subcommand runs proposal workflow
- **WHEN** the user runs `orchv3 proposal <task description>`
- **THEN** the system runs the existing proposal workflow using `<task description>` as input

#### Scenario: Legacy proposal invocation still works
- **WHEN** the user runs `orchv3 <task description>` without an explicit subcommand
- **THEN** the system runs the existing proposal workflow using `<task description>` as input

### Requirement: Apply runtime configuration
The system SHALL load apply runner runtime configuration from `.env` and environment variables, including implementation branch prefix and pull request title prefix, while reusing the configured repository URL, remote name, cleanup setting, and external command paths.

#### Scenario: Apply branch settings are configured
- **WHEN** `.env` or the process environment contains apply branch and PR title settings
- **THEN** the system uses those settings when creating the implementation branch and pull request title

#### Scenario: Apply uses shared repository configuration
- **WHEN** the apply runner starts
- **THEN** the system uses the configured repository URL, remote name, cleanup setting, git path, Codex path, and GitHub CLI path from the shared runner configuration

#### Scenario: Required repository setting is missing
- **WHEN** the target repository setting is absent
- **THEN** the system returns a configuration error before running the apply workflow

### Requirement: Environment variable template for apply
The repository SHALL keep `.env.example` synchronized with all apply runner configuration keys without secrets or committed default values.

#### Scenario: Template lists apply runtime keys
- **WHEN** a developer needs to configure the apply runner locally
- **THEN** `.env.example` lists the apply-specific keys without values

### Requirement: Temporary clone from proposal branch
The system SHALL create a unique temporary directory for each apply run, clone the configured GitHub repository into that directory, and ensure the working copy is on the provided proposal branch before invoking Codex.

#### Scenario: Proposal branch checkout succeeds
- **WHEN** the configured repository is reachable and the proposal branch exists
- **THEN** the system creates a temporary directory, logs its path, prepares a clone rooted at the proposal branch, and continues the workflow in the clone root

#### Scenario: Proposal branch checkout fails
- **WHEN** the configured repository is reachable but the proposal branch cannot be checked out
- **THEN** the system logs the git output and returns an error that identifies the proposal branch preparation step

#### Scenario: Apply temp directory retention
- **WHEN** the apply workflow finishes without an explicit cleanup setting
- **THEN** the system leaves the temporary directory on disk and logs its path for diagnostics

#### Scenario: Apply temp cleanup
- **WHEN** the apply workflow finishes with cleanup enabled by configuration
- **THEN** the system removes the temporary directory and logs the cleanup result

### Requirement: Codex CLI openspec apply execution
The system SHALL run Codex CLI using the current local non-interactive command format `codex exec --sandbox danger-full-access --cd <clone-dir> -`, with the prompt passed through stdin and containing the `openspec-apply` skill instruction plus the proposal branch name.

#### Scenario: Codex CLI receives apply prompt
- **WHEN** the workflow reaches the Codex step
- **THEN** the system logs the prompt and invokes `codex exec --sandbox danger-full-access --cd <clone-dir> -` with that prompt on stdin

#### Scenario: Codex CLI succeeds
- **WHEN** Codex CLI exits successfully after implementing the OpenSpec proposal
- **THEN** the system continues to git status, implementation branch creation, commit, push, and PR creation

#### Scenario: Codex CLI fails
- **WHEN** Codex CLI exits with a non-zero status
- **THEN** the system logs Codex output and returns an error that identifies the Codex apply step

### Requirement: No pre-apply implementation branch creation
The system SHALL NOT create the implementation branch before Codex apply has completed successfully and produced changes.

#### Scenario: Branch is created after Codex changes
- **WHEN** Codex apply succeeds and `git status --short` reports changes
- **THEN** the system creates a new implementation branch from the current proposal branch checkout

#### Scenario: No changes were produced
- **WHEN** Codex apply succeeds but `git status --short` reports no changes
- **THEN** the system returns an error and does not create an implementation branch, commit, push, or pull request

### Requirement: Apply pull request creation
The system SHALL commit Codex apply changes, push an implementation branch, create a pull request through the authenticated `gh` CLI, and return the pull request URL.

#### Scenario: Apply pull request is created
- **WHEN** Codex apply succeeds and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes the implementation branch, creates a PR with `--base <proposal-branch>` and `--head <implementation-branch>`, logs the PR URL, and returns that URL to the caller

#### Scenario: GitHub CLI is unavailable or unauthenticated
- **WHEN** PR creation requires `gh` but `gh` is missing or not authenticated
- **THEN** the system returns an error that identifies the GitHub CLI prerequisite

#### Scenario: PR creation fails
- **WHEN** the PR creation command exits with an error
- **THEN** the system logs the PR creation output and returns an error that identifies the PR step

### Requirement: Console logging of apply workflow steps
The system SHALL log all apply workflow steps and Codex CLI interaction to the console, including prompt text, command output, proposal branch preparation, implementation branch creation, PR creation progress, and final PR URL.

#### Scenario: Apply workflow emits step logs
- **WHEN** the apply runner executes a workflow
- **THEN** the console output includes logs for temp directory creation, git clone or checkout, Codex prompt, Codex output, git commit and push, PR creation, and final PR URL

#### Scenario: Codex emits apply output
- **WHEN** Codex CLI writes reasoning, progress, stderr, or final output to its process streams
- **THEN** the system forwards that output to the console without filtering it out

### Requirement: Testable apply command execution
The apply runner SHALL allow tests to replace external command execution so unit tests do not require real GitHub access, Codex CLI, or network calls.

#### Scenario: Command runner is substituted in apply tests
- **WHEN** a unit test constructs the apply runner with a fake command runner
- **THEN** the test can assert the ordered git, Codex, and PR commands without executing external programs
