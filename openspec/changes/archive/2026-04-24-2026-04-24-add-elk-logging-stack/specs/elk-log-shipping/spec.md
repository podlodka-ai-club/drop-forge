## ADDED Requirements

### Requirement: Optional TCP log sink to Logstash
The system SHALL provide an optional log sink that forwards every structured log event to a configured Logstash TCP endpoint in addition to the existing stderr stream.

#### Scenario: Sink disabled when address is empty
- **WHEN** the application starts with `LOGSTASH_ADDR` empty or unset
- **THEN** the logger writes events only to stderr as it does today
- **AND** no TCP connection attempt is made
- **AND** no sink-related warning is written to stderr

#### Scenario: Sink delivers events when address is configured
- **WHEN** the application starts with `LOGSTASH_ADDR` pointing to a reachable `host:port`
- **THEN** every log event is written to stderr as a JSON line
- **AND** the same JSON line is delivered over TCP to the configured endpoint within five seconds of the log call on a healthy network

#### Scenario: Sink does not block the main flow
- **WHEN** the Logstash endpoint is unreachable
- **THEN** calls to the logger return without waiting for a network round trip
- **AND** the orchestrator continues processing its workflow without hangs

### Requirement: Bounded buffer with drop-oldest overflow semantics
The sink SHALL use an in-memory bounded queue whose capacity is configured by `LOGSTASH_BUFFER_SIZE` and SHALL count dropped events when the queue is full instead of blocking the caller.

#### Scenario: Default buffer size
- **WHEN** `LOGSTASH_BUFFER_SIZE` is unset
- **THEN** the sink uses a buffer capacity of 1024 events

#### Scenario: Write drops events on overflow
- **WHEN** the queue is full and a new event is written
- **THEN** the write call returns without error and without blocking
- **AND** an internal drop counter is incremented by one

#### Scenario: Invalid buffer size fails configuration load
- **WHEN** `LOGSTASH_BUFFER_SIZE` is not a positive integer
- **THEN** application configuration loading returns an error
- **AND** the binary exits with a non-zero status before any log I/O is performed

### Requirement: Reconnect with exponential backoff
The sink SHALL transparently reconnect to Logstash after connection loss, without requiring a restart of the orchestrator.

#### Scenario: Startup before Logstash is available
- **WHEN** the orchestrator starts before Logstash is reachable
- **THEN** the orchestrator does not crash
- **AND** the sink periodically retries the dial with exponential backoff capped at thirty seconds
- **AND** a single warning line is emitted to stderr indicating the sink is unavailable
- **AND** events produced meanwhile are buffered up to the configured capacity

#### Scenario: Logstash restarts during orchestrator run
- **WHEN** the Logstash endpoint becomes unreachable while the orchestrator is running
- **THEN** subsequent events continue to be accepted by the logger without error
- **AND** the sink re-establishes the connection when the endpoint becomes reachable again
- **AND** events still present in the queue are delivered after reconnect

### Requirement: Dial timeout configurable via `LOGSTASH_DIAL_TIMEOUT`
The sink SHALL enforce a per-dial timeout configured by `LOGSTASH_DIAL_TIMEOUT` with a default of two seconds.

#### Scenario: Default dial timeout
- **WHEN** `LOGSTASH_DIAL_TIMEOUT` is unset
- **THEN** each dial attempt uses a timeout of two seconds

#### Scenario: Invalid dial timeout fails configuration load
- **WHEN** `LOGSTASH_DIAL_TIMEOUT` is not a valid Go duration string
- **THEN** application configuration loading returns an error

### Requirement: Graceful close flushes pending events
The sink SHALL expose a `Close` operation that drains pending events to the current connection within a bounded time window before returning.

#### Scenario: Close flushes queued events
- **WHEN** the application calls sink Close while events are still in the queue and a connection is established
- **THEN** remaining events are written to the connection before Close returns
- **AND** the total time spent flushing does not exceed two seconds

#### Scenario: Close does not hang on a dead network
- **WHEN** the sink has no live connection at the time of Close
- **THEN** Close returns within the flush budget
- **AND** the orchestrator shutdown proceeds without blocking

### Requirement: Periodic drop summary in stderr
The sink SHALL periodically publish a summary of dropped events to the warning writer so operators notice sustained overflow.

#### Scenario: Drop summary is emitted after overflow
- **WHEN** the dropped counter increases during a polling interval
- **THEN** a single summary line containing the number of dropped events since the last report is written to the warning writer

#### Scenario: No summary when there are no drops
- **WHEN** no events are dropped during a polling interval
- **THEN** no drop-summary line is written

## ADDED Requirements

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
