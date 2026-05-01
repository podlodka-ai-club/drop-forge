## 1. Console Color Writer

- [x] 1.1 Add a small writer helper that buffers newline-terminated JSON log lines and wraps only `type=error` lines with ANSI red/reset when color is enabled.
- [x] 1.2 Make the helper pass through info events, malformed/non-JSON lines, and partial buffered output without changing their content.
- [x] 1.3 Add unit tests for red error output, unchanged info output, unchanged malformed lines, and partial-line flushing behavior.

## 2. CLI Logger Wiring

- [x] 2.1 Update `cmd/orchv3` logger setup so the local stderr writer is colorized only when it is an interactive terminal.
- [x] 2.2 Keep Logstash/TCP sink wiring on raw JSON output so secondary sinks never receive ANSI escape sequences.
- [x] 2.3 Add tests for non-interactive stderr preserving valid JSON and for multiwriter mode sending colored console output while the sink receives raw JSON.

## 3. Verification

- [x] 3.1 Run `go fmt ./...`.
- [x] 3.2 Run `go test ./...`.
- [x] 3.3 Manually verify, where possible, that a CLI error printed to a real terminal appears red while redirected output remains parseable JSON.
