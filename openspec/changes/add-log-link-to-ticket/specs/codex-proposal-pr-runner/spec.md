## MODIFIED Requirements

### Requirement: Runtime configuration from environment files
The system SHALL read runtime configuration from `.env` with `github.com/joho/godotenv` and environment variables, including the target GitHub repository, branch settings, external command paths, and an optional logs URL for the created pull request.

#### Scenario: Repository is configured in env
- **WHEN** `.env` or the process environment contains the target repository setting
- **THEN** the system uses that repository for `git clone`

#### Scenario: Required repository setting is missing
- **WHEN** the target repository setting is absent
- **THEN** the system returns a configuration error before running the proposal workflow

#### Scenario: Environment overrides dot env
- **WHEN** the same configuration key exists in `.env` and in the process environment
- **THEN** the system uses the process environment value

#### Scenario: Dot env syntax is parsed by godotenv
- **WHEN** `.env` contains values that rely on supported godotenv syntax such as quoted strings or inline comments
- **THEN** the system loads those values using godotenv-compatible parsing

#### Scenario: Optional logs URL is configured
- **WHEN** `.env` or the process environment contains `PROPOSAL_LOGS_URL`
- **THEN** the system loads that value into the proposal runner configuration for later use in PR body generation

### Requirement: Environment variable template
The repository SHALL keep `.env.example` synchronized with all supported configuration keys without secrets or default values, including optional proposal runner keys.

#### Scenario: Template lists runtime keys
- **WHEN** a developer needs to configure the proposal runner locally
- **THEN** `.env.example` lists the required keys without committed values

#### Scenario: Template includes optional logs URL key
- **WHEN** a developer wants the created pull request to contain a logs link
- **THEN** `.env.example` includes `PROPOSAL_LOGS_URL` as an available proposal runner setting without a committed value

## ADDED Requirements

### Requirement: Pull request body can include logs link
The system SHALL include a logs link in the created pull request body when the proposal runner configuration provides a non-empty `PROPOSAL_LOGS_URL` value.

#### Scenario: Logs link is appended to PR body
- **WHEN** the proposal workflow creates a pull request and `PROPOSAL_LOGS_URL` is configured
- **THEN** the `gh pr create --body` payload contains the original task description
- **AND** the same body contains a separate logs section with the configured URL

#### Scenario: Logs link is omitted when not configured
- **WHEN** the proposal workflow creates a pull request and `PROPOSAL_LOGS_URL` is empty or whitespace-only
- **THEN** the `gh pr create --body` payload does not contain a logs section
- **AND** the pull request is still created successfully
