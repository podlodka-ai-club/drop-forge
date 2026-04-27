## ADDED Requirements

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
The system SHALL provide reusable `GitManager` operations for resolving a pull request head branch, creating a pull request, parsing the created PR URL, and adding a pull request comment through the configured `gh` CLI.

#### Scenario: Pull request branch is resolved
- **WHEN** a runner provides a pull request URL as the branch source
- **THEN** `GitManager` runs `gh pr view <url> --json headRefName --jq .headRefName`
- **AND** returns the non-empty head branch name

#### Scenario: Pull request is created
- **WHEN** a runner asks to create a pull request with base, head, title, and body
- **THEN** `GitManager` runs `gh pr create` with those values
- **AND** returns the PR URL parsed from plain URL, JSON, or mixed command output

#### Scenario: Pull request comment is skipped when empty
- **WHEN** a runner asks to publish an empty or whitespace-only pull request comment body
- **THEN** `GitManager` does not call `gh pr comment`
- **AND** logs that the empty comment was skipped

#### Scenario: Pull request comment is published
- **WHEN** a runner asks to publish a non-empty pull request comment body
- **THEN** `GitManager` runs `gh pr comment <pr-url> --body <body>`
- **AND** logs the comment creation step

### Requirement: GitManager command execution is testable
The `GitManager` package SHALL allow tests to replace external command execution, temporary directory creation, temporary directory removal, current time, stdout, and stderr without invoking real git, GitHub CLI, filesystem cleanup, or network access.

#### Scenario: Command runner is substituted in tests
- **WHEN** a unit test constructs `GitManager` with a fake command runner
- **THEN** the test can assert git and gh command names, arguments, working directories, stdout capture, stderr capture, and errors without executing external commands

#### Scenario: Workspace filesystem hooks are substituted in tests
- **WHEN** a unit test constructs `GitManager` with fake temp directory and removal hooks
- **THEN** the test can assert workspace path handling and cleanup behavior without depending on the host filesystem
