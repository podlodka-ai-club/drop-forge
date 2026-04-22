## 1. Docker Infrastructure

- [ ] 1.1 Add root `docker-compose.yml` with an `elasticsearch` service using the official Elasticsearch image.
- [ ] 1.2 Configure the service for local single-node mode with security disabled for development.
- [ ] 1.3 Expose the Elasticsearch HTTP API on host port `9200`.
- [ ] 1.4 Add a Docker-managed volume for Elasticsearch data persistence.
- [ ] 1.5 Add a healthcheck that verifies the Elasticsearch HTTP health endpoint responds successfully.

## 2. Application Configuration

- [ ] 2.1 Add `ELASTICSEARCH_URL` and `ELASTICSEARCH_LOG_INDEX` fields to the centralized Go config structure.
- [ ] 2.2 Load the Elastic settings from environment variables in `internal/config.Load`.
- [ ] 2.3 Keep omitted Elastic settings optional so application startup does not require Elasticsearch.
- [ ] 2.4 Update `.env.example` with the Elastic variable keys without default values or secrets.

## 3. Tests

- [ ] 3.1 Extend config tests to assert Elastic settings are returned when environment variables are set.
- [ ] 3.2 Extend config tests to assert omitted Elastic settings are represented as empty values and do not fail config loading.

## 4. Documentation

- [ ] 4.1 Document the Docker Compose command for starting local Elasticsearch.
- [ ] 4.2 Document the Elasticsearch health check command or endpoint.
- [ ] 4.3 Document that the local Compose security settings are development-only and not production settings.
- [ ] 4.4 Document that this change does not automatically ingest application logs into Elasticsearch.

## 5. Verification

- [ ] 5.1 Run `go fmt ./...`.
- [ ] 5.2 Run `go test ./...`.
- [ ] 5.3 If Docker is available, run the documented Compose startup and health check commands.
