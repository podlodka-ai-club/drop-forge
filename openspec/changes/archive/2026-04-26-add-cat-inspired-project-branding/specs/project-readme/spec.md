## MODIFIED Requirements

### Requirement: Root README for project entrypoint
The repository SHALL contain a root-level `README.md` that presents `Purrch` as the public project name, shows the project logo, and explains that the current technical CLI `orchv3` runs the proposal-runner workflow.

#### Scenario: New developer opens repository
- **WHEN** a developer opens the repository root without prior context
- **THEN** `README.md` gives a concise description of the project purpose and the current proposal-runner workflow

#### Scenario: README stays aligned with implementation scope
- **WHEN** `README.md` describes the application capabilities
- **THEN** it only documents behaviors that are present in the current codebase and does not describe unsupported APIs or modes

#### Scenario: README presents the public brand
- **WHEN** a developer opens the repository root
- **THEN** `README.md` shows the `Purrch` name and project logo before the detailed workflow documentation
- **AND** it keeps the current `orchv3` CLI name visible for command usage
