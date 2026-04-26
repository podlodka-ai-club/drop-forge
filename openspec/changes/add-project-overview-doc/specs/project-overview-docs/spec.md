## ADDED Requirements

### Requirement: Project overview document exists at the repository level
The repository SHALL contain a dedicated project overview document in `docs/` that explains the purpose of `orchv3` and the current scope of the system in plain project-level language.

#### Scenario: Reader needs a quick project overview
- **WHEN** a reader opens the project documentation to understand what `orchv3` is
- **THEN** the repository provides a dedicated overview document under `docs/`
- **AND** that document explains the project purpose and current scope without requiring the reader to inspect source code first

### Requirement: Project overview explains current workflow and major roles
The project overview document SHALL describe the current proposal-stage workflow and identify the major internal roles involved in that flow, including `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, and `Logger`.

#### Scenario: Reader wants to understand how the project works today
- **WHEN** a reader uses the overview document to understand the system behavior
- **THEN** the document summarizes the current proposal-stage workflow
- **AND** it names the major internal roles and their responsibilities at a high level

### Requirement: Project overview routes the reader to deeper documentation
The project overview document SHALL link to the detailed documents that expand on specific areas, including the root `README.md`, `architecture.md`, `docs/proposal-runner.md`, and `docs/linear-task-manager.md`.

#### Scenario: Reader needs more detail after the overview
- **WHEN** a reader finishes the overview document and needs implementation or subsystem detail
- **THEN** the document points to the deeper project documents for architecture, proposal runner behavior, and Linear task management

### Requirement: Project overview stays aligned with implemented behavior
The project overview document SHALL describe only capabilities and workflows that exist in the current repository and SHALL avoid presenting future ideas or target architecture as already implemented behavior.

#### Scenario: Overview is updated alongside the current project state
- **WHEN** the overview document describes the project
- **THEN** it reflects the implemented behavior and currently accepted documentation
- **AND** it does not claim unsupported runtime modes, APIs, or orchestration features as current functionality
