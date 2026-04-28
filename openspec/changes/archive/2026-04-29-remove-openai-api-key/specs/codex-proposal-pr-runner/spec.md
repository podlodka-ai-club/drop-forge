## MODIFIED Requirements

### Requirement: Runtime configuration from environment files
The system SHALL read runtime configuration from `.env` with `github.com/joho/godotenv` and environment variables, including the target GitHub repository, branch settings, external command paths, orchestration polling interval, Linear task manager settings, and logging settings. The system SHALL NOT expose `OPENAI_API_KEY` as an application runtime configuration key unless a future capability introduces direct OpenAI API calls.

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

#### Scenario: OpenAI API key is not part of app config
- **WHEN** `OPENAI_API_KEY` exists in `.env` or the process environment
- **THEN** configuration loading does not expose it through the application `Config`
- **AND** proposal runner behavior remains governed by the configured Codex CLI executable and the CLI's own authentication environment

### Requirement: Environment variable template
The repository SHALL keep `.env.example` synchronized with all supported application configuration keys without secrets or default values. The template SHALL NOT include `OPENAI_API_KEY` while the application does not directly call OpenAI APIs.

#### Scenario: Template lists runtime keys
- **WHEN** a developer needs to configure the proposal runner locally
- **THEN** `.env.example` lists the required application keys without committed values

#### Scenario: Template excludes unused OpenAI API key
- **WHEN** a developer opens `.env.example`
- **THEN** it does not list `OPENAI_API_KEY`
