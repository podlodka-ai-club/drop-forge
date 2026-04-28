## ADDED Requirements

### Requirement: Proposal orchestration rejects insufficient task context

The system SHALL validate that a ready-to-propose Linear task contains enough human-authored context before moving it to proposing-in-progress or calling the proposal runner. A task SHALL be considered sufficiently described when it has a non-empty description or at least one non-empty comment after trimming whitespace.

#### Scenario: Ready-to-propose task with description is executed

- **WHEN** `TaskManager` returns a ready-to-propose task with a non-empty description
- **THEN** proposal orchestration treats the task as eligible for the normal proposal execution flow

#### Scenario: Ready-to-propose task with comments is executed

- **WHEN** `TaskManager` returns a ready-to-propose task with no description and at least one non-empty comment
- **THEN** proposal orchestration treats the task as eligible for the normal proposal execution flow

#### Scenario: Title-only task is skipped before runner execution

- **WHEN** `TaskManager` returns a ready-to-propose task with an identifier and title but no non-empty description or comments
- **THEN** proposal orchestration does not move the task to proposing-in-progress
- **AND** proposal orchestration does not call the proposal runner
- **AND** proposal orchestration does not attach a pull request URL or move the task to proposal review

#### Scenario: Insufficient task receives feedback comment

- **WHEN** proposal orchestration skips a ready-to-propose task because its context is insufficient
- **THEN** proposal orchestration asks `TaskManager` to add a comment explaining that the task needs a goal, expected behavior, and acceptance criteria

#### Scenario: Insufficient task skip is logged

- **WHEN** proposal orchestration skips a ready-to-propose task because its context is insufficient
- **THEN** the logs include a structured event with the task identity and insufficient-context reason

#### Scenario: Feedback comment failure stops processing

- **WHEN** `TaskManager` fails to add the insufficient-context feedback comment
- **THEN** proposal orchestration returns an error that identifies the task and comment operation
- **AND** proposal orchestration does not call the proposal runner
