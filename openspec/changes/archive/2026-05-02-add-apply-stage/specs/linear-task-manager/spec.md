## ADDED Requirements

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
