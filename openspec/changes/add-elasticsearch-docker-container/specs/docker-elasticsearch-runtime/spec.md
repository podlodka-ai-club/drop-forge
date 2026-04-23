## ADDED Requirements

### Requirement: Local Elasticsearch compose service
The repository SHALL provide a Docker Compose configuration for local development that starts exactly one Elasticsearch container suitable for the application development workflow.

#### Scenario: Developer starts Elasticsearch locally
- **WHEN** a developer runs the documented Docker Compose command from the repository root
- **THEN** Docker starts one Elasticsearch container for local development
- **AND** the container exposes the Elasticsearch HTTP endpoint on a documented local port

#### Scenario: Container starts in development-safe single-node mode
- **WHEN** the Elasticsearch container is created from the repository Compose configuration
- **THEN** it runs in single-node mode
- **AND** it uses a named Docker volume for persistent local data
- **AND** it includes a healthcheck that can determine when the service is ready to accept HTTP requests

### Requirement: Local Elasticsearch connection configuration
The system SHALL expose Elasticsearch connection settings through centralized environment configuration so the application can connect to the Dockerized local service without hardcoded endpoints.

#### Scenario: Developer configures the application from template
- **WHEN** a developer inspects `.env.example`
- **THEN** the template includes the required Elasticsearch connection keys without committed secret values

#### Scenario: Application uses configured Elasticsearch endpoint
- **WHEN** the application loads runtime configuration with Elasticsearch settings present in `.env` or process environment
- **THEN** it reads the configured Elasticsearch endpoint from centralized config code
- **AND** it does not require a hardcoded localhost URL in business logic

### Requirement: Local Elasticsearch operating guide
The repository SHALL document the local Elasticsearch workflow so a developer can start, stop, verify, and reset the service without external tribal knowledge.

#### Scenario: Developer follows startup instructions
- **WHEN** a developer reads the project documentation for Elasticsearch
- **THEN** the documentation describes how to start the container
- **AND** how to verify that Elasticsearch is healthy and reachable by HTTP

#### Scenario: Developer resets local Elasticsearch state
- **WHEN** a developer needs a clean local Elasticsearch state
- **THEN** the documentation describes how to stop the container and remove the persisted Docker volume
