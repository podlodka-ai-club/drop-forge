## 1. CLI routing and config flow

- [ ] 1.1 Add explicit detection of project-overview requests in the CLI input path for args and `stdin`.
- [ ] 1.2 Refactor config loading so project-overview mode does not require valid proposal-runner settings such as `PROPOSAL_REPOSITORY_URL`.
- [ ] 1.3 Update `run(...)` to route overview requests to a local response path and keep existing proposal workflow behavior for change requests.

## 2. Local project overview builder

- [ ] 2.1 Implement a small local module/helper that builds a concise project overview from `README.md` as the primary source.
- [ ] 2.2 Add fallback sourcing from `docs/proposal-runner.md`, `.env.example`, or verified CLI/config artifacts when `README.md` is missing or insufficient.
- [ ] 2.3 Format the overview output for `stdout` so it consistently summarizes project purpose, current workflow, configuration expectations, and pointers to deeper docs.

## 3. Tests and verification

- [ ] 3.1 Extend CLI tests with table-driven coverage for overview-request detection from args and `stdin`.
- [ ] 3.2 Add tests proving overview mode returns a non-empty local summary without invoking proposal workflow or requiring proposal config.
- [ ] 3.3 Add tests for fallback overview generation when the primary documentation source is unavailable.
- [ ] 3.4 Run `go fmt ./...`.
- [ ] 3.5 Run `go test ./...`.
