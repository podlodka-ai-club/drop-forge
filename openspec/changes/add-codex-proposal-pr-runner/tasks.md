## 1. Configuration

- [ ] 1.1 Extend `internal/config` with `.env` loading using the Go standard library and process environment override semantics.
- [ ] 1.2 Add required proposal runner configuration fields: repository URL, base branch, remote name, branch prefix, PR title prefix, temp retention flag, and paths for `git`, `codex`, and `gh`.
- [ ] 1.3 Validate required configuration and return contextual errors for missing repository or invalid values.
- [ ] 1.4 Add `env.dist` and update `.env.example` with all proposal runner keys, leaving values empty.
- [ ] 1.5 Add table-driven tests for `.env` parsing, process environment precedence, and missing required settings.

## 2. Command Execution and Logging

- [ ] 2.1 Introduce a command runner abstraction that can stream stdout and stderr to configured writers.
- [ ] 2.2 Implement the production command runner with `os/exec`, `context.Context`, working directory support, argv logging, and contextual errors.
- [ ] 2.3 Add step logging helpers or a small logger wrapper for `temp`, `git`, `codex`, and `github` prefixes.
- [ ] 2.4 Add unit tests with a fake command runner to verify command order, working directories, stream forwarding, and error propagation.

## 3. Proposal Runner Flow

- [ ] 3.1 Create `internal/proposalrunner` with a `Runner` type and a public method that accepts a task description string and returns a PR URL.
- [ ] 3.2 Reject empty or whitespace-only task descriptions before any filesystem or command side effects.
- [ ] 3.3 Create a unique temp directory per run and clone the configured repository into it with `git clone`.
- [ ] 3.4 Build and log the Codex prompt containing the `openspec-propose` skill instruction and original task description.
- [ ] 3.5 Run Codex CLI in the cloned repository and stream all Codex stdout/stderr to the console.
- [ ] 3.6 Inspect `git status --short` after Codex; stop with a contextual error when no changes were produced.
- [ ] 3.7 Create a branch, add changes, commit, push, create a PR through `gh pr create`, parse the PR URL, log it, and return it.
- [ ] 3.8 Support cleanup of the temp directory by default and an environment flag to retain it for debugging.

## 4. CLI Integration

- [ ] 4.1 Extend `cmd/orchv3` to accept a task description argument or stdin input for invoking the proposal runner.
- [ ] 4.2 Preserve the existing startup/config behavior where useful, but route proposal execution errors to clear fatal logs and non-zero exit.
- [ ] 4.3 Print the resulting PR URL to stdout in a form that can be consumed by scripts.

## 5. Verification

- [ ] 5.1 Add unit tests for the proposal runner happy path, Codex failure, git clone failure, missing changes, and PR creation failure.
- [ ] 5.2 Add tests for generated branch slug and Codex prompt construction.
- [ ] 5.3 Run `go fmt ./...`.
- [ ] 5.4 Run `go test ./...`.
- [ ] 5.5 Manually document any external prerequisites that tests do not cover, including `git`, `codex`, and authenticated `gh`.
