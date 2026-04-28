## ADDED Requirements

### Requirement: Task comments support orchestration feedback

The task manager SHALL allow orchestration routes to publish deterministic, human-actionable feedback comments on managed Linear tasks when a task cannot proceed automatically.

#### Scenario: Proposal feedback comment is published

- **WHEN** proposal orchestration asks the task manager to publish an insufficient-context feedback comment for a managed task
- **THEN** the task manager creates that comment on the target Linear task

#### Scenario: Duplicate proposal feedback is avoided

- **WHEN** the managed task already contains the same insufficient-context feedback comment
- **THEN** the task manager or calling orchestration flow does not create a duplicate comment for the same reason

#### Scenario: Feedback comment failure includes task context

- **WHEN** Linear rejects an orchestration feedback comment for a managed task
- **THEN** the task manager returns an error that identifies the task and comment operation
