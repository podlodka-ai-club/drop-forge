# documentation-freshness Specification

## Purpose
Define how documentation freshness audits keep public project documentation aligned with current repository behavior, configuration, and active OpenSpec contracts.

## Requirements

### Requirement: Documentation audit covers public project documents
The project SHALL define documentation freshness as an audit of the current public documentation set against the current repository behavior and configuration.

#### Scenario: Public documents are selected for audit
- **WHEN** a documentation freshness audit is performed
- **THEN** the audit includes `README.md`, every Markdown file in `docs/`, `architecture.md`, `.env.example`, and active specs under `openspec/specs/`

#### Scenario: Archived changes are not treated as current behavior
- **WHEN** the audit reads OpenSpec content
- **THEN** archived changes under `openspec/changes/archive/` are used only as historical context and not as the current behavior contract

### Requirement: Documentation matches current CLI behavior
The project documentation SHALL describe only CLI modes and output behavior that are supported by the current Go implementation.

#### Scenario: Direct proposal runner documentation is checked
- **WHEN** the audit compares documentation with the current CLI direct mode
- **THEN** the documentation describes task input through CLI arguments or `stdin`, PR URL output through `stdout`, and workflow logs through `stderr`

#### Scenario: Proposal orchestration documentation is checked
- **WHEN** the audit compares documentation with the current proposal orchestration mode
- **THEN** the documentation describes the `orchestrate-proposals` pass, Linear task selection, PR attachment, and review-state transition according to the current implementation

#### Scenario: Unsupported CLI behavior is found
- **WHEN** documentation describes a CLI mode or output contract that the current code does not support
- **THEN** the documentation is updated to remove or qualify that behavior before the audit is considered complete

### Requirement: Documentation matches runtime configuration
The project documentation SHALL keep runtime configuration references aligned with centralized config loading and `.env.example`.

#### Scenario: Environment variable list is checked
- **WHEN** the audit compares documentation with runtime configuration
- **THEN** every documented environment variable is present in `.env.example` or explicitly described as external to the application

#### Scenario: New or undocumented config key is found
- **WHEN** the current config code supports a runtime variable that is missing from `.env.example`
- **THEN** `.env.example` is updated with the key and without a default value or secret

#### Scenario: Secret or default value is found in env template
- **WHEN** `.env.example` contains a secret, environment URL, token, or default value
- **THEN** the value is removed so the file contains only supported keys

### Requirement: Architecture documentation matches component responsibilities
The architecture documentation SHALL reflect the current component boundaries and implementation mapping for non-trivial orchestration responsibilities.

#### Scenario: Component mapping is checked
- **WHEN** the audit compares `architecture.md` with the current packages
- **THEN** the documented mapping for `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, and `Logger` matches the current implementation state

#### Scenario: Future responsibilities are described
- **WHEN** `architecture.md` mentions future behavior that is not implemented
- **THEN** the text clearly labels it as target or future architecture rather than current implementation

### Requirement: Documentation corrections are minimal and traceable
Documentation freshness work SHALL update only documents that are stale, incomplete, misleading, or missing required configuration keys.

#### Scenario: No discrepancy is found
- **WHEN** a checked document already matches the current code and specs
- **THEN** the implementation does not change that document solely for cosmetic reasons

#### Scenario: Discrepancy is corrected
- **WHEN** the audit finds stale or incomplete documentation
- **THEN** the implementation updates the relevant document with the smallest clear correction and records the checked area in the task checklist

#### Scenario: Code discrepancy is discovered
- **WHEN** the audit finds that code behavior appears inconsistent with active OpenSpec requirements
- **THEN** the implementation either fixes the code with tests if the fix is local and safe, or records a follow-up question instead of silently changing the documented contract
