# elk-log-shipping Specification

## Purpose
TBD - created by archiving change 2026-04-24-add-elk-logging-stack. Update Purpose after archive.
## Requirements
### Requirement: Local ELK stack via Docker Compose
The repository SHALL provide a Docker Compose configuration that starts an Elasticsearch, Logstash, and Kibana stack suitable for local demo use.

#### Scenario: Developer starts the stack
- **WHEN** a developer runs the documented `docker compose -f deploy/docker-compose.yml up -d` command from the repository root
- **THEN** Docker starts an Elasticsearch service, a Logstash service, a Kibana service, and a one-shot Kibana setup service
- **AND** each service exposes its port only on `127.0.0.1`
- **AND** Elasticsearch runs in single-node mode with security disabled for development use only

#### Scenario: Container images are pinned to explicit versions
- **WHEN** the compose file is inspected
- **THEN** every image reference uses a concrete tag such as `8.13.4` for the Elastic family
- **AND** no image reference uses the `latest` tag

#### Scenario: Persistent volume for Elasticsearch data
- **WHEN** the stack is stopped with `docker compose down` (without `-v`) and started again
- **THEN** previously indexed events remain available in Elasticsearch

#### Scenario: Clean slate via volume removal
- **WHEN** the developer runs `docker compose -f deploy/docker-compose.yml down -v`
- **THEN** the named `es-data` volume is removed
- **AND** the next start begins with empty Elasticsearch indices

### Requirement: Logstash pipeline for orchestrator events
The Logstash pipeline SHALL accept JSON-line events on TCP port 5000, normalize the event timestamp, and write them to a date-partitioned Elasticsearch index.

#### Scenario: Pipeline parses orchestrator events
- **WHEN** a client writes newline-terminated JSON events to Logstash on port 5000
- **THEN** each event is stored in Elasticsearch index `orchv3-YYYY.MM.dd`
- **AND** the `time` field from the input is used as the event `@timestamp`
- **AND** the `type` field from the input is renamed to `level` in the stored document

### Requirement: Automated Kibana provisioning
The stack SHALL automatically create a Kibana data view named `orchv3-*` so a developer does not need to configure it manually.

#### Scenario: Data view is present after first start
- **WHEN** `kibana-setup` exits successfully after the stack first comes up
- **THEN** Kibana contains a data view titled `orchv3-*` with time field `@timestamp`

#### Scenario: Provisioning is idempotent on restart
- **WHEN** the `kibana-setup` container is recreated against an already-provisioned Kibana
- **THEN** the import succeeds with `overwrite=true`
- **AND** no duplicate saved objects are created

### Requirement: Environment variables for log shipping
The application SHALL expose the sink configuration through environment variables following the existing centralized configuration pattern.

#### Scenario: Developer configures from template
- **WHEN** a developer inspects `.env.example`
- **THEN** the file contains a `# ELK integration` section listing `LOGSTASH_ADDR`, `LOGSTASH_BUFFER_SIZE`, and `LOGSTASH_DIAL_TIMEOUT`
- **AND** the listed keys have no committed values

#### Scenario: Configuration is centralized
- **WHEN** the sink is constructed
- **THEN** every sink parameter is read from `internal/config.LogstashConfig`
- **AND** no Logstash endpoint, buffer size, or dial timeout is hardcoded outside the config package

### Requirement: Operator documentation for the demo workflow
The repository SHALL document the demo workflow so a developer can start, verify, stop, and reset the stack without external knowledge.

#### Scenario: Developer follows the demo documentation
- **WHEN** a developer reads `docs/elk-demo.md`
- **THEN** the document describes the minimum Docker and memory requirements
- **AND** it describes how to start the stack, verify Kibana and Elasticsearch health, and stop the stack
- **AND** it describes the `docker compose down -v` procedure for resetting local state
- **AND** it describes a `nc`-based smoke test and the Kibana dashboard URL

