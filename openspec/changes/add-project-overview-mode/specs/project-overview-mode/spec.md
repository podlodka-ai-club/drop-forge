## ADDED Requirements

### Requirement: Explicit project overview requests are handled locally
The CLI SHALL detect explicit informational requests about the current project and return a local project overview instead of starting the proposal workflow.

#### Scenario: Overview request via CLI arguments
- **WHEN** the user starts `orchv3` with an explicit project-overview request such as `Расскажи мне о проекте`
- **THEN** the CLI returns a project overview in `stdout`
- **AND** the process exits successfully
- **AND** the proposal workflow is not started

#### Scenario: Overview request via stdin
- **WHEN** the user sends an explicit project-overview request through `stdin`
- **THEN** the CLI returns a project overview in `stdout`
- **AND** the process exits successfully
- **AND** the proposal workflow is not started

#### Scenario: Change request still uses proposal workflow
- **WHEN** the user provides a non-empty input that describes a change to build or implement rather than a project overview request
- **THEN** the CLI does not route that input to project-overview mode
- **AND** the existing proposal workflow behavior remains available for that input

### Requirement: Project overview is derived from local repository sources
The project overview SHALL be built from local repository artifacts and SHALL summarize the current purpose and usage of the project without requiring network access.

#### Scenario: README is available
- **WHEN** `README.md` is present in the repository
- **THEN** the overview uses it as the primary source for the project description
- **AND** the response summarizes what `orchv3` does today
- **AND** the response points to more detailed documentation when available

#### Scenario: README fallback is needed
- **WHEN** `README.md` is missing or cannot provide enough project context
- **THEN** the overview is built from fallback local sources such as `docs/proposal-runner.md`, `.env.example`, or verified CLI/config artifacts
- **AND** the CLI still returns a non-empty project overview

### Requirement: Overview mode avoids external workflow side effects
The project-overview mode SHALL not require temporary clone workspaces or external command execution.

#### Scenario: Overview mode avoids external tools
- **WHEN** the CLI handles a project-overview request
- **THEN** it does not invoke `git`, `codex`, or `gh`
- **AND** it does not create a temporary clone directory
- **AND** it does not require configured proposal repository access

#### Scenario: Overview mode remains local without network
- **WHEN** the machine has no network connectivity or external CLIs are unavailable
- **THEN** the CLI can still complete a project-overview request using local repository context
