# git-manager Specification

## Purpose
Описывает общий внутренний пакет для git и GitHub CLI операций, используемый runner'ами оркестратора.
## Requirements
### Requirement: GitManager manages isolated clone workspaces
The system SHALL provide an internal `GitManager` package that creates isolated temporary repository workspaces, clones the configured repository into a `repo` directory inside the workspace, logs workspace lifecycle events, and applies the configured cleanup behavior.

#### Scenario: Workspace clone succeeds
- **WHEN** a runner requests a repository workspace
- **THEN** `GitManager` creates a unique temporary directory
- **AND** clones the configured repository into `<temp-dir>/repo`
- **AND** returns both the temporary directory path and clone directory path to the caller

#### Scenario: Workspace is preserved by default
- **WHEN** a workspace is closed and cleanup is disabled
- **THEN** `GitManager` leaves the temporary directory on disk
- **AND** logs that the workspace was preserved

#### Scenario: Workspace cleanup is enabled
- **WHEN** a workspace is closed and cleanup is enabled
- **THEN** `GitManager` removes the temporary directory
- **AND** logs the cleanup result

#### Scenario: Clone failure is contextual
- **WHEN** `git clone` fails while creating a workspace
- **THEN** `GitManager` returns an error that identifies the clone step
- **AND** forwards command output through structured git logs

### Requirement: GitManager provides reusable git operations
The system SHALL provide reusable `GitManager` operations for checking repository status, checking out existing branches, creating new branches, staging all changes, committing changes, and pushing to the configured remote.

#### Scenario: Status detects repository changes
- **WHEN** a runner asks for short repository status in a clone
- **THEN** `GitManager` runs `git status --short`
- **AND** returns the command output to the caller

#### Scenario: Existing branch checkout
- **WHEN** a runner asks to checkout an existing task branch
- **THEN** `GitManager` runs `git checkout <branch>`
- **AND** wraps failures with the branch name

#### Scenario: New branch checkout
- **WHEN** a runner asks to create a proposal branch
- **THEN** `GitManager` runs `git checkout -b <branch>`
- **AND** wraps failures with the branch name

#### Scenario: Commit and push changes
- **WHEN** a runner asks to commit and push produced changes
- **THEN** `GitManager` runs `git add -A`
- **AND** runs `git commit -m <message>`
- **AND** pushes the requested branch to the configured remote

### Requirement: GitManager provides reusable GitHub CLI operations
The system SHALL provide reusable `GitManager` operations for resolving a review request head branch, creating a review request, parsing the created review request URL, and adding a review request comment through the configured provider CLI. For the GitHub provider the system SHALL use `gh`; for the GitLab provider the system SHALL use `glab`.

#### Scenario: GitHub pull request branch is resolved
- **WHEN** the selected provider is `github` and a runner provides a pull request URL as the branch source
- **THEN** `GitManager` runs `gh pr view <url> --json headRefName --jq .headRefName`
- **AND** returns the non-empty head branch name

#### Scenario: GitLab merge request branch is resolved
- **WHEN** the selected provider is `gitlab` and a runner provides a merge request URL as the branch source
- **THEN** `GitManager` runs `glab mr view <url> --output json`
- **AND** extracts and returns the non-empty source branch from the JSON output

#### Scenario: GitHub pull request is created
- **WHEN** the selected provider is `github` and a runner asks to create a review request with base, head, title, and body
- **THEN** `GitManager` runs `gh pr create` with those values
- **AND** returns the pull request URL parsed from plain URL, JSON, or mixed command output

#### Scenario: GitLab merge request is created
- **WHEN** the selected provider is `gitlab` and a runner asks to create a review request with base, head, title, and body
- **THEN** `GitManager` runs `glab mr create --source-branch <head> --target-branch <base> --title <title> --description <body> --yes`
- **AND** returns the merge request URL parsed from plain URL, JSON, or mixed command output

#### Scenario: Review request comment is skipped when empty
- **WHEN** a runner asks to publish an empty or whitespace-only review request comment body
- **THEN** `GitManager` does not call the provider CLI
- **AND** logs that the empty comment was skipped

#### Scenario: GitHub pull request comment is published
- **WHEN** the selected provider is `github` and a runner asks to publish a non-empty review request comment body
- **THEN** `GitManager` runs `gh pr comment <url> --body <body>`
- **AND** logs the comment creation step

#### Scenario: GitLab merge request comment is published
- **WHEN** the selected provider is `gitlab` and a runner asks to publish a non-empty review request comment body
- **THEN** `GitManager` runs `glab mr note create <url> --message <body>`
- **AND** logs the comment creation step

### Requirement: GitManager command execution is testable
The `GitManager` package SHALL allow tests to replace external command execution, temporary directory creation, temporary directory removal, current time, stdout, and stderr without invoking real git, GitHub CLI, GitLab CLI, filesystem cleanup, or network access.

#### Scenario: Command runner is substituted in tests
- **WHEN** a unit test constructs `GitManager` with a fake command runner
- **THEN** the test can assert git, gh, and glab command names, arguments, working directories, stdout capture, stderr capture, and errors without executing external commands

#### Scenario: Workspace filesystem hooks are substituted in tests
- **WHEN** a unit test constructs `GitManager` with fake temp directory and removal hooks
- **THEN** the test can assert workspace path handling and cleanup behavior without depending on the host filesystem

