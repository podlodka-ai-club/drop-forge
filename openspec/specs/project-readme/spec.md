# project-readme Specification

## Purpose
TBD - created by archiving change add-project-readme. Update Purpose after archive.
## Requirements
### Requirement: Root README for project entrypoint
The repository SHALL contain a root-level `README.md` that explains what `orchv3` is and what problem the current CLI solves.

#### Scenario: New developer opens repository
- **WHEN** a developer opens the repository root without prior context
- **THEN** `README.md` gives a concise description of the project purpose and the current proposal-runner workflow

#### Scenario: README stays aligned with implementation scope
- **WHEN** `README.md` describes the application capabilities
- **THEN** it only documents behaviors that are present in the current codebase and does not describe unsupported APIs or modes

### Requirement: README documents startup and execution flow
The root `README.md` SHALL describe how to run the application locally, including invocation with a task description argument and invocation through `stdin`, and SHALL explain that the resulting PR URL is written to `stdout` while workflow logs are written to `stderr`.

#### Scenario: Run command with CLI argument
- **WHEN** a developer wants to execute the proposal runner with a direct task description
- **THEN** `README.md` includes an example command that passes the task as CLI arguments

#### Scenario: Run command through stdin
- **WHEN** a developer wants to pipe the task description into the application
- **THEN** `README.md` includes an example command that sends the task through standard input

#### Scenario: Understand command output channels
- **WHEN** a developer reads the run instructions
- **THEN** `README.md` explains which output is intended for `stdout` and which output is intended for `stderr`

### Requirement: README documents dependencies and configuration
The root `README.md` SHALL list the required development and runtime dependencies for the current workflow, SHALL describe the role of `.env` and `.env.example`, and SHALL identify the external CLI prerequisites used by the application.

#### Scenario: Configure local environment
- **WHEN** a developer prepares a local setup
- **THEN** `README.md` points to `.env.example` as the template of supported environment variables and explains that actual values belong in `.env`

#### Scenario: Verify external tool prerequisites
- **WHEN** a developer checks what is needed before running the workflow
- **THEN** `README.md` lists Go, `git`, `codex`, `gh`, and the requirement for authenticated GitHub access to the target repository

### Requirement: README supports repository navigation and development workflow
The root `README.md` SHALL include at least one section that helps developers navigate the repository or continue local development, such as key directories, links to detailed docs, or the standard verification commands used in the project.

#### Scenario: Developer looks for more detailed documentation
- **WHEN** the root README gives only a summary of the workflow
- **THEN** it links to `docs/proposal-runner.md` for detailed behavior and prerequisites

#### Scenario: Developer needs baseline verification commands
- **WHEN** a developer prepares to validate local changes
- **THEN** `README.md` includes the standard project commands for formatting and tests

