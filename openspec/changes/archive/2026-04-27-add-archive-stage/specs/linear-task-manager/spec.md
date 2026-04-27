## ADDED Requirements

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
