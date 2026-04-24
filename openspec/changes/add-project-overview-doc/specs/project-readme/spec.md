## MODIFIED Requirements

### Requirement: README supports repository navigation and development workflow
The root `README.md` SHALL include sections that help developers navigate the repository or continue local development, including a clear link to the dedicated project overview document, key directories, links to detailed docs, or the standard verification commands used in the project.

#### Scenario: Developer looks for a high-level introduction
- **WHEN** a developer opens the root README to understand where to start
- **THEN** `README.md` links to the dedicated project overview document as the primary place to learn the project context and structure

#### Scenario: Developer looks for more detailed workflow documentation
- **WHEN** the root README gives only a summary of the workflow
- **THEN** it links to `docs/proposal-runner.md` for detailed behavior and prerequisites

#### Scenario: Developer needs baseline verification commands
- **WHEN** a developer prepares to validate local changes
- **THEN** `README.md` includes the standard project commands for formatting and tests
