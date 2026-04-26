## 1. Agent Executor Contract

- [x] 1.1 Add minimal `AgentExecutor` input/result types for proposal generation in `internal/proposalrunner`
- [x] 1.2 Update `Runner` construction so the default path wires a Codex CLI executor while tests can inject a fake executor
- [x] 1.3 Change `Runner.Run` to call `AgentExecutor` after clone and before git status, without building Codex prompt or args directly

## 2. Codex CLI Implementation

- [x] 2.1 Move Codex prompt, argv, last-message file, and final-message reading into a Codex executor implementation
- [x] 2.2 Preserve the current Codex command format including `exec`, `--json`, `--sandbox danger-full-access`, `--output-last-message`, `--cd <clone-dir>`, and stdin prompt
- [x] 2.3 Preserve Codex output forwarding and diagnostics through structured log events
- [x] 2.4 Keep existing `PROPOSAL_CODEX_PATH` behavior unless implementation reveals a clear need for a documented migration

## 3. Tests

- [x] 3.1 Update proposal runner happy-path tests to assert orchestration around a fake agent executor instead of Codex argv details
- [x] 3.2 Add Codex executor tests for prompt content, command args, working directory, last-message capture, stdout/stderr forwarding, and failure context
- [x] 3.3 Update failure-path tests so agent executor errors are reported as agent proposal failures and do not continue to git status or PR creation
- [x] 3.4 Verify empty final agent messages skip PR comments while non-empty messages are published

## 4. Documentation And Architecture

- [x] 4.1 Update `README.md` and `docs/proposal-runner.md` to describe the agent executor boundary and current Codex CLI prerequisite
- [x] 4.2 Update `.env.example` only if supported configuration keys change
- [x] 4.3 Update `architecture.md` mapping so `AgentExecutor` is no longer only implicit inside `proposalrunner`

## 5. Verification

- [x] 5.1 Run `go fmt ./...`
- [x] 5.2 Run `go test ./...`
- [x] 5.3 Review the OpenSpec delta for consistency with the implementation before archiving
