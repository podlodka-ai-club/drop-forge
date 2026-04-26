## ADDED Requirements

### Requirement: Dedicated project overview document
The repository SHALL contain a dedicated overview document that explains what `orchv3` is, what problem it solves, and what scope is currently implemented in the codebase.

#### Scenario: Reader needs a quick explanation of the project
- **WHEN** a developer or reviewer opens the project overview document without prior context
- **THEN** the document gives a concise explanation of the project purpose and the current proposal-stage scope

#### Scenario: Overview stays aligned with real implementation
- **WHEN** the overview document describes current project behavior
- **THEN** it only documents flows and components that are present in the repository and does not describe unsupported runtime behavior as already implemented

### Requirement: Overview distinguishes current system from target architecture
The project overview document SHALL explicitly distinguish between the currently implemented workflow and the broader target architecture described elsewhere in the repository.

#### Scenario: Reader compares implemented flow and roadmap
- **WHEN** a reader uses the overview to understand how the system works today
- **THEN** the document separates the current implemented vertical slice from the target architecture and points to `architecture.md` for deeper architectural context

### Requirement: Overview routes readers to deeper documentation
The project overview document SHALL link to the main deep-dive materials needed to continue reading about the system.

#### Scenario: Reader wants operational details
- **WHEN** a reader finishes the overview and needs more detail
- **THEN** the document links to the relevant workflow, task-management, and architecture documentation instead of duplicating their full contents
