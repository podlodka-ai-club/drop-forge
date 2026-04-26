## MODIFIED Requirements

### Requirement: Proposal input is built from Linear task payload

The system SHALL build the proposal runner input from the selected task's identifier, title, description, and comments so the generated OpenSpec proposal has enough task context. The prepared input SHALL explicitly preserve the source task ID, human-readable identifier, title, description, and available comments in a readable structure.

#### Scenario: Task with description and comments is prepared

- **WHEN** a ready-to-propose task has a title, description, and comments
- **THEN** the proposal runner receives an input string containing the task identifier, title, description, and comments

#### Scenario: Test Linear task context is prepared

- **WHEN** a ready-to-propose task has ID `f5f622b6-b706-4d83-acec-b4a59876ea30`, identifier `DRO-28`, title `Тестовая задача`, description `Проверка как подтягиваетеся описание`, and a comment `Проверка как тянутся комменты`
- **THEN** the proposal runner receives an input string that explicitly contains that ID, identifier, title, description, and comment

#### Scenario: Task without description is prepared

- **WHEN** a ready-to-propose task has no description
- **THEN** the proposal runner still receives a non-empty input string containing the task identifier, title, and any available comments

#### Scenario: Task without comments is prepared

- **WHEN** a ready-to-propose task has no comments
- **THEN** the proposal runner input remains valid and explicitly represents that no review comments are available
