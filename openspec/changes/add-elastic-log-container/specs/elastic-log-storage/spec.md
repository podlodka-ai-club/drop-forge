## ADDED Requirements

### Requirement: Local Elasticsearch container

The project SHALL provide a Docker Compose service named `elasticsearch` for local log storage.

#### Scenario: Developer starts local Elastic

- **WHEN** a developer runs the documented Docker Compose command
- **THEN** Docker Compose starts an `elasticsearch` service from the official Elasticsearch image
- **AND** the service runs in single-node mode
- **AND** the service exposes the Elasticsearch HTTP API on the host for local development

#### Scenario: Elastic reports readiness

- **WHEN** the `elasticsearch` service is running
- **THEN** Docker Compose evaluates a healthcheck against the Elasticsearch HTTP API
- **AND** the healthcheck reports healthy only after Elasticsearch can answer health requests

#### Scenario: Elastic data survives container restart

- **WHEN** the `elasticsearch` service is stopped and started again without removing Docker volumes
- **THEN** Elasticsearch data remains stored in a Docker-managed volume

### Requirement: Elastic connection configuration

The application SHALL load Elasticsearch connection settings through the centralized configuration package.

#### Scenario: Elastic environment is configured

- **WHEN** `ELASTICSEARCH_URL` and `ELASTICSEARCH_LOG_INDEX` are set in the environment
- **THEN** `internal/config.Load` returns these values as part of the application configuration

#### Scenario: Elastic environment is omitted

- **WHEN** `ELASTICSEARCH_URL` or `ELASTICSEARCH_LOG_INDEX` is omitted
- **THEN** `internal/config.Load` still succeeds
- **AND** the missing value is represented as an empty configuration field

#### Scenario: Environment template lists Elastic keys

- **WHEN** a developer opens `.env.example`
- **THEN** it lists `ELASTICSEARCH_URL` and `ELASTICSEARCH_LOG_INDEX`
- **AND** the listed keys do not contain secret values or default values

### Requirement: Local Elastic documentation

The project SHALL document the minimal local workflow for starting and checking Elasticsearch.

#### Scenario: Developer verifies local Elastic

- **WHEN** a developer follows the repository documentation
- **THEN** the documentation provides the Docker Compose command to start Elasticsearch
- **AND** the documentation provides a command or endpoint for checking Elasticsearch health
- **AND** the documentation states that the local Compose security settings are not production settings

### Requirement: No log ingestion behavior yet

The change SHALL NOT make application startup depend on Elasticsearch or automatically send application logs to Elasticsearch.

#### Scenario: Application starts without Elastic

- **WHEN** Elasticsearch is not running
- **AND** the application is started with its existing required configuration
- **THEN** application startup is not blocked by the absence of Elasticsearch

#### Scenario: Logs are not automatically indexed

- **WHEN** the application writes logs through its existing logging path
- **THEN** the change does not require those logs to be sent to Elasticsearch automatically
