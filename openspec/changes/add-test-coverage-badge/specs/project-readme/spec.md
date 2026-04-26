## ADDED Requirements

### Requirement: README displays test coverage percentage
The root `README.md` SHALL display the current test coverage percentage for the Go codebase in a visible location near the project overview or verification section.

#### Scenario: Repository visitor checks project quality signal
- **WHEN** a developer or reviewer opens the repository root on GitHub
- **THEN** `README.md` shows the test coverage percentage without requiring them to run local commands first

#### Scenario: Developer wants to refresh coverage value
- **WHEN** a developer reads the README coverage information
- **THEN** `README.md` explains the command or documented workflow used to recalculate the percentage

#### Scenario: README avoids unsupported automation claims
- **WHEN** `README.md` displays the coverage percentage
- **THEN** it does not imply automatic CI publication unless that automation exists in the codebase
