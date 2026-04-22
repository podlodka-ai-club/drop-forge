## ADDED Requirements

### Requirement: Apply proposal branch input
The system SHALL expose an apply runner workflow that accepts a proposal branch name as the primary input and rejects empty or whitespace-only branch names before creating a temp directory or running external commands.

#### Scenario: Valid proposal branch is accepted
- **WHEN** the caller starts the apply runner with a non-empty proposal branch name
- **THEN** the system starts the OpenSpec apply PR workflow for that branch

#### Scenario: Empty proposal branch is rejected
- **WHEN** the caller starts the apply runner with an empty or whitespace-only proposal branch name
- **THEN** the system returns an error before creating a temp directory or running external commands

### Requirement: Apply workflow starts from proposal branch
The system SHALL clone the configured GitHub repository for apply runs, check out the caller-provided proposal branch in the clone, and run the implementation workflow from that checked-out branch.

#### Scenario: Proposal branch checkout succeeds
- **WHEN** the apply workflow clones the configured repository and the proposal branch exists
- **THEN** the system checks out the proposal branch in the clone before invoking Codex CLI

#### Scenario: Proposal branch checkout fails
- **WHEN** the apply workflow cannot check out the caller-provided proposal branch
- **THEN** the system logs the checkout output and returns an error that identifies the proposal branch checkout step

#### Scenario: No implementation branch before apply
- **WHEN** the apply workflow prepares to invoke Codex CLI
- **THEN** the system has not created a new implementation branch and the clone HEAD points at the caller-provided proposal branch

### Requirement: Codex CLI openspec apply execution
The system SHALL run Codex CLI using the same non-interactive command format as the proposal workflow, with the prompt passed through stdin and containing the `openspec-apply` skill instruction plus the proposal branch name.

#### Scenario: Codex apply receives prompt
- **WHEN** the apply workflow reaches the Codex step
- **THEN** the system logs the prompt and invokes `codex exec --sandbox danger-full-access --cd <clone-dir> -` with that prompt on stdin

#### Scenario: Codex apply succeeds
- **WHEN** Codex CLI exits successfully after applying the OpenSpec change
- **THEN** the system continues to git status, implementation branch creation, commit, push, and PR creation

#### Scenario: Codex apply fails
- **WHEN** Codex CLI exits with a non-zero status during apply
- **THEN** the system logs Codex output and returns an error that identifies the Codex apply step

### Requirement: Apply implementation pull request
The system SHALL create a pull request for implementation changes after a successful apply run, using the caller-provided proposal branch as the PR base branch.

#### Scenario: Implementation pull request is created
- **WHEN** Codex apply succeeds and the cloned repository has changes to commit
- **THEN** the system creates an implementation branch, commits the changes, pushes that branch, creates a PR with base set to the proposal branch, logs the PR URL, and returns that URL to the caller

#### Scenario: No implementation changes were produced
- **WHEN** Codex apply succeeds but git status shows no changes
- **THEN** the system returns an error and does not create an empty implementation pull request

### Requirement: Apply CLI mode
The CLI SHALL provide an explicit apply mode that accepts a proposal branch name without changing the existing proposal invocation behavior.

#### Scenario: Existing proposal invocation is preserved
- **WHEN** an operator runs `orchv3` with a task description as arguments or stdin
- **THEN** the CLI starts the existing proposal workflow for that task description

#### Scenario: Apply invocation uses branch input
- **WHEN** an operator runs `orchv3 apply <proposal-branch>`
- **THEN** the CLI starts the apply workflow with `<proposal-branch>` as the input branch

#### Scenario: Apply invocation requires branch
- **WHEN** an operator runs `orchv3 apply` without a proposal branch
- **THEN** the CLI returns an input error before starting any runner workflow

### Requirement: Apply runtime configuration template
The repository SHALL keep `.env.example` synchronized with apply workflow configuration keys without secrets or default values.

#### Scenario: Template lists apply runtime keys
- **WHEN** a developer needs to configure the apply runner locally
- **THEN** `.env.example` lists the supported apply-specific keys without committed values
