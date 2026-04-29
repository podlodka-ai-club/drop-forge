## ADDED Requirements

### Requirement: Proposal runner delegates repository operations to GitManager
The proposal runner SHALL use the internal `GitManager` package for repository clone workspace creation, status inspection, proposal branch checkout, staging, commit, push, pull request creation, and final agent response comments while preserving the existing proposal workflow contract.

#### Scenario: Proposal runner uses GitManager for clone and PR workflow
- **WHEN** the proposal runner receives a valid proposal input and the agent produces changes
- **THEN** it delegates clone workspace creation, `git status --short`, proposal branch creation, `git add`, `git commit`, `git push`, `gh pr create`, and optional `gh pr comment` to `GitManager`
- **AND** it returns the created PR URL to the caller

#### Scenario: Proposal metadata remains stage-specific
- **WHEN** the proposal runner prepares branch name, PR title, PR body, and commit message
- **THEN** it derives those values from the existing proposal input and config rules before passing them to `GitManager`
- **AND** `GitManager` does not derive proposal metadata from Linear task fields or agent prompts

#### Scenario: Proposal no-change behavior is preserved
- **WHEN** the agent succeeds but `GitManager` returns empty short status
- **THEN** the proposal runner returns the existing no-changes error
- **AND** it does not ask `GitManager` to commit, push, create a PR, or comment on a PR

#### Scenario: Proposal runner repository dependency is testable
- **WHEN** a unit test constructs the proposal runner with a fake `GitManager`
- **THEN** the test can assert proposal workflow decisions without executing real git, GitHub CLI, Codex CLI, or network calls
