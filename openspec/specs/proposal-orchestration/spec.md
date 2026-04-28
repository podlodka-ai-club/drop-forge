# proposal-orchestration Specification

## Purpose
TBD - created by archiving change add-proposal-orchestrator. Update Purpose after archive.
## Requirements
### Requirement: Proposal orchestration uses ready-to-propose tasks

The system SHALL provide a proposal orchestration stage that loads managed tasks through `TaskManager` and processes only tasks whose workflow state ID matches `LINEAR_STATE_READY_TO_PROPOSE_ID`.

#### Scenario: Ready-to-propose task is selected

- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_PROPOSE_ID`
- **THEN** the proposal orchestration stage treats that task as eligible for proposal execution

#### Scenario: Non-proposal managed task is skipped

- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-code or ready-to-archive
- **THEN** the proposal orchestration stage does not call the proposal runner for that task

#### Scenario: No ready-to-propose tasks exist

- **WHEN** `TaskManager` returns no tasks in `LINEAR_STATE_READY_TO_PROPOSE_ID`
- **THEN** the proposal orchestration stage completes without calling the proposal runner and without mutating task state

### Requirement: Proposal input is built from Linear task payload

The system SHALL build the proposal runner input as a structured `ProposalInput { Title, Identifier, AgentPrompt }` value derived from the selected task's identifier, title, description, and comments. The `Title` SHALL be the task's title (or a non-empty fallback when the task has no title), the `Identifier` SHALL be the task's Linear identifier (or empty if absent), and the `AgentPrompt` SHALL be a multi-line block containing the task identifier, title, description, and comments so the generated OpenSpec proposal has enough task context.

#### Scenario: Task with description and comments is prepared

- **WHEN** a ready-to-propose task has a title, description, and comments
- **THEN** the proposal runner receives a `ProposalInput` whose `Title` equals the task title, `Identifier` equals the task identifier, and `AgentPrompt` contains the task identifier, title, description, and comments

#### Scenario: Task without description is prepared

- **WHEN** a ready-to-propose task has no description
- **THEN** the proposal runner still receives a `ProposalInput` whose `AgentPrompt` is non-empty and contains the task identifier, title, and any available comments

#### Scenario: Task without comments is prepared

- **WHEN** a ready-to-propose task has no comments
- **THEN** the proposal runner input remains valid and the `AgentPrompt` explicitly represents that no review comments are available

#### Scenario: Task without title falls back to placeholder

- **WHEN** a ready-to-propose task has an empty title
- **THEN** the proposal runner receives a `ProposalInput` whose `Title` is set to a non-empty fallback so the runner does not reject the input

### Requirement: Existing proposal runner is used as the proposal executor

The system SHALL execute proposals by calling the existing proposal runner contract with one prepared `ProposalInput` per eligible task and SHALL not change the proposal runner's internal git, Codex, PR, or comment workflow as part of proposal orchestration.

#### Scenario: Proposal runner is called for eligible task

- **WHEN** a ready-to-propose task is processed
- **THEN** the orchestration stage calls the proposal runner with the prepared `ProposalInput`

#### Scenario: Multiple proposal tasks are started concurrently

- **WHEN** multiple ready-to-propose tasks are returned in one orchestration pass
- **THEN** the orchestration stage starts proposal processing for each eligible task in a separate goroutine
- **AND** one proposal task does not wait for another proposal task to finish before starting

### Requirement: Successful proposal updates the Linear task

The system SHALL move a ready-to-propose task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID` before executing the proposal runner, then attach the proposal PR URL to the task and move the task to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID` after the proposal runner succeeds.

#### Scenario: Proposal task enters in-progress state before execution

- **WHEN** a ready-to-propose task is selected for proposal processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the proposal runner until that transition succeeds

#### Scenario: Proposal task reaches review state

- **WHEN** the proposal runner returns a PR URL for a task moved to proposing-in-progress
- **THEN** the orchestration stage asks `TaskManager` to attach that PR URL to the task
- **AND** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`

#### Scenario: Proposal transitions happen in order

- **WHEN** a ready-to-propose task is successfully processed
- **THEN** the orchestration stage moves the task to proposing-in-progress before running the proposal runner
- **AND** the orchestration stage attaches the PR URL before moving the task to the proposal review state

### Requirement: Proposal orchestration preserves task state on failure

The system SHALL return contextual errors, avoid running proposal work when the initial in-progress transition fails, and avoid moving a task to proposal review when proposal execution or PR attachment fails.

#### Scenario: Proposing in-progress transition fails

- **WHEN** `TaskManager` fails to move a ready-to-propose task to `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation
- **AND** the orchestration stage does not call the proposal runner, attach a PR URL, or move the task to proposal review

#### Scenario: Proposal runner fails

- **WHEN** the proposal runner returns an error for a task already moved to proposing-in-progress
- **THEN** the orchestration stage returns an error that identifies the task
- **AND** the orchestration stage does not attach a PR URL or move the task to proposal review

#### Scenario: PR attachment fails

- **WHEN** the proposal runner returns a PR URL but `TaskManager` fails to attach it to the task
- **THEN** the orchestration stage returns an error that identifies the task and PR attachment operation
- **AND** the orchestration stage does not move the task to proposal review

#### Scenario: Proposal review transition fails

- **WHEN** the PR URL is attached but `TaskManager` fails to move the task to proposal review
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation

### Requirement: Proposal orchestration emits structured logs

The system SHALL log proposal orchestration decisions and outcomes using the existing structured logger format.

#### Scenario: Task processing is logged

- **WHEN** the orchestration stage starts processing a ready-to-propose task
- **THEN** the logs include a structured event with the orchestration module and task identity

#### Scenario: Task skip is logged

- **WHEN** the orchestration stage skips a managed task because it is not in the ready-to-propose state
- **THEN** the logs include a structured event with the task identity and current state

#### Scenario: Task processing fails

- **WHEN** proposal orchestration fails for a task
- **THEN** the logs include a structured error event with task identity and failure context

### Requirement: Proposal orchestration runs as a continuous monitor

The system SHALL provide a continuous Drop Forge orchestration monitoring loop that repeatedly executes orchestration for tasks in managed Linear states, including `LINEAR_STATE_READY_TO_PROPOSE_ID`, until its context is cancelled.

#### Scenario: Monitor starts orchestration polling

- **WHEN** the CLI starts without unsupported manual task input
- **THEN** the system loads config, wires `TaskManager`, proposal runner, Apply runner, Archive runner, logger, and orchestration dependencies
- **AND** the system starts a continuous Drop Forge orchestration monitoring loop

#### Scenario: Monitor repeats after successful pass

- **WHEN** an orchestration pass completes successfully
- **THEN** the monitor waits for the configured polling interval
- **AND** the monitor starts the next orchestration pass

#### Scenario: Monitor continues after pass failure

- **WHEN** an orchestration pass returns an error after the monitor has started
- **THEN** the monitor logs a structured error event with the orchestration failure context
- **AND** the monitor waits for the configured polling interval before starting the next pass

#### Scenario: Monitor stops on context cancellation

- **WHEN** the monitor context is cancelled while waiting between passes
- **THEN** the monitor exits without starting another orchestration pass

### Requirement: Proposal polling interval is runtime configurable

The system SHALL read the Drop Forge orchestration monitor polling interval from `.env` and environment variables using `DROP_FORGE_POLL_INTERVAL`, validate that it is a positive duration, and keep `.env.example` synchronized without a committed value.

#### Scenario: Poll interval is configured

- **WHEN** the environment contains `DROP_FORGE_POLL_INTERVAL=1m`
- **THEN** the orchestration monitor waits one minute between orchestration passes

#### Scenario: Poll interval uses default

- **WHEN** `DROP_FORGE_POLL_INTERVAL` is absent
- **THEN** the orchestration monitor uses the code-defined default polling interval

#### Scenario: Poll interval is invalid

- **WHEN** `DROP_FORGE_POLL_INTERVAL` is not a valid positive duration
- **THEN** configuration loading returns an error before orchestration monitoring starts

#### Scenario: Environment example lists poll interval

- **WHEN** a developer opens `.env.example`
- **THEN** it includes `DROP_FORGE_POLL_INTERVAL` without a default value

### Requirement: Proposal orchestration is available from the CLI

The system SHALL expose Drop Forge orchestration as the default CLI runtime by starting a continuous orchestration monitoring loop and SHALL reject the removed direct proposal runner mode for task descriptions passed through args or stdin.

#### Scenario: CLI starts orchestration monitoring mode

- **WHEN** the user invokes the CLI without a manual proposal task description
- **THEN** the CLI loads config, wires `TaskManager`, proposal runner, Apply runner, Archive runner, logger, and orchestration dependencies
- **AND** the CLI starts the continuous Drop Forge orchestration monitoring loop

#### Scenario: CLI direct proposal args are rejected

- **WHEN** the user provides a task description through CLI arguments
- **THEN** the CLI returns a usage error explaining that manual proposal execution is unsupported in Drop Forge
- **AND** the CLI does not call the proposal runner

#### Scenario: CLI direct proposal stdin is rejected

- **WHEN** the user provides a task description through stdin
- **THEN** the CLI returns a usage error explaining that manual proposal execution is unsupported in Drop Forge
- **AND** the CLI does not call the proposal runner

#### Scenario: Legacy one-pass command is not the public runtime

- **WHEN** the user invokes the previous one-pass proposal orchestration command
- **THEN** the CLI rejects the command as unsupported or treats it as invalid manual input
- **AND** the CLI does not run a one-pass orchestration mode as public behavior

### Requirement: Proposal orchestration dependencies are testable

The proposal orchestration stage SHALL allow tests to replace task management and proposal execution dependencies without network access, Codex CLI, GitHub CLI, or Linear API calls.

#### Scenario: Task manager is substituted in tests

- **WHEN** a unit test constructs proposal orchestration with a fake task manager
- **THEN** the test can assert task filtering, PR attachment, and state transition behavior without Linear API calls

#### Scenario: Proposal runner is substituted in tests

- **WHEN** a unit test constructs proposal orchestration with a fake proposal runner
- **THEN** the test can assert proposal execution behavior without Codex CLI, GitHub CLI, git, or network calls

### Requirement: Archive orchestration uses ready-to-archive tasks
The system SHALL provide an Archive orchestration stage that loads managed tasks through `TaskManager` and processes tasks whose workflow state ID matches `LINEAR_STATE_READY_TO_ARCHIVE_ID`.

#### Scenario: Ready-to-archive task is selected
- **WHEN** `TaskManager` returns a task whose state ID equals `LINEAR_STATE_READY_TO_ARCHIVE_ID`
- **THEN** the orchestration stage treats that task as eligible for Archive execution

#### Scenario: Non-archive managed task is skipped by Archive route
- **WHEN** `TaskManager` returns a task from another managed state such as ready-to-propose or ready-to-code
- **THEN** the Archive route does not call the Archive runner for that task

#### Scenario: Proposal Apply and Archive tasks are processed in one pass
- **WHEN** one orchestration pass receives ready-to-propose, ready-to-code, and ready-to-archive tasks
- **THEN** the system routes each task to the executor matching its current state
- **AND** the system processes eligible tasks concurrently instead of sequentially in the returned order

### Requirement: Archive input is built from Linear task payload
The system SHALL build Archive runner input from the selected task's identity, title, description, comments, and associated task branch source. The task branch source SHALL be either a concrete branch name or a pull request URL from which the Archive runner can resolve the branch.

#### Scenario: Ready-to-archive task with PR URL is prepared
- **WHEN** a ready-to-archive task includes an associated pull request URL
- **THEN** the Archive runner receives input containing the task identity, task context, and pull request URL

#### Scenario: Ready-to-archive task with branch is prepared
- **WHEN** a ready-to-archive task includes a concrete branch name
- **THEN** the Archive runner receives input containing the task identity, task context, and branch name

#### Scenario: Ready-to-archive task without branch source is rejected
- **WHEN** a ready-to-archive task has no branch name and no associated pull request URL
- **THEN** the orchestration stage returns a contextual error for that task
- **AND** the orchestration stage does not call the Archive runner
- **AND** the orchestration stage does not move the task to archive review

### Requirement: Archive runner archives the OpenSpec change on the task branch
The system SHALL execute Archive by cloning the configured repository into a temporary directory, checking out the task branch, running archival through the OpenSpec Archive skill, committing produced changes, and pushing the task branch.

#### Scenario: Archive runner uses isolated temporary clone
- **WHEN** the Archive runner starts for a valid ready-to-archive task
- **THEN** it creates a temporary workspace separate from the operator checkout
- **AND** it clones the configured repository into that workspace
- **AND** it checks out the task branch before running archival

#### Scenario: Archive runner pushes archival changes
- **WHEN** OpenSpec Archive produces repository changes
- **THEN** the Archive runner stages the changes
- **AND** commits them with a task-specific commit message
- **AND** pushes the commit to the task branch

#### Scenario: Archive runner does not create a new pull request
- **WHEN** Archive execution succeeds
- **THEN** the Archive runner returns success without creating a new pull request

#### Scenario: Archive runner fails when archival produces no changes
- **WHEN** OpenSpec Archive completes but `git status --short` shows no repository changes
- **THEN** the Archive runner returns an error that identifies the no-change condition
- **AND** it does not commit or push

### Requirement: Successful Archive updates the Linear task
The system SHALL move a ready-to-archive task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` before executing the Archive runner and move it to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` after the Archive runner succeeds.

#### Scenario: Archive task enters in-progress state before execution
- **WHEN** a ready-to-archive task is selected for Archive processing
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- **AND** the orchestration stage does not call the Archive runner until that transition succeeds

#### Scenario: Archive task reaches review state after push
- **WHEN** the Archive runner succeeds for a task moved to archiving-in-progress
- **THEN** the orchestration stage asks `TaskManager` to move the task to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`

#### Scenario: Archive transitions happen in order
- **WHEN** a ready-to-archive task is successfully processed
- **THEN** the orchestration stage moves the task to archiving-in-progress before running the Archive runner
- **AND** the orchestration stage moves the task to archive review only after the Archive runner succeeds

### Requirement: Archive orchestration preserves task state on failure
The system SHALL return contextual errors, avoid running Archive work when the initial in-progress transition fails, and avoid moving a task to archive review when Archive execution fails.

#### Scenario: Archiving in-progress transition fails
- **WHEN** `TaskManager` fails to move a ready-to-archive task to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation
- **AND** the orchestration stage does not call the Archive runner or move the task to archive review

#### Scenario: Archive runner fails
- **WHEN** the Archive runner returns an error for a task already moved to archiving-in-progress
- **THEN** the orchestration stage returns an error that identifies the task and Archive operation
- **AND** the orchestration stage does not move the task to archive review

#### Scenario: Archive review transition fails
- **WHEN** the Archive runner succeeds but `TaskManager` fails to move the task to archive review
- **THEN** the orchestration stage returns an error that identifies the task and state transition operation

### Requirement: Archive orchestration emits structured logs
The system SHALL log Archive orchestration decisions and outcomes using the existing structured logger format.

#### Scenario: Archive task processing is logged
- **WHEN** the orchestration stage starts processing a ready-to-archive task
- **THEN** the logs include a structured event with the orchestration module and task identity

#### Scenario: Archive task processing fails
- **WHEN** Archive orchestration fails for a task
- **THEN** the logs include a structured error event with task identity and failure context

### Requirement: Orchestration dependencies support Archive tests
The Archive orchestration stage SHALL allow tests to replace task management and Archive execution dependencies without network access, Codex CLI, GitHub CLI, git, or Linear API calls.

#### Scenario: Archive runner is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake Archive runner
- **THEN** the test can assert Archive execution behavior without Codex CLI, GitHub CLI, git, or network calls

#### Scenario: Archive task manager is substituted in tests
- **WHEN** a unit test constructs orchestration with a fake task manager
- **THEN** the test can assert Archive task filtering and state transition behavior without Linear API calls

### Requirement: Orchestration pass runs eligible tasks concurrently
The system SHALL start each eligible proposal, Apply, and Archive task in its own goroutine within a single orchestration pass, while continuing to route each task by its current Linear workflow state.

#### Scenario: Tasks from different routes run concurrently
- **WHEN** one orchestration pass receives ready-to-propose, ready-to-code, and ready-to-archive tasks
- **THEN** the system starts proposal, Apply, and Archive processing in separate goroutines
- **AND** no route waits for another route's runner to finish before starting its own eligible task

#### Scenario: Multiple tasks from one route run concurrently
- **WHEN** one orchestration pass receives multiple eligible tasks for the same route
- **THEN** the system starts each eligible task in a separate goroutine
- **AND** the route does not require the previous task from that route to finish before starting the next eligible task

#### Scenario: Non-managed task is not started
- **WHEN** one orchestration pass receives a task whose state does not match proposal, Apply, or Archive input states
- **THEN** the system logs the skip decision
- **AND** it does not start a processing goroutine for that task

### Requirement: Orchestration pass waits for concurrent tasks
The system SHALL wait for all task-processing goroutines started in a pass before the pass returns and before the continuous monitor starts the next polling wait.

#### Scenario: Pass waits for slow task
- **WHEN** one started task finishes quickly and another started task is still running
- **THEN** the orchestration pass does not return until the slow task also finishes

#### Scenario: Loop polls only after pass completion
- **WHEN** a continuous monitor iteration starts concurrent task processing
- **THEN** the monitor waits for the orchestration pass to finish
- **AND** only then waits for the configured polling interval before starting the next iteration

### Requirement: Concurrent task failures are aggregated
The system SHALL collect contextual errors from all failed task goroutines and return an aggregated pass error after all started goroutines finish.

#### Scenario: One task fails while another succeeds
- **WHEN** one concurrent task returns an error and another concurrent task succeeds
- **THEN** the orchestration pass waits for both tasks
- **AND** returns an error that includes the failed task context

#### Scenario: Multiple tasks fail
- **WHEN** multiple concurrent tasks return errors in the same pass
- **THEN** the orchestration pass waits for all started tasks
- **AND** returns an aggregated error that preserves each failed task's context

#### Scenario: Failed task does not cancel sibling task
- **WHEN** one concurrent task returns an error while another task is still running
- **THEN** the system does not cancel the sibling task solely because of that error
- **AND** the sibling task can still complete its normal route workflow

### Requirement: Per-task workflow ordering is preserved during concurrent execution
The system SHALL keep each individual task's existing state transition and runner ordering even when multiple tasks run concurrently.

#### Scenario: Proposal task keeps internal order
- **WHEN** a ready-to-propose task is processed concurrently with other tasks
- **THEN** the orchestration stage moves the task to proposing-in-progress before running the proposal runner
- **AND** attaches the PR URL before moving the task to proposal review

#### Scenario: Apply task keeps internal order
- **WHEN** a ready-to-code task is processed concurrently with other tasks
- **THEN** the orchestration stage moves the task to code-in-progress before running the Apply runner
- **AND** moves the task to code review only after the Apply runner succeeds

#### Scenario: Archive task keeps internal order
- **WHEN** a ready-to-archive task is processed concurrently with other tasks
- **THEN** the orchestration stage moves the task to archiving-in-progress before running the Archive runner
- **AND** moves the task to archive review only after the Archive runner succeeds

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

