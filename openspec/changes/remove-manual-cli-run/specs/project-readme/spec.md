## MODIFIED Requirements

### Requirement: README documents startup and execution flow
The root `README.md` SHALL describe how to run the application locally as a long-running Linear proposal monitor, SHALL explain that proposal tasks are read from the configured `Ready to propose` workflow state, and SHALL no longer document manual task description execution through CLI arguments or `stdin`.

#### Scenario: Run command starts monitoring
- **WHEN** a developer wants to run the proposal-stage orchestrator locally
- **THEN** `README.md` includes an example command that starts the long-running Linear monitoring process

#### Scenario: Understand Linear-driven task input
- **WHEN** a developer reads the run instructions
- **THEN** `README.md` explains that task input comes from Linear tasks in the configured `Ready to propose` state rather than from CLI arguments or `stdin`

#### Scenario: Understand command output channels
- **WHEN** a developer reads the run instructions
- **THEN** `README.md` explains that workflow logs are written through the configured logger and that the CLI does not print a manually generated PR URL to `stdout`
