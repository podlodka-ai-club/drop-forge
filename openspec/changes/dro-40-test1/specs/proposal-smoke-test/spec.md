## ADDED Requirements

### Requirement: Smoke proposal preserves Linear traceability
The OpenSpec smoke proposal SHALL preserve the source Linear task identity fields that are available in the task payload.

#### Scenario: Minimal Linear task identity is recorded
- **WHEN** the source Linear task has ID `e7119ecd-e524-4b32-ac12-b5dd4dc6db3c`, identifier `DRO-40`, and title `Test1`
- **THEN** the proposal artifacts reference those values as the source context for the change

### Requirement: Smoke proposal avoids invented runtime behavior
The OpenSpec smoke proposal SHALL NOT define runtime code, CLI, API, configuration, or architecture behavior that is not supported by the source Linear task description or comments.

#### Scenario: Source task has no description or comments
- **WHEN** the Linear task description is absent and no comments are available
- **THEN** the proposal scope is limited to OpenSpec smoke-test artifacts and traceability criteria
- **AND** the proposal does not require Go code changes

### Requirement: Smoke proposal is apply-ready as an OpenSpec change
The OpenSpec smoke proposal SHALL include the artifacts required by the repository's spec-driven OpenSpec workflow so that the change can move to implementation review without additional discovery.

#### Scenario: Required proposal artifacts exist
- **WHEN** the smoke proposal is prepared
- **THEN** `proposal.md`, `design.md`, `tasks.md`, and `specs/proposal-smoke-test/spec.md` exist under `openspec/changes/dro-40-test1`

#### Scenario: Smoke proposal can be validated
- **WHEN** a developer validates the change with the OpenSpec CLI
- **THEN** the change satisfies the schema requirements for proposal, design, specs, and tasks artifacts
