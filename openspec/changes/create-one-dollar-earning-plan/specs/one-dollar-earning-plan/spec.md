## ADDED Requirements

### Requirement: Repository contains a one dollar earning plan
The repository SHALL contain `docs/one-dollar-earning-plan.md` with a concrete plan for earning the first `$1` from work related to the current `orchv3` project.

#### Scenario: Developer opens the plan
- **WHEN** a developer opens `docs/one-dollar-earning-plan.md`
- **THEN** the document explains the target offer, intended buyer profile, delivery format, and payment confirmation criteria

#### Scenario: Plan stays tied to the project
- **WHEN** the plan describes the value proposition
- **THEN** it connects the offer to the current `orchv3` capability of preparing OpenSpec proposals from Linear task context

### Requirement: Plan is actionable without new product infrastructure
The one dollar earning plan SHALL be executable manually without adding new runtime services, payment integrations, CLI commands, or environment variables.

#### Scenario: Execute first validation manually
- **WHEN** a maintainer follows the plan for the first validation attempt
- **THEN** the required steps can be completed with existing communication, repository, and documentation tools

#### Scenario: Avoid hidden configuration changes
- **WHEN** the implementation adds the earning plan document
- **THEN** `.env.example` remains unchanged unless a future implementation introduces actual runtime configuration

### Requirement: Plan defines measurable success
The one dollar earning plan SHALL define a measurable done condition for earning or validating the first `$1`.

#### Scenario: Confirm successful outcome
- **WHEN** the plan is used to evaluate whether DRO-29 is complete
- **THEN** success is measured by a received payment, a paid invoice or payment link, or explicit written agreement to pay at least `$1` for a specific deliverable

#### Scenario: Track next iteration
- **WHEN** the first attempt does not produce payment or commitment
- **THEN** the document includes a short feedback loop for updating the offer, buyer profile, or outreach message
