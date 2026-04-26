## MODIFIED Requirements

### Requirement: README supports repository navigation and development workflow
The root `README.md` SHALL include at least one section that helps developers navigate the repository or continue local development, such as key directories, links to detailed docs, the project overview document, or the standard verification commands used in the project.

#### Scenario: Developer looks for more detailed documentation
- **WHEN** the root README gives only a summary of the workflow
- **THEN** it links to `docs/proposal-runner.md` for detailed behavior and prerequisites

#### Scenario: Developer looks for a project overview
- **WHEN** a reader needs a connected explanation of the project purpose, key actors, and proposal-stage flow
- **THEN** `README.md` links to the dedicated overview document in `docs/`

#### Scenario: Developer needs baseline verification commands
- **WHEN** a developer prepares to validate local changes
- **THEN** `README.md` includes the standard project commands for formatting and tests
