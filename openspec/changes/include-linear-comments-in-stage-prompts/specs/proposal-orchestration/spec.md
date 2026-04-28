## ADDED Requirements

### Requirement: Apply and Archive prompts include Linear comments
The system SHALL preserve Linear task comments in the agent prompt passed to Apply and Archive execution. The prompt SHALL include the same task identity, title, description, and comments block that is available to orchestration, and SHALL explicitly represent the absence of comments.

#### Scenario: Apply prompt includes task comments
- **WHEN** a ready-to-code task has one or more Linear comments
- **THEN** the Apply runner receives an input whose `AgentPrompt` contains the `Comments:` block
- **AND** the prompt contains each available comment body
- **AND** the prompt includes available comment author and creation time metadata

#### Scenario: Archive prompt includes task comments
- **WHEN** a ready-to-archive task has one or more Linear comments
- **THEN** the Archive runner receives an input whose `AgentPrompt` contains the `Comments:` block
- **AND** the prompt contains each available comment body
- **AND** the prompt includes available comment author and creation time metadata

#### Scenario: Apply prompt represents missing comments
- **WHEN** a ready-to-code task has no Linear comments
- **THEN** the Apply runner receives an input whose `AgentPrompt` contains `No comments available.`
- **AND** the Apply runner input remains valid when the task has a branch source

#### Scenario: Archive prompt represents missing comments
- **WHEN** a ready-to-archive task has no Linear comments
- **THEN** the Archive runner receives an input whose `AgentPrompt` contains `No comments available.`
- **AND** the Archive runner input remains valid when the task has a branch source

#### Scenario: Runner Codex prompt keeps orchestration comments
- **WHEN** Apply or Archive runner builds the Codex CLI prompt from orchestration input
- **THEN** the resulting Codex prompt contains the comments already present in `AgentPrompt`
- **AND** the runner does not drop, rewrite, or replace the comments block.
