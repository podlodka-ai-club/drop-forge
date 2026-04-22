## 1. Structured Logger

- [ ] 1.1 Add JSON log event model in `internal/steplog` with `time`, `module`, `type`, and `message` fields.
- [ ] 1.2 Implement `Infof` and `Errorf` helpers that write newline-terminated JSON objects to the configured writer.
- [ ] 1.3 Normalize empty module names to `unknown` and write timestamps in UTC RFC3339Nano format.
- [ ] 1.4 Add a line-oriented writer adapter for wrapping external command stdout/stderr as JSON log events.
- [ ] 1.5 Add unit tests for info/error events, required fields, timestamp parsing, module normalization, multiline messages, and quoted messages.

## 2. Workflow Integration

- [ ] 2.1 Replace proposal runner step logs with structured `steplog` calls and stable module names.
- [ ] 2.2 Wrap Codex, git, and gh stdout/stderr streams so forwarded command output is emitted as JSON log events while preserving buffers needed for parsing.
- [ ] 2.3 Emit `error` log events for workflow failures that occur after logging initialization.
- [ ] 2.4 Update command execution logging so command invocation messages use the structured JSON logger instead of `[command] ...` text.

## 3. CLI Integration

- [ ] 3.1 Replace standard `log` usage in `cmd/orchv3` with `steplog`.
- [ ] 3.2 Log CLI startup as `module=cli`, `type=info` when no proposal task is provided.
- [ ] 3.3 Log CLI fatal errors as `module=cli`, `type=error` before exiting with a non-zero status.
- [ ] 3.4 Keep the successful PR URL on stdout for scripting while sending structured diagnostic logs to stderr.

## 4. Verification

- [ ] 4.1 Update existing proposal runner and CLI tests to assert JSON log output instead of bracketed text logs.
- [ ] 4.2 Add regression tests proving Codex output and command output are represented as JSON log events without being filtered out.
- [ ] 4.3 Run `go fmt ./...`.
- [ ] 4.4 Run `go test ./...`.
