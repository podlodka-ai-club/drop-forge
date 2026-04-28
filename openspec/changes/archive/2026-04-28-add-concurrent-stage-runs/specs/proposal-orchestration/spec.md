## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Existing proposal runner is used as the proposal executor

The system SHALL execute proposals by calling the existing proposal runner contract with one prepared `ProposalInput` per eligible task and SHALL not change the proposal runner's internal git, Codex, PR, or comment workflow as part of proposal orchestration.

#### Scenario: Proposal runner is called for eligible task

- **WHEN** a ready-to-propose task is processed
- **THEN** the orchestration stage calls the proposal runner with the prepared `ProposalInput`

#### Scenario: Multiple proposal tasks are started concurrently

- **WHEN** multiple ready-to-propose tasks are returned in one orchestration pass
- **THEN** the orchestration stage starts proposal processing for each eligible task in a separate goroutine
- **AND** one proposal task does not wait for another proposal task to finish before starting

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
