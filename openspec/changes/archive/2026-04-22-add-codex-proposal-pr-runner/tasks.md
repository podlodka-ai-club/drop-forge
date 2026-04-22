## 1. Configuration

- [x] 1.1 Add `github.com/joho/godotenv` and extend `internal/config` with `.env` loading through godotenv while preserving process environment override semantics.
- [x] 1.2 Add required proposal runner configuration fields: repository URL, base branch, remote name, branch prefix, PR title prefix, temp cleanup flag that is disabled by default, and paths for `git`, `codex`, and `gh`.
- [x] 1.3 Validate required configuration and return contextual errors for missing repository or invalid values.
- [x] 1.4 Update `.env.example` with all proposal runner keys, leaving values empty.
- [x] 1.5 Add table-driven tests for godotenv loading, `.env` parsing behavior, process environment precedence, and missing required settings.

## 2. Command Execution and Logging

- [x] 2.1 Introduce a command runner abstraction that can stream stdout and stderr to configured writers.
- [x] 2.2 Implement the production command runner with `os/exec`, `context.Context`, working directory support, argv logging, and contextual errors.
- [x] 2.3 Add step logging helpers or a small logger wrapper for `temp`, `git`, `codex`, and `github` prefixes.
- [x] 2.4 Add unit tests with a fake command runner to verify command order, working directories, stream forwarding, and error propagation.

## 3. Proposal Runner Flow

- [x] 3.1 Create `internal/proposalrunner` with a `Runner` type and a public method that accepts a task description string and returns a PR URL.
- [x] 3.2 Reject empty or whitespace-only task descriptions before any filesystem or command side effects.
- [x] 3.3 Create a unique temp directory per run and clone the configured repository into it with `git clone`.
- [x] 3.4 Build and log the Codex prompt containing the `openspec-propose` skill instruction and original task description.
- [x] 3.5 Run Codex CLI in the cloned repository and stream all Codex stdout/stderr to the console.
- [x] 3.6 Inspect `git status --short` after Codex; stop with a contextual error when no changes were produced.
- [x] 3.7 Create a branch, add changes, commit, push, create a PR through `gh pr create`, parse the PR URL, log it, and return it.
- [x] 3.8 Add a separate PR comment with open implementation questions when questions are produced during the run.
- [x] 3.9 Use the current local Codex non-interactive format: `codex exec --sandbox danger-full-access --cd <clone-dir> -` with prompt passed through stdin; do not add an ENV-based argv template in v1.
- [x] 3.10 Preserve the temp directory by default, log its path, and support an environment flag to enable cleanup.

## 4. CLI Integration

- [x] 4.1 Extend `cmd/orchv3` to accept a task description argument or stdin input for invoking the proposal runner.
- [x] 4.2 Preserve the existing startup/config behavior where useful, but route proposal execution errors to clear fatal logs and non-zero exit.
- [x] 4.3 Print the resulting PR URL to stdout in a form that can be consumed by scripts.

## 5. Verification

- [x] 5.1 Add unit tests for the proposal runner happy path, Codex failure, git clone failure, missing changes, PR creation failure, and open-questions comment failure.
- [x] 5.2 Add tests for generated branch slug, Codex prompt construction, exact Codex argv, stdin prompt passing, and default temp directory retention.
- [x] 5.3 Run `go fmt ./...`.
- [x] 5.4 Run `go test ./...`.
- [x] 5.5 Manually document any external prerequisites that tests do not cover, including `git`, `codex`, authenticated `gh`, and the `godotenv` configuration behavior.
