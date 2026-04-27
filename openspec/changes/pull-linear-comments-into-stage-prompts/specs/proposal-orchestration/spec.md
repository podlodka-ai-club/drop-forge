## ADDED Requirements

### Requirement: Apply prompt includes Linear task comments

The system SHALL build Apply runner `AgentPrompt` from the selected Linear task's identity, title, description, and comments before executing the Apply runner.

#### Scenario: Ready-to-code task with comments is prepared for Apply

- **WHEN** a ready-to-code task has a title, description, and one or more comments
- **THEN** the Apply runner receives input whose `AgentPrompt` contains the task ID, identifier, title, description, and comments

#### Scenario: Ready-to-code task without comments is prepared for Apply

- **WHEN** a ready-to-code task has no comments
- **THEN** the Apply runner receives input whose `AgentPrompt` remains non-empty
- **AND** the `AgentPrompt` explicitly represents that no comments are available

#### Scenario: Ready-to-code task without description keeps Apply comments

- **WHEN** a ready-to-code task has no description but has comments
- **THEN** the Apply runner receives input whose `AgentPrompt` explicitly represents the missing description
- **AND** the `AgentPrompt` still contains the available comments

#### Scenario: Empty Apply comment body is represented

- **WHEN** a ready-to-code task has a comment with an empty body
- **THEN** the Apply runner receives input whose `AgentPrompt` includes a placeholder for that empty comment instead of dropping the comment

### Requirement: Archive prompt includes Linear task comments

The system SHALL build Archive runner `AgentPrompt` from the selected Linear task's identity, title, description, and comments before executing the Archive runner.

#### Scenario: Ready-to-archive task with comments is prepared for Archive

- **WHEN** a ready-to-archive task has a title, description, and one or more comments
- **THEN** the Archive runner receives input whose `AgentPrompt` contains the task ID, identifier, title, description, and comments

#### Scenario: Ready-to-archive task without comments is prepared for Archive

- **WHEN** a ready-to-archive task has no comments
- **THEN** the Archive runner receives input whose `AgentPrompt` remains non-empty
- **AND** the `AgentPrompt` explicitly represents that no comments are available

#### Scenario: Ready-to-archive task without description keeps Archive comments

- **WHEN** a ready-to-archive task has no description but has comments
- **THEN** the Archive runner receives input whose `AgentPrompt` explicitly represents the missing description
- **AND** the `AgentPrompt` still contains the available comments

#### Scenario: Empty Archive comment body is represented

- **WHEN** a ready-to-archive task has a comment with an empty body
- **THEN** the Archive runner receives input whose `AgentPrompt` includes a placeholder for that empty comment instead of dropping the comment

### Requirement: Stage prompts preserve comment author and time context

The system SHALL include available comment author and creation time metadata in Apply and Archive prompts so review feedback remains traceable during agent execution.

#### Scenario: Comment metadata is available

- **WHEN** a ready-to-code or ready-to-archive task has comments with author and creation time metadata
- **THEN** the runner input `AgentPrompt` includes the comment body together with the available author and creation time

#### Scenario: Comment author is missing

- **WHEN** a ready-to-code or ready-to-archive task has a comment without author display metadata
- **THEN** the runner input `AgentPrompt` includes a stable fallback author marker
- **AND** the comment body remains present in the prompt

#### Scenario: Multiple comments are available

- **WHEN** a ready-to-code or ready-to-archive task has multiple comments
- **THEN** the runner input `AgentPrompt` includes each available comment in deterministic order
