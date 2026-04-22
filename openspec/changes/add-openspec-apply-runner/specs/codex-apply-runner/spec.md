## ADDED Requirements

### Requirement: Proposal branch input
The system SHALL expose an apply runner workflow that accepts a proposal branch name as the primary input and rejects invalid branch names before creating a temporary directory or running external commands.

#### Scenario: Valid proposal branch is accepted
- **WHEN** the caller starts the apply runner with a non-empty branch name such as `codex/proposal/20260422120000-add-feature`
- **THEN** the system starts the apply workflow for that branch

#### Scenario: Empty proposal branch is rejected
- **WHEN** the caller starts the apply runner with an empty or whitespace-only branch name
- **THEN** the system returns an error before creating a temporary directory or running external commands

#### Scenario: Unsafe proposal branch is rejected
- **WHEN** the caller starts the apply runner with a branch name containing whitespace or starting with `-`
- **THEN** the system returns an input validation error before running external commands

### Requirement: Apply runtime configuration
The system SHALL read apply workflow runtime configuration from `.env` and process environment, including the target repository, remote name, commit title prefix, cleanup behavior, and external command paths.

#### Scenario: Apply repository is configured
- **WHEN** `.env` or the process environment contains the apply repository setting
- **THEN** the system uses that repository for cloning the proposal branch

#### Scenario: Required apply repository setting is missing
- **WHEN** the apply repository setting is absent
- **THEN** the system returns a configuration error before running the apply workflow

#### Scenario: Environment overrides dot env for apply settings
- **WHEN** the same apply configuration key exists in `.env` and in the process environment
- **THEN** the system uses the process environment value

#### Scenario: Apply environment template lists keys
- **WHEN** a developer configures the apply runner locally
- **THEN** `.env.example` lists all supported apply keys without committed values

### Requirement: Temporary clone of proposal branch
The system SHALL create a unique temporary directory for each apply run and clone the configured GitHub repository from the provided proposal branch using Git CLI.

#### Scenario: Proposal branch clone succeeds
- **WHEN** the configured repository contains the provided proposal branch
- **THEN** the system runs `git clone --branch <proposal-branch> --single-branch <repo-url> <clone-dir>` and continues the workflow in the clone root

#### Scenario: Proposal branch clone fails
- **WHEN** the configured repository does not contain the provided proposal branch or clone exits with an error
- **THEN** the system logs the clone output and returns an error that identifies the proposal branch clone step

#### Scenario: Apply temp directory retention
- **WHEN** the apply workflow finishes without explicit cleanup enabled
- **THEN** the system leaves the temporary directory on disk and logs its path for diagnostics

#### Scenario: Apply temp cleanup
- **WHEN** the apply workflow finishes with cleanup enabled by configuration
- **THEN** the system removes the temporary directory and logs the cleanup result

### Requirement: Codex CLI openspec apply execution
The system SHALL run Codex CLI using `codex exec --sandbox danger-full-access --cd <clone-dir> -`, with the prompt passed through stdin and containing the `openspec-apply` skill instruction plus the proposal branch name.

#### Scenario: Codex apply receives prompt
- **WHEN** the apply workflow reaches the Codex step
- **THEN** the system logs the prompt and invokes `codex exec --sandbox danger-full-access --cd <clone-dir> -` with that prompt on stdin

#### Scenario: Codex apply succeeds
- **WHEN** Codex CLI exits successfully after implementing the OpenSpec change
- **THEN** the system continues to git status, commit, and push

#### Scenario: Codex apply fails
- **WHEN** Codex CLI exits with a non-zero status
- **THEN** the system logs Codex output and returns an error that identifies the Codex apply step

### Requirement: Apply workflow logging
The system SHALL log all apply workflow steps and Codex CLI interaction to the console, including branch input, command output, commit progress, push progress, and final updated branch.

#### Scenario: Apply workflow emits step logs
- **WHEN** the apply runner executes a workflow
- **THEN** the console output includes logs for temp directory creation, proposal branch clone, Codex prompt, Codex output, git status, commit, push, and final updated branch

#### Scenario: Codex emits apply output
- **WHEN** Codex CLI writes progress, stderr, or final output to its process streams
- **THEN** the system forwards that output to the console without filtering it out

### Requirement: Commit and push to existing proposal branch
The system SHALL commit changes produced by `openspec-apply` and push them to the provided proposal branch without creating a new branch or pull request.

#### Scenario: Apply changes are pushed
- **WHEN** Codex CLI succeeds and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes `HEAD` to the provided proposal branch, logs the updated branch, and returns the branch name to the caller

#### Scenario: New branch is not created
- **WHEN** the apply workflow prepares Git commands after Codex succeeds
- **THEN** the command sequence does not include `git checkout -b`

#### Scenario: Pull request is not created
- **WHEN** the apply workflow finishes pushing changes
- **THEN** the system does not run `gh pr create` and does not require GitHub CLI for apply

#### Scenario: No apply changes were produced
- **WHEN** Codex CLI succeeds but git status shows no changes
- **THEN** the system returns an error and does not create an empty commit

#### Scenario: Apply push fails
- **WHEN** pushing to the provided proposal branch exits with an error
- **THEN** the system logs the push output and returns an error that identifies the push step

### Requirement: CLI apply mode
The CLI SHALL provide an explicit apply mode that invokes the apply runner with a proposal branch name while preserving existing proposal workflow behavior.

#### Scenario: Explicit apply command
- **WHEN** an operator runs `orchv3 apply <proposal-branch>`
- **THEN** the CLI invokes the apply runner for the provided branch and prints only the updated branch name to stdout on success

#### Scenario: Explicit proposal command
- **WHEN** an operator runs `orchv3 proposal <task description>`
- **THEN** the CLI invokes the existing proposal runner for the task description and prints the PR URL to stdout on success

#### Scenario: Legacy proposal invocation remains available
- **WHEN** an operator runs `orchv3 <task description>` without a known subcommand
- **THEN** the CLI preserves the existing proposal workflow behavior

### Requirement: Testable apply execution
The apply runner module SHALL allow tests to replace external command execution so unit tests do not require real GitHub access, Codex CLI, or network calls.

#### Scenario: Apply command runner is substituted in tests
- **WHEN** a unit test constructs the apply runner with a fake command runner
- **THEN** the test can assert the ordered git and Codex commands without executing external programs

#### Scenario: Proposal behavior remains covered
- **WHEN** common workflow helpers are introduced for apply reuse
- **THEN** existing proposal runner tests still verify proposal clone, branch creation, PR creation, and open-questions comment behavior
