## MODIFIED Requirements

### Requirement: README supports repository navigation and development workflow
The root `README.md` SHALL include at least one section that helps developers navigate the repository or continue local development, such as key directories, links to detailed docs, or the standard verification commands used in the project, and SHALL explicitly point readers to the project overview document as the next level of documentation after the initial repository entrypoint.

#### Scenario: Developer looks for more detailed documentation
- **WHEN** the root README gives only a summary of the workflow
- **THEN** it links to `docs/project-overview.md` as the recommended project overview
- **AND** it links to `docs/proposal-runner.md` for detailed behavior and prerequisites

#### Scenario: Developer needs baseline verification commands
- **WHEN** a developer prepares to validate local changes
- **THEN** `README.md` includes the standard project commands for formatting and tests
