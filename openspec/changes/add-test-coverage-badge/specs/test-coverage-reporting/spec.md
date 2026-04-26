## ADDED Requirements

### Requirement: Reproducible Go test coverage calculation
The project SHALL define a reproducible way to calculate the total test coverage percentage for the Go codebase using the standard Go toolchain.

#### Scenario: Developer calculates coverage locally
- **WHEN** a developer follows the documented coverage command from the repository root
- **THEN** the command calculates coverage for all Go packages included by `go test ./...`
- **AND** the command produces a total percentage that can be copied into the visible project documentation

#### Scenario: Coverage calculation avoids runtime configuration
- **WHEN** the coverage command is executed for local verification
- **THEN** it does not require secrets, Linear credentials, GitHub credentials, or additional runtime values beyond what the existing test suite requires

### Requirement: Coverage artifacts stay local
The project SHALL treat generated coverage profile files as local verification artifacts rather than committed source files.

#### Scenario: Coverage profile is generated
- **WHEN** a developer runs the documented coverage calculation
- **THEN** any generated coverage profile file is excluded from normal source changes or is clearly documented as a disposable local artifact
