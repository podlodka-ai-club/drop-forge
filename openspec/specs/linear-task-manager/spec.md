# linear-task-manager Specification

## Purpose
TBD - created by archiving change add-linear-task-manager. Update Purpose after archive.
## Requirements
### Requirement: Linear task selection is scoped to one configured project
The system SHALL expose a task manager module that loads tasks from Linear only for the configured Linear project and only for the configured workflow state IDs used by the orchestration routes `ready to propose`, `ready to code`, and `ready to archive`.

#### Scenario: Tasks are loaded for managed states inside the configured project
- **WHEN** an external orchestration layer requests managed tasks from the task manager
- **THEN** it requests tasks from Linear for the configured project and managed states and prepares them for orchestration

#### Scenario: Tasks outside the configured project are ignored
- **WHEN** Linear contains tasks in the managed states but outside the configured project
- **THEN** the task manager does not return those tasks to the caller

#### Scenario: No matching tasks exist in the configured project
- **WHEN** Linear has no tasks in the configured project and managed states
- **THEN** the task manager completes the run without executor calls and without mutating any task state

### Requirement: Task payload includes description and comments
The system SHALL return each managed Linear task with the data needed by an external orchestration layer, including the task description and task comments.

#### Scenario: Task details include human feedback comments
- **WHEN** the task manager returns a task from Linear
- **THEN** the returned task includes the issue description and the available comments for that task

#### Scenario: Task without comments is returned consistently
- **WHEN** the task manager returns a task that has no comments in Linear
- **THEN** the returned task includes an empty comments collection and remains valid for downstream processing

#### Scenario: Task without description is returned consistently
- **WHEN** the task manager returns a task that has no description in Linear
- **THEN** the returned task still includes its identity, state, and comments without failing the read operation

#### Scenario: Rejected proposal task is fetched again with comments
- **WHEN** a human adds review comments in Linear and moves the task back to `ready to propose`
- **THEN** the next task fetch returns the same task with the updated comments so an external orchestration layer can pass that feedback into the next proposal attempt

### Requirement: Managed tasks expose the current workflow state
The system SHALL return the current Linear workflow state for each managed task so an external orchestration layer can decide which executor to call.

#### Scenario: Returned task contains current state
- **WHEN** the task manager returns a task from a managed state
- **THEN** the returned task includes the current workflow state identifier or name together with the task payload

### Requirement: Task state transitions can be applied back to Linear
The system SHALL allow a caller to move a managed task to another configured Linear state.

#### Scenario: Managed task is moved to a new state
- **WHEN** a caller requests a state change for a managed task
- **THEN** the task manager updates the task in Linear to the requested target state

#### Scenario: Proposal task is moved to proposal review
- **WHEN** an external orchestration layer completes proposal execution and requests a move to the configured `Need Proposal Review` state
- **THEN** the task manager updates the task in Linear to that review state

#### Scenario: Code task is moved to code review
- **WHEN** an external orchestration layer completes code execution and requests a move to the configured `Need Code Review` state
- **THEN** the task manager updates the task in Linear to that review state

#### Scenario: Archive task is moved to archive review
- **WHEN** an external orchestration layer completes archive execution and requests a move to the configured `Need Archive Review` state
- **THEN** the task manager updates the task in Linear to that review state

#### Scenario: State transition failure is returned with context
- **WHEN** Linear rejects a requested state transition for a managed task
- **THEN** the task manager returns an error that identifies the task and the state transition operation

### Requirement: Task comments can be added back to Linear
The system SHALL allow a caller to publish a comment on a managed Linear task.

#### Scenario: Comment is published for a managed task
- **WHEN** a caller provides a comment body for a managed task
- **THEN** the task manager creates that comment in Linear for the target task

#### Scenario: Comment publish failure is returned with context
- **WHEN** Linear rejects a comment creation request for a managed task
- **THEN** the task manager returns an error that identifies the task and the comment operation

### Requirement: Pull request links can be attached to a task
The system SHALL allow a caller to associate a pull request URL with a managed Linear task.

#### Scenario: Pull request is attached to task
- **WHEN** a caller provides a PR URL for a managed task
- **THEN** the task manager stores the PR association for that task in Linear

#### Scenario: Invalid pull request URL is rejected
- **WHEN** a caller provides an empty or invalid PR URL for a managed task
- **THEN** the task manager returns a validation error before sending the association request to Linear

#### Scenario: Pull request attachment failure is returned with context
- **WHEN** Linear rejects a pull request association request for a managed task
- **THEN** the task manager returns an error that identifies the task and the PR attachment operation

### Requirement: Runtime configuration for Linear task management
The system SHALL read Linear task manager runtime parameters from `.env` and environment variables, including the target Linear project, managed workflow state IDs, and configured review target state IDs, and the repository SHALL keep `.env.example` synchronized with those keys without committed values.

#### Scenario: Linear task manager configuration is present
- **WHEN** the environment contains the required Linear connection, project filter, managed state IDs, and review target state IDs
- **THEN** the task manager uses those values to select tasks and apply workflow transitions

#### Scenario: Required Linear task manager configuration is missing
- **WHEN** a required Linear connection, project filter, managed state ID, or review target state ID is absent
- **THEN** the system returns a configuration error before starting task processing

### Requirement: Testable orchestration dependencies
The task manager module SHALL allow tests to replace Linear access so unit tests do not require real network calls.

#### Scenario: Linear client is substituted in tests
- **WHEN** a unit test constructs the task manager with a fake Linear client
- **THEN** the test can assert project-scoped task selection, returned comments, state transitions, and PR associations without calling the real Linear API

#### Scenario: Partial Linear payloads are covered in tests
- **WHEN** a unit test provides partially filled Linear task data such as missing description or missing comments
- **THEN** the task manager behavior for those payloads can be verified without calling the real Linear API

