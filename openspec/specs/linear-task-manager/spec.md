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
The system SHALL read Linear task manager runtime parameters from `.env` and environment variables, including the target Linear project, managed workflow state IDs (ready-to-propose, ready-to-code, ready-to-archive), in-progress target state IDs, configured human review target state IDs, and the optional AI review target state IDs (`LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`). The repository SHALL keep `.env.example` synchronized with all configured keys without committed values.

#### Scenario: Linear task manager configuration is present
- **WHEN** the environment contains the required Linear connection, project filter, managed state IDs, in-progress target state IDs, and human review target state IDs
- **THEN** the task manager uses those values to select tasks and apply workflow transitions

#### Scenario: Required Linear task manager configuration is missing
- **WHEN** a required Linear connection, project filter, managed state ID, in-progress target state ID, or human review target state ID is absent
- **THEN** the system returns a configuration error before starting task processing

#### Scenario: Environment example lists in-progress workflow states
- **WHEN** a developer opens `.env.example`
- **THEN** it includes keys for `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` without default values

#### Scenario: Environment example lists AI review workflow states
- **WHEN** a developer opens `.env.example`
- **THEN** it includes keys for `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, and `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID` without default values

### Requirement: In-progress state IDs are transition targets only
The task manager SHALL keep configured in-progress workflow state IDs available as transition targets without treating those states as managed input queues for task selection. The task manager SHALL also keep human review state IDs as transition targets only, while the AI review state IDs (when configured) SHALL be both managed input queues and transition targets.

#### Scenario: Managed task selection excludes in-progress states
- **WHEN** the task manager builds the list of managed state IDs for loading tasks from Linear
- **THEN** the list includes ready-to-propose, ready-to-code, and ready-to-archive state IDs
- **AND** the list does not include proposing-in-progress, code-in-progress, or archiving-in-progress state IDs

#### Scenario: Managed task selection excludes human review states
- **WHEN** the task manager builds the list of managed state IDs for loading tasks from Linear
- **THEN** the list does not include need-proposal-review, need-code-review, or need-archive-review state IDs

#### Scenario: Managed task selection includes AI review states when configured
- **WHEN** all three AI review state IDs are configured and the task manager builds the list of managed state IDs
- **THEN** the list includes need-proposal-ai-review, need-code-ai-review, and need-archive-ai-review state IDs

### Requirement: AI review state IDs are conditionally required as a unit
The task manager configuration SHALL accept three AI review workflow state IDs (`LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`) on an all-or-nothing basis. Configuration validation SHALL accept either all three populated together or all three empty together. Any partial combination SHALL be a configuration error.

#### Scenario: All three AI review state IDs populated
- **WHEN** the environment populates all three AI review state IDs
- **THEN** task manager configuration validation succeeds
- **AND** the AI review state IDs become managed input states for the Review orchestration stage

#### Scenario: All three AI review state IDs empty
- **WHEN** the environment leaves all three AI review state IDs empty
- **THEN** task manager configuration validation succeeds
- **AND** producer routes transition tasks directly to their human review state

#### Scenario: Partial AI review configuration is rejected
- **WHEN** at least one AI review state ID is populated and at least one is empty
- **THEN** task manager configuration validation returns an error identifying the partial AI review configuration
- **AND** the orchestrator does not start

### Requirement: AI review state IDs are managed input queues for the Review orchestration stage
When all three AI review state IDs are configured, the task manager SHALL include them in the list of managed state IDs used to load tasks from Linear so the orchestration layer can route those tasks to the Review runner.

#### Scenario: AI review state IDs are loaded as managed tasks
- **WHEN** all three AI review state IDs are configured and an orchestration layer requests managed tasks
- **THEN** the task manager includes the three AI review state IDs in the state IDs used to load tasks

#### Scenario: AI review state IDs are not loaded when feature disabled
- **WHEN** all three AI review state IDs are empty and an orchestration layer requests managed tasks
- **THEN** the task manager does not include any AI review state ID in the state IDs used to load tasks

### Requirement: AI review review-state transitions are applied to Linear
The task manager SHALL allow callers to move a managed task into any of the three AI review state IDs and from any of them into the corresponding human review state.

#### Scenario: Task is moved to need-proposal-ai-review
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Task is moved to need-code-ai-review
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Task is moved to need-archive-ai-review
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Task is moved from AI review to human review
- **WHEN** an orchestration layer completes Review execution and requests a move from an AI review state to the matching human review state
- **THEN** the task manager applies that state transition to Linear

### Requirement: Testable orchestration dependencies
The task manager module SHALL allow tests to replace Linear access so unit tests do not require real network calls.

#### Scenario: Linear client is substituted in tests
- **WHEN** a unit test constructs the task manager with a fake Linear client
- **THEN** the test can assert project-scoped task selection, returned comments, state transitions, and PR associations without calling the real Linear API

#### Scenario: Partial Linear payloads are covered in tests
- **WHEN** a unit test provides partially filled Linear task data such as missing description or missing comments
- **THEN** the task manager behavior for those payloads can be verified without calling the real Linear API

### Requirement: Task payload includes proposal orchestration identity fields

The task manager SHALL return enough stable task identity fields for proposal orchestration to build traceable proposal input, including the Linear task ID, human-readable identifier, and title.

#### Scenario: Returned task includes identity and title

- **WHEN** the task manager returns a managed Linear task
- **THEN** the returned task includes the Linear task ID, identifier, and title together with its current workflow state

#### Scenario: Proposal input can reference the source task

- **WHEN** an external orchestration layer receives a task from the task manager
- **THEN** it can include the task identifier and title in downstream proposal execution input without performing another Linear lookup

### Requirement: Task payload includes pull request branch source
The task manager SHALL return enough pull request metadata for a ready-to-code task to identify the branch that should receive Apply changes. The branch source SHALL be represented as a pull request URL, a branch name, or both.

#### Scenario: Ready-to-code task includes attached pull request URL
- **WHEN** the task manager returns a managed Linear task in `LINEAR_STATE_READY_TO_CODE_ID`
- **THEN** the returned task includes the pull request URL previously attached to the task when one is available

#### Scenario: Task without pull request remains valid
- **WHEN** the task manager returns a managed task that has no pull request attachment
- **THEN** the returned task remains valid
- **AND** the missing pull request branch source is represented explicitly as an empty value

#### Scenario: Multiple pull request attachments are deterministic
- **WHEN** a managed task has multiple pull request attachments
- **THEN** the task manager returns them in a deterministic order or selects a deterministic primary pull request for downstream Apply processing

### Requirement: Code route state IDs are used by Apply orchestration
The task manager SHALL keep `LINEAR_STATE_READY_TO_CODE_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, and `LINEAR_STATE_NEED_CODE_REVIEW_ID` available for Apply orchestration as the input queue, in-progress transition target, and review transition target.

#### Scenario: Ready-to-code tasks are loaded as managed tasks
- **WHEN** an orchestration layer requests managed tasks
- **THEN** the task manager includes `LINEAR_STATE_READY_TO_CODE_ID` in the state IDs used to load tasks

#### Scenario: Code in-progress is a transition target
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_CODE_IN_PROGRESS_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Need code review is a transition target
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_CODE_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Code transition failure is returned with context
- **WHEN** Linear rejects a code-route state transition for a managed task
- **THEN** the task manager returns an error that identifies the task and requested state transition

### Requirement: Archive task payload includes pull request branch source
The task manager SHALL return enough pull request metadata for a ready-to-archive task to identify the branch that should receive OpenSpec Archive changes. The branch source SHALL be represented as a pull request URL, a branch name, or both.

#### Scenario: Ready-to-archive task includes attached pull request URL
- **WHEN** the task manager returns a managed Linear task in `LINEAR_STATE_READY_TO_ARCHIVE_ID`
- **THEN** the returned task includes the pull request URL previously attached to the task when one is available

#### Scenario: Ready-to-archive task includes branch when available
- **WHEN** the task manager returns a ready-to-archive task whose pull request metadata includes a branch name
- **THEN** the returned task includes that branch name for downstream Archive processing

#### Scenario: Ready-to-archive task without pull request remains valid
- **WHEN** the task manager returns a managed ready-to-archive task that has no pull request attachment
- **THEN** the returned task remains valid
- **AND** the missing pull request branch source is represented explicitly as an empty value

#### Scenario: Multiple archive pull request attachments are deterministic
- **WHEN** a managed ready-to-archive task has multiple pull request attachments
- **THEN** the task manager returns them in a deterministic order or selects a deterministic primary pull request for downstream Archive processing

### Requirement: Archive route state IDs are used by Archive orchestration
The task manager SHALL keep `LINEAR_STATE_READY_TO_ARCHIVE_ID`, `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`, and `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` available for Archive orchestration as the input queue, in-progress transition target, and review transition target.

#### Scenario: Ready-to-archive tasks are loaded as managed tasks
- **WHEN** an orchestration layer requests managed tasks
- **THEN** the task manager includes `LINEAR_STATE_READY_TO_ARCHIVE_ID` in the state IDs used to load tasks

#### Scenario: Archiving in-progress is a transition target
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Need archive review is a transition target
- **WHEN** an orchestration layer requests that a task move to `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`
- **THEN** the task manager applies that state transition to Linear

#### Scenario: Archive transition failure is returned with context
- **WHEN** Linear rejects an archive-route state transition for a managed task
- **THEN** the task manager returns an error that identifies the task and requested state transition

### Requirement: Task status transitions publish events
The task manager SHALL publish a `task.status_changed` event after a managed task is successfully moved to another Linear workflow state and SHALL include expanded task context when the caller provides it for that transition.

#### Scenario: Successful move publishes status change event
- **WHEN** a caller requests a task state change through `TaskManager`
- **AND** Linear accepts the state transition
- **THEN** the task manager publishes a `task.status_changed` event containing the task ID and target state ID

#### Scenario: Successful review move publishes task and PR context
- **WHEN** an orchestration stage requests a task move to `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, or `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`
- **AND** the orchestration stage provides the task identifier, task title, target state name, and pull request URL or branch source
- **AND** Linear accepts the state transition
- **THEN** the task manager publishes a `task.status_changed` event containing those provided context fields

#### Scenario: Failed move does not publish status change event
- **WHEN** a caller requests a task state change through `TaskManager`
- **AND** Linear rejects the state transition
- **THEN** the task manager returns the state transition error
- **AND** the task manager does not publish a `task.status_changed` event

#### Scenario: Event publish failure does not revert successful move
- **WHEN** Linear accepts a task state transition
- **AND** publishing the resulting `task.status_changed` event fails
- **THEN** the task manager logs the event publication failure
- **AND** the task manager still reports the state transition as successful to the caller

### Requirement: Task status event publishing is optional and testable
The task manager SHALL allow tests and application wiring to provide an event publisher, and SHALL keep existing task management behavior valid when no publisher is configured.

#### Scenario: Task manager runs without publisher
- **WHEN** a task manager has no event publisher configured
- **AND** Linear accepts a requested state transition
- **THEN** the task manager completes the state transition without failing because of missing event wiring

#### Scenario: Fake publisher captures transition event
- **WHEN** a unit test configures the task manager with a fake event publisher
- **AND** a task state transition succeeds
- **THEN** the test can assert that exactly one `task.status_changed` event was published with the expected task ID and target state ID
