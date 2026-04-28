## ADDED Requirements

### Requirement: Apply and Archive runners delegate repository operations to GitManager
The Apply and Archive runners SHALL use the internal `GitManager` package for repository clone workspace creation, pull request branch resolution, branch checkout, status inspection, staging, commit, and push while preserving their existing orchestration contracts.

#### Scenario: Apply runner uses GitManager for task branch workflow
- **WHEN** the Apply runner receives valid input and the agent produces changes
- **THEN** it delegates clone workspace creation, optional PR branch resolution, branch checkout, `git status --short`, `git add`, `git commit`, and `git push` to `GitManager`
- **AND** it completes without creating a new pull request

#### Scenario: Archive runner uses GitManager for task branch workflow
- **WHEN** the Archive runner receives valid input and the agent produces archive changes
- **THEN** it delegates clone workspace creation, optional PR branch resolution, branch checkout, `git status --short`, `git add`, `git commit`, and `git push` to `GitManager`
- **AND** it completes without creating a new pull request

#### Scenario: Direct branch source bypasses GitHub lookup
- **WHEN** Apply or Archive input contains a non-empty branch name
- **THEN** the runner passes that branch directly to `GitManager` checkout
- **AND** it does not ask `GitManager` to resolve a branch through `gh pr view`

#### Scenario: Apply and Archive no-change behavior is preserved
- **WHEN** the agent succeeds but `GitManager` returns empty short status
- **THEN** the runner returns the existing no-changes error
- **AND** it does not ask `GitManager` to commit or push

#### Scenario: Apply and Archive repository dependency is testable
- **WHEN** a unit test constructs Apply or Archive runner with a fake `GitManager`
- **THEN** the test can assert runner workflow decisions without executing real git, GitHub CLI, Codex CLI, or network calls
