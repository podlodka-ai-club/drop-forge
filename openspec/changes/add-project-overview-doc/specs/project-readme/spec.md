## MODIFIED Requirements

### Requirement: README supports repository navigation and development workflow
The root `README.md` SHALL include at least one section that helps developers navigate the repository or continue local development, such as key directories, links to detailed docs, or the standard verification commands used in the project. It SHALL link to a dedicated project overview document as the primary quick-start reading path for understanding the project.

#### Scenario: Developer looks for a quick project overview
- **WHEN** a developer opens the root README to understand what the repository contains
- **THEN** `README.md` links to the dedicated project overview document as the main entrypoint for a concise explanation of the project

#### Scenario: Developer looks for more detailed documentation
- **WHEN** the root README gives only a summary of the workflow
- **THEN** it links to `docs/project-overview.md` for the concise project overview and to `docs/proposal-runner.md` for detailed behavior and prerequisites

#### Scenario: Developer needs baseline verification commands
- **WHEN** a developer prepares to validate local changes
- **THEN** `README.md` includes the standard project commands for formatting and tests
