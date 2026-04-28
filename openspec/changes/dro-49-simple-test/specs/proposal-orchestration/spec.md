## ADDED Requirements

### Requirement: Low-context Linear proposal tasks are handled explicitly

The system SHALL treat a ready-to-propose Linear task with an identifier and title but without description or comments as valid proposal input, and SHALL make the missing context explicit in the `ProposalInput.AgentPrompt` so the generated OpenSpec proposal records assumptions instead of inventing unstated product behavior.

#### Scenario: Minimal Linear task is prepared for proposal generation

- **WHEN** a ready-to-propose task has identifier `DRO-49`, title `Просто тест`, no description, and no comments
- **THEN** the proposal orchestration stage prepares a non-empty `ProposalInput`
- **AND** the `ProposalInput.Identifier` equals `DRO-49`
- **AND** the `ProposalInput.Title` equals `Просто тест`
- **AND** the `ProposalInput.AgentPrompt` explicitly states that no description was provided
- **AND** the `ProposalInput.AgentPrompt` explicitly states that no comments are available

#### Scenario: Minimal Linear task keeps traceable source metadata

- **WHEN** the proposal runner receives input for a low-context Linear task
- **THEN** the input contains the Linear identifier and title in the human-readable prompt context
- **AND** downstream generated OpenSpec artifacts can trace the proposal back to the source task without another Linear lookup

#### Scenario: Low-context task does not trigger special workflow routing

- **WHEN** a low-context task is in `LINEAR_STATE_READY_TO_PROPOSE_ID`
- **THEN** proposal orchestration processes it through the same proposal route as other ready-to-propose tasks
- **AND** no separate Linear state, CLI mode, or runtime configuration is required
