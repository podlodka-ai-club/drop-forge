## MODIFIED Requirements

### Requirement: README documents dependencies and configuration
The root `README.md` SHALL list the required development and runtime dependencies for the current workflow, SHALL describe the role of `.env` and `.env.example`, and SHALL identify the external CLI prerequisites used by the application for GitHub and GitLab repositories.

#### Scenario: Configure local environment
- **WHEN** a developer prepares a local setup
- **THEN** `README.md` points to `.env.example` as the template of supported environment variables and explains that actual values belong in `.env`

#### Scenario: Verify GitHub external tool prerequisites
- **WHEN** a developer configures `PROPOSAL_GIT_PROVIDER=github` or relies on the default provider
- **THEN** `README.md` lists Go, `git`, `codex`, `gh`, and the requirement for authenticated GitHub access to the target repository

#### Scenario: Verify GitLab external tool prerequisites
- **WHEN** a developer configures `PROPOSAL_GIT_PROVIDER=gitlab`
- **THEN** `README.md` lists Go, `git`, `codex`, `glab`, and the requirement for authenticated GitLab access to the target repository

#### Scenario: Understand provider configuration
- **WHEN** a developer reads the configuration documentation
- **THEN** `README.md` explains how `PROPOSAL_GIT_PROVIDER`, `PROPOSAL_GH_PATH`, and `PROPOSAL_GLAB_PATH` affect provider-specific review request operations
