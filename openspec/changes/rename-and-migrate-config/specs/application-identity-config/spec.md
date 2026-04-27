## ADDED Requirements

### Requirement: Application identity is centralized
The system SHALL define the public application identity in one configuration location and SHALL use that identity for the default app name, service labels, CLI-facing text, temporary workspace prefixes, and documentation references that describe the current product.

#### Scenario: Default identity uses the selected product name
- **WHEN** configuration loading runs without an explicit `APP_NAME`
- **THEN** the default application name is `Catmarine`

#### Scenario: Explicit app name still overrides the default
- **WHEN** `.env` or the process environment contains `APP_NAME`
- **THEN** the system uses that value as the runtime application name

#### Scenario: Workspace prefixes reflect the new identity
- **WHEN** proposal, apply, or archive runners create temporary workspaces using default settings
- **THEN** the temporary directory prefix uses the new application identity instead of `orchv3`

### Requirement: Catmarine configuration namespace is primary
The system SHALL read shared repository, external command, cleanup, and polling runtime settings from `CATMARINE_*` environment keys as the primary configuration namespace.

#### Scenario: Repository URL is read from the new namespace
- **WHEN** the environment contains `CATMARINE_REPOSITORY_URL`
- **THEN** the system uses that value as the target repository URL

#### Scenario: Runner command paths are read from the new namespace
- **WHEN** the environment contains `CATMARINE_GIT_PATH`, `CATMARINE_CODEX_PATH`, and `CATMARINE_GH_PATH`
- **THEN** the system uses those values for external command execution

#### Scenario: Poll interval is read from the new namespace
- **WHEN** the environment contains `CATMARINE_POLL_INTERVAL=1m`
- **THEN** the orchestration monitor waits one minute between passes

#### Scenario: Environment example lists primary keys
- **WHEN** a developer opens `.env.example`
- **THEN** `.env.example` lists the supported `CATMARINE_*` runtime keys without committed values

### Requirement: Legacy proposal configuration remains compatible
The system SHALL keep legacy `PROPOSAL_*` keys as deprecated aliases for the migrated `CATMARINE_*` settings until a separate breaking change removes them.

#### Scenario: Legacy repository URL is accepted
- **WHEN** `CATMARINE_REPOSITORY_URL` is absent and `PROPOSAL_REPOSITORY_URL` is present
- **THEN** the system uses `PROPOSAL_REPOSITORY_URL` as the target repository URL

#### Scenario: New key wins over legacy key
- **WHEN** both `CATMARINE_REPOSITORY_URL` and `PROPOSAL_REPOSITORY_URL` are present with different values
- **THEN** the system uses `CATMARINE_REPOSITORY_URL`

#### Scenario: Legacy poll interval validation is preserved
- **WHEN** `CATMARINE_POLL_INTERVAL` is absent and `PROPOSAL_POLL_INTERVAL` contains an invalid duration
- **THEN** configuration loading returns a validation error before orchestration starts

#### Scenario: Legacy command path aliases are accepted
- **WHEN** `CATMARINE_GIT_PATH`, `CATMARINE_CODEX_PATH`, or `CATMARINE_GH_PATH` are absent and the matching `PROPOSAL_*` key is present
- **THEN** the system uses the legacy command path value

### Requirement: Configuration migration is documented and testable
The repository SHALL document the mapping between legacy `PROPOSAL_*` keys and primary `CATMARINE_*` keys and SHALL include tests for new-key loading, legacy fallback, and conflict precedence.

#### Scenario: Migration mapping is documented
- **WHEN** a developer reads the configuration documentation
- **THEN** it identifies each migrated `PROPOSAL_*` key and its `CATMARINE_*` replacement

#### Scenario: New namespace loading is covered by tests
- **WHEN** a developer changes config loading for migrated settings
- **THEN** tests fail if `CATMARINE_*` keys are no longer accepted

#### Scenario: Legacy fallback is covered by tests
- **WHEN** a developer removes fallback handling for `PROPOSAL_*`
- **THEN** tests fail for the documented compatibility scenarios

#### Scenario: Conflict precedence is covered by tests
- **WHEN** a developer changes precedence between new and legacy keys
- **THEN** tests fail unless `CATMARINE_*` remains preferred
