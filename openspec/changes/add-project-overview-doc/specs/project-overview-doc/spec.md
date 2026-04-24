## ADDED Requirements

### Requirement: Dedicated project overview document
The repository SHALL contain a dedicated project overview document that explains what `orchv3` is, what problem it solves today, and how a new developer should approach the codebase.

#### Scenario: New developer needs a project introduction
- **WHEN** a developer wants to understand the repository without reading the source code first
- **THEN** the repository provides a single overview document focused on project purpose, current scope, and developer orientation

### Requirement: Overview document describes the current end-to-end workflow
The project overview document SHALL describe the currently implemented flow from task input to pull request creation using only behaviors that are present in the codebase.

#### Scenario: Reader follows the current execution path
- **WHEN** a developer reads the overview document
- **THEN** the document explains that the CLI accepts a task description from command-line arguments or `stdin`
- **AND** it explains that configuration is loaded from environment and `.env`
- **AND** it explains that `proposalrunner` clones the target repository, invokes Codex to generate an OpenSpec proposal, commits the result, pushes a branch, and creates a GitHub pull request

### Requirement: Overview document maps key repository areas and boundaries
The project overview document SHALL identify the main packages and directories relevant to the current workflow and SHALL state the important boundaries of the current implementation.

#### Scenario: Reader needs navigation and limitations
- **WHEN** a developer uses the overview document to orient themselves in the repository
- **THEN** the document names the key areas responsible for CLI entrypoint, configuration loading, proposal workflow orchestration, and supporting documentation
- **AND** it states that the current product scope is a Go CLI for proposal-runner automation rather than a general orchestration platform or HTTP service
