## MODIFIED Requirements

### Requirement: Runtime configuration from environment files
The system SHALL read runtime configuration from `.env` with `github.com/joho/godotenv` and environment variables, including the target repository, branch settings, selected Git provider, and external command paths.

#### Scenario: Repository is configured in env
- **WHEN** `.env` or the process environment contains the target repository setting
- **THEN** the system uses that repository for `git clone`

#### Scenario: Required repository setting is missing
- **WHEN** the target repository setting is absent
- **THEN** the system returns a configuration error before running the proposal workflow

#### Scenario: Provider is configured in env
- **WHEN** `.env` or the process environment contains `PROPOSAL_GIT_PROVIDER=gitlab`
- **THEN** the proposal runner uses GitLab provider operations for review request creation and comments

#### Scenario: Environment overrides dot env
- **WHEN** the same configuration key exists in `.env` and in the process environment
- **THEN** the system uses the process environment value

#### Scenario: Dot env syntax is parsed by godotenv
- **WHEN** `.env` contains values that rely on supported godotenv syntax such as quoted strings or inline comments
- **THEN** the system loads those values using godotenv-compatible parsing

### Requirement: Pull request creation
The system SHALL create a provider-specific review request in the target repository after the agent executor produces changes and SHALL return the review request URL. For GitHub this SHALL be a pull request through `gh`; for GitLab this SHALL be a merge request through `glab`.

#### Scenario: GitHub pull request is created
- **WHEN** the selected provider is `github`, the agent executor succeeds, and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes a branch, creates a PR through `gh`, logs the PR URL, and returns that URL to the caller

#### Scenario: GitLab merge request is created
- **WHEN** the selected provider is `gitlab`, the agent executor succeeds, and the cloned repository has changes to commit
- **THEN** the system commits the changes, pushes a branch, creates an MR through `glab`, logs the MR URL, and returns that URL to the caller

#### Scenario: Selected provider CLI is unavailable or unauthenticated
- **WHEN** review request creation requires the selected provider CLI but that CLI is missing or not authenticated
- **THEN** the system returns an error that identifies the provider CLI prerequisite

#### Scenario: No changes were produced
- **WHEN** the agent executor succeeds but git status shows no changes
- **THEN** the system returns an error and does not create an empty review request

#### Scenario: Review request creation fails
- **WHEN** the review request creation command exits with an error
- **THEN** the system logs the review request creation output and returns an error that identifies the review request step

### Requirement: Codex final response PR comment
The system SHALL publish the last non-empty agent response as a separate comment on the created review request. The default Codex CLI implementation SHALL obtain that response from `codex exec --output-last-message`.

#### Scenario: Final agent response is present
- **WHEN** the workflow creates a review request and the agent executor returns a non-empty final message
- **THEN** the system publishes that message as a review request comment through the selected provider and logs the comment creation step

#### Scenario: Final agent response is empty
- **WHEN** the workflow creates a review request and the agent executor returns an empty or whitespace-only final message
- **THEN** the system does not create an empty review request comment and still returns the review request URL

#### Scenario: Agent response comment fails
- **WHEN** the review request is created but publishing the last agent message as a comment fails
- **THEN** the system returns an error that identifies the comment step and logs the comment creation output

### Requirement: Proposal runner delegates repository operations to GitManager
The proposal runner SHALL use the internal `GitManager` package for repository clone workspace creation, status inspection, proposal branch checkout, staging, commit, push, review request creation, and final agent response comments while preserving the existing proposal workflow contract.

#### Scenario: Proposal runner uses GitManager for clone and review request workflow
- **WHEN** the proposal runner receives a valid proposal input and the agent produces changes
- **THEN** it delegates clone workspace creation, `git status --short`, proposal branch creation, `git add`, `git commit`, `git push`, provider-specific review request creation, and optional provider-specific review request comment to `GitManager`
- **AND** it returns the created review request URL to the caller

#### Scenario: Proposal metadata remains stage-specific
- **WHEN** the proposal runner prepares branch name, review request title, review request body, and commit message
- **THEN** it derives those values from the existing proposal input and config rules before passing them to `GitManager`
- **AND** `GitManager` does not derive proposal metadata from Linear task fields or agent prompts

#### Scenario: Proposal no-change behavior is preserved
- **WHEN** the agent succeeds but `GitManager` returns empty short status
- **THEN** the proposal runner returns the existing no-changes error
- **AND** it does not ask `GitManager` to commit, push, create a review request, or comment on a review request

#### Scenario: Proposal runner repository dependency is testable
- **WHEN** a unit test constructs the proposal runner with a fake `GitManager`
- **THEN** the test can assert proposal workflow decisions without executing real git, provider CLI, Codex CLI, or network calls

