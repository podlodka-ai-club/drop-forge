# drop-forge-configuration Specification

## Purpose
TBD - created by archiving change rename-to-drop-forge-and-migrate-config. Update Purpose after archive.
## Requirements
### Requirement: Drop Forge is the default external application identity
The system SHALL use `Drop Forge` as the default external application name in CLI startup messages, structured service identity, documentation, and generated user-facing metadata unless an explicit runtime override is configured.

#### Scenario: Default application name is used
- **WHEN** `APP_NAME` is absent
- **THEN** configuration loading sets the application display name to `Drop Forge`

#### Scenario: Application name override is used
- **WHEN** `APP_NAME` is present
- **THEN** configuration loading uses the configured value as the service/display name

#### Scenario: Startup log names Drop Forge
- **WHEN** the CLI starts the default orchestration runtime without an `APP_NAME` override
- **THEN** the structured startup log identifies the application as `Drop Forge`

### Requirement: Shared orchestration runtime configuration uses Drop Forge environment keys
The system SHALL read shared orchestration runtime settings from `.env` and environment variables using `DROP_FORGE_*` keys for repository URL, base branch, remote name, temporary workspace cleanup, polling interval, and external command paths.

#### Scenario: Shared repository settings are configured
- **WHEN** the environment contains `DROP_FORGE_REPOSITORY_URL`, `DROP_FORGE_BASE_BRANCH`, and `DROP_FORGE_REMOTE_NAME`
- **THEN** proposal, Apply, and Archive runner wiring uses those values for repository operations

#### Scenario: Shared command paths are configured
- **WHEN** the environment contains `DROP_FORGE_GIT_PATH`, `DROP_FORGE_CODEX_PATH`, and `DROP_FORGE_GH_PATH`
- **THEN** GitManager and agent executors use those command paths for their external command invocations

#### Scenario: Shared cleanup setting is configured
- **WHEN** the environment contains `DROP_FORGE_CLEANUP_TEMP`
- **THEN** runner workspace cleanup behavior uses that configured value

#### Scenario: Shared poll interval is configured
- **WHEN** the environment contains `DROP_FORGE_POLL_INTERVAL=1m`
- **THEN** the orchestration monitor waits one minute between passes

### Requirement: Legacy proposal-only shared environment keys are not accepted
The system SHALL reject the removed shared `PROPOSAL_*` environment keys that were replaced by `DROP_FORGE_*` keys and SHALL report validation errors that name the new supported key.

#### Scenario: Legacy repository key is used
- **WHEN** `PROPOSAL_REPOSITORY_URL` is present and `DROP_FORGE_REPOSITORY_URL` is absent
- **THEN** configuration loading fails with an error that names `DROP_FORGE_REPOSITORY_URL`

#### Scenario: Legacy poll interval key is used
- **WHEN** `PROPOSAL_POLL_INTERVAL` is present and `DROP_FORGE_POLL_INTERVAL` is absent
- **THEN** configuration loading fails with an error that names `DROP_FORGE_POLL_INTERVAL`

#### Scenario: Legacy command path key is used
- **WHEN** a removed shared key such as `PROPOSAL_GIT_PATH`, `PROPOSAL_CODEX_PATH`, or `PROPOSAL_GH_PATH` is present without its `DROP_FORGE_*` replacement
- **THEN** configuration loading fails with an error that names the corresponding supported `DROP_FORGE_*` key

### Requirement: Environment template lists only supported keys without values
The repository SHALL keep `.env.example` synchronized with supported Drop Forge configuration keys and SHALL list only variable names without committed default values or secrets.

#### Scenario: Template lists Drop Forge shared keys
- **WHEN** a developer opens `.env.example`
- **THEN** it includes `DROP_FORGE_REPOSITORY_URL`, `DROP_FORGE_BASE_BRANCH`, `DROP_FORGE_REMOTE_NAME`, `DROP_FORGE_CLEANUP_TEMP`, `DROP_FORGE_POLL_INTERVAL`, `DROP_FORGE_GIT_PATH`, `DROP_FORGE_CODEX_PATH`, and `DROP_FORGE_GH_PATH` without values

#### Scenario: Template excludes removed shared proposal keys
- **WHEN** a developer opens `.env.example`
- **THEN** it does not include removed shared keys such as `PROPOSAL_REPOSITORY_URL`, `PROPOSAL_POLL_INTERVAL`, `PROPOSAL_GIT_PATH`, `PROPOSAL_CODEX_PATH`, or `PROPOSAL_GH_PATH`

#### Scenario: Template preserves proposal-specific keys
- **WHEN** a developer opens `.env.example`
- **THEN** it still includes proposal-specific keys `PROPOSAL_BRANCH_PREFIX` and `PROPOSAL_PR_TITLE_PREFIX` without values

### Requirement: Configuration documentation describes the migration boundary
The project documentation SHALL explain which environment keys are shared Drop Forge runtime settings and which keys remain stage-specific.

#### Scenario: README documents shared keys
- **WHEN** a developer reads the configuration section
- **THEN** the documentation lists the `DROP_FORGE_*` shared runtime keys and describes that they apply to proposal, Apply, and Archive stages

#### Scenario: README documents stage-specific keys
- **WHEN** a developer reads the configuration section
- **THEN** the documentation identifies `PROPOSAL_BRANCH_PREFIX`, `PROPOSAL_PR_TITLE_PREFIX`, and `LINEAR_STATE_*` keys as stage-specific settings

