## MODIFIED Requirements

### Requirement: Task payload includes description and comments
The system SHALL return each managed Linear task with the data needed by an external orchestration layer, including the task description and the latest available task comments for proposal, Apply, and Archive stages.

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

#### Scenario: Ready-to-code task is fetched with implementation comments
- **WHEN** a human adds implementation feedback comments in Linear and moves the task to `ready to code`
- **THEN** the next task fetch returns the task with the updated comments so Apply orchestration can pass that feedback into the Apply prompt

#### Scenario: Ready-to-archive task is fetched with archive comments
- **WHEN** a human adds archival feedback comments in Linear and moves the task to `ready to archive`
- **THEN** the next task fetch returns the task with the updated comments so Archive orchestration can pass that feedback into the Archive prompt
