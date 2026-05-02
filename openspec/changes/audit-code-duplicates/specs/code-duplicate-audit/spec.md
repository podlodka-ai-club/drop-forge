## ADDED Requirements

### Requirement: Duplicate audit report documents findings
The repository SHALL include a duplicate-code audit report that covers production Go code and relevant test code, identifies meaningful duplicate clusters, and records a recommended action for each cluster.

#### Scenario: Audit report is created
- **WHEN** the duplicate audit change is implemented
- **THEN** the repository contains `docs/duplicate-code-audit.md`
- **AND** the report includes the audit date, scope, methodology, and limitations

#### Scenario: Duplicate cluster is documented
- **WHEN** the audit identifies a meaningful duplicate cluster
- **THEN** the report lists the affected files or packages
- **AND** classifies the duplicate as mechanical, stage-specific, test-support, or intentionally accepted
- **AND** records a recommended action and rationale

#### Scenario: No immediate fix is recommended
- **WHEN** a duplicate is intentionally accepted or deferred
- **THEN** the report explains why immediate extraction is not recommended
- **AND** identifies what future signal would justify revisiting the decision

### Requirement: Duplicate fixes preserve observable behavior
Any code changes made while addressing duplicate clusters SHALL preserve existing CLI behavior, orchestration state transitions, git workflow, OpenSpec workflow, runtime configuration, and public package contracts.

#### Scenario: Mechanical duplicate is extracted
- **WHEN** implementation extracts a mechanical duplicate into a shared helper
- **THEN** existing tests for all affected packages continue to pass
- **AND** the refactor does not introduce new runtime configuration or external service calls

#### Scenario: Stage-specific duplicate is left explicit
- **WHEN** a duplicate encodes different stage semantics such as prompt wording, commit message prefix, PR behavior, or result contract
- **THEN** implementation leaves the stage-specific behavior explicit unless a shared abstraction keeps those differences visible

### Requirement: Audit implementation avoids new mandatory external tooling
The duplicate audit SHALL NOT add a new mandatory external duplicate-detection dependency to `go.mod`, CI, or the required verification commands unless the change explicitly documents why standard Go tooling and repository-local inspection are insufficient.

#### Scenario: Audit uses local inspection
- **WHEN** implementation performs the duplicate audit
- **THEN** it may use repository-local commands and standard Go tooling
- **AND** it does not require developers to install a new tool to run the existing project verification commands

#### Scenario: External tool is proposed but not required
- **WHEN** the audit recommends a specialized duplicate detector for future use
- **THEN** the recommendation is documented as optional follow-up
- **AND** the current change does not add it as a required build, test, or runtime dependency

### Requirement: Verification covers performed refactors
The duplicate audit change SHALL include tests or existing-test justification for every logic-affecting refactor performed as part of the accepted fixes.

#### Scenario: Shared helper behavior changes ownership
- **WHEN** duplicate helper logic is moved to a common package
- **THEN** that common package has direct unit test coverage or existing caller tests demonstrate equivalent behavior

#### Scenario: No code refactor is performed
- **WHEN** the audit produces only documentation and recommendations
- **THEN** the final implementation notes explain why additional tests were not needed beyond running the standard project verification commands
