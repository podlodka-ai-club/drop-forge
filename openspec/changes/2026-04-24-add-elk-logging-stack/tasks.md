## 1. Structured Logging Extension

- [ ] 1.1 Add optional `service` field to `Event` with `omitempty` JSON tag.
- [ ] 1.2 Add `NewWithService(out io.Writer, service string) Logger` constructor; keep `New(out)` working as `NewWithService(out, "")`.
- [ ] 1.3 Extend `logger_test.go` with coverage for `service`-present and `service`-omitted cases.

## 2. TCP Sink

- [ ] 2.1 Create `internal/steplog/tcp_sink.go` with `TCPSink` type, bounded channel, bufio writer, goroutine-driven drain.
- [ ] 2.2 Make `Write` non-blocking using `select`+`default`; count dropped payloads via `atomic.Uint64`.
- [ ] 2.3 Add exponential backoff reconnect (1s → 30s) inside the drain goroutine.
- [ ] 2.4 Implement `Close()` with best-effort flush of pending queue within `closeFlushTimeout = 2s`.
- [ ] 2.5 Add periodic drop-summary warning goroutine (default 30s tick).
- [ ] 2.6 Add tests: delivery, non-blocking, overflow drop counter, reconnect after server restart, close-flushes-pending, periodic drop warning.

## 3. Configuration

- [ ] 3.1 Add `LogstashConfig{Addr, BufferSize, DialTimeout}` type and `Config.Logstash` field in `internal/config`.
- [ ] 3.2 Parse `LOGSTASH_ADDR` (optional), `LOGSTASH_BUFFER_SIZE` (default 1024, must be ≥ 1), `LOGSTASH_DIAL_TIMEOUT` (default 2s, parsed via `time.ParseDuration`).
- [ ] 3.3 Add `durationFromEnv` helper next to `intFromEnv`/`boolFromEnv`.
- [ ] 3.4 Extend `config_test.go` with defaults, custom values, invalid buffer size (non-int and ≤ 0), invalid duration cases.
- [ ] 3.5 Append `# ELK integration` section with three empty keys to `.env.example`.

## 4. CLI Wiring

- [ ] 4.1 Create `cmd/orchv3/logger_setup.go` with `buildLogger(stderr, cfg, warnOut) (steplog.Logger, io.Closer, error)` helper.
- [ ] 4.2 When `cfg.Logstash.Addr == ""` return `steplog.NewWithService(stderr, cfg.AppName)` and `nil` closer; otherwise return multi-writer logger and the sink as closer.
- [ ] 4.3 Add `cmd/orchv3/logger_setup_test.go` covering both branches (disabled and enabled with local TCP listener).
- [ ] 4.4 Replace `steplog.New(stderr)` in `cmd/orchv3/main.go:run` with `buildLogger(stderr, cfg, os.Stderr)`; defer sink `Close` when returned.

## 5. Deployment (docker-compose)

- [ ] 5.1 Add `deploy/docker-compose.yml` with four services: `elasticsearch`, `logstash`, `kibana`, `kibana-setup`.
- [ ] 5.2 Pin all image tags to concrete versions (`8.13.4` Elastic family, `curlimages/curl:8.7.1`). No `latest`.
- [ ] 5.3 Configure ES single-node with `xpack.security.enabled=false`, `ES_JAVA_OPTS=-Xms512m -Xmx512m`, named `es-data` volume, healthcheck on `/_cluster/health`.
- [ ] 5.4 Bind Logstash `5000`, Kibana `5601`, ES `9200` to `127.0.0.1` only. Logstash depends on ES healthcheck; Kibana depends on ES healthcheck and exposes its own `/api/status` healthcheck.
- [ ] 5.5 Add `deploy/logstash/pipeline/orchv3.conf` (input TCP json_lines → date filter parsing `time` to `@timestamp` → rename `type` to `level` → output ES index `orchv3-%{+YYYY.MM.dd}`).
- [ ] 5.6 Add `deploy/kibana/setup.sh` that waits for Kibana status and POSTs saved-objects import with `overwrite=true`.
- [ ] 5.7 Add `deploy/kibana/saved-objects.ndjson` with data view `orchv3-*` (time field `@timestamp`). Dashboard export is a follow-up after one-time manual creation in Kibana UI.
- [ ] 5.8 `kibana-setup` service bind-mounts setup.sh and saved-objects.ndjson, depends on kibana health, uses `restart: "no"`.

## 6. Documentation

- [ ] 6.1 Add `docs/elk-demo.md` covering requirements (Docker, ≥ 2 GB RAM, ports), start/stop commands, health checks, volume reset via `down -v`, `nc` smoke-test, orchestrator wiring via `LOGSTASH_ADDR`, dashboard export workflow, and `jq` stderr-purity check.

## 7. Verification

- [ ] 7.1 `go fmt ./...` returns empty diff.
- [ ] 7.2 `go test ./... -timeout 120s` passes for all packages.
- [ ] 7.3 `go build ./...` succeeds.
- [ ] 7.4 `docker compose -f deploy/docker-compose.yml config --quiet` validates without error (if Docker available).
- [ ] 7.5 `grep -n 'image:' deploy/docker-compose.yml | grep -i ':latest'` returns no matches.
- [ ] 7.6 Manual smoke (if Docker available): `docker compose up -d` → `kibana-setup` exits 0 → `nc 127.0.0.1 5000 <<< '<event>'` → event visible in Kibana Discover within 5s.
- [ ] 7.7 Manual smoke: `jq -c . < run.log` on a real orchestrator run reports zero parse errors.
