## ADDED Requirements

### Requirement: Apply prompt includes Linear comments
The system SHALL build Apply runner input so `AgentPrompt` contains the selected Linear task's identifier, title, description, and comments before calling the Apply runner. The comments section SHALL preserve available comment bodies and SHALL explicitly state when no comments are available.

#### Scenario: Ready-to-code task with comments is prepared
- **WHEN** a ready-to-code task has comments in the task payload
- **THEN** the Apply runner receives input whose `AgentPrompt` contains the task identifier, title, description, and comment bodies

#### Scenario: Ready-to-code task without comments is prepared
- **WHEN** a ready-to-code task has no comments in the task payload
- **THEN** the Apply runner receives input whose `AgentPrompt` remains non-empty
- **AND** the `AgentPrompt` explicitly represents that no comments are available

#### Scenario: Apply comments reach the agent executor
- **WHEN** the Apply runner starts agent execution for a prepared task
- **THEN** the task description passed to the agent executor is the Apply input `AgentPrompt` that contains the Linear comments section

### Requirement: Archive prompt includes Linear comments
The system SHALL build Archive runner input so `AgentPrompt` contains the selected Linear task's identifier, title, description, and comments before calling the Archive runner. The comments section SHALL preserve available comment bodies and SHALL explicitly state when no comments are available.

#### Scenario: Ready-to-archive task with comments is prepared
- **WHEN** a ready-to-archive task has comments in the task payload
- **THEN** the Archive runner receives input whose `AgentPrompt` contains the task identifier, title, description, and comment bodies

#### Scenario: Ready-to-archive task without comments is prepared
- **WHEN** a ready-to-archive task has no comments in the task payload
- **THEN** the Archive runner receives input whose `AgentPrompt` remains non-empty
- **AND** the `AgentPrompt` explicitly represents that no comments are available

#### Scenario: Archive comments reach the agent executor
- **WHEN** the Archive runner starts agent execution for a prepared task
- **THEN** the task description passed to the agent executor is the Archive input `AgentPrompt` that contains the Linear comments section
