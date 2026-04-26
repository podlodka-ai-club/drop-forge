## ADDED Requirements

### Requirement: Task payload includes proposal orchestration identity fields

The task manager SHALL return enough stable task identity fields for proposal orchestration to build traceable proposal input, including the Linear task ID, human-readable identifier, and title.

#### Scenario: Returned task includes identity and title

- **WHEN** the task manager returns a managed Linear task
- **THEN** the returned task includes the Linear task ID, identifier, and title together with its current workflow state

#### Scenario: Proposal input can reference the source task

- **WHEN** an external orchestration layer receives a task from the task manager
- **THEN** it can include the task identifier and title in downstream proposal execution input without performing another Linear lookup
