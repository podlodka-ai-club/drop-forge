## ADDED Requirements

### Requirement: Runtime settings come from environment variables
The application SHALL read runtime configuration from environment variables loaded through `.env` and MUST NOT hardcode secrets, environment URLs, database URLs, tokens, or runtime ports in application logic.

#### Scenario: Backend needs database connection
- **WHEN** the backend initializes Prisma
- **THEN** the database connection string is read from an environment variable

#### Scenario: Frontend needs API base URL
- **WHEN** the frontend calls the backend
- **THEN** it uses `VITE_API_URL` from its environment configuration as the API base URL

#### Scenario: Runtime values differ by developer
- **WHEN** two developers use different local credentials or ports
- **THEN** they can configure those values in their local `.env` files without changing tracked source code

### Requirement: Environment template lists keys without values
The root `.env.example` SHALL list every supported runtime environment variable key required by Docker, backend, frontend, and Prisma, and SHALL NOT include default values or secrets.

#### Scenario: New variable is introduced
- **WHEN** implementation adds a new runtime environment variable
- **THEN** `.env.example` is updated with the variable key and no value

#### Scenario: Template is checked for secrets
- **WHEN** `.env.example` is reviewed
- **THEN** it contains only variable names or empty assignments and no passwords, tokens, URLs with credentials, or environment-specific values

### Requirement: Configuration loading is documented
The root README SHALL explain how `.env` and `.env.example` are used by Docker, backend, frontend, and Prisma in local development.

#### Scenario: Developer prepares local environment
- **WHEN** a developer configures the project for the first time
- **THEN** README points to `.env.example` as the supported key list and explains that real values belong in `.env`

#### Scenario: Developer runs Prisma commands
- **WHEN** a developer runs Prisma migration, seed, or Studio commands
- **THEN** README identifies the environment variables Prisma needs to connect to PostgreSQL
