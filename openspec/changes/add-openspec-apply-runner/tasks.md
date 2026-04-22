## 1. Configuration

- [ ] 1.1 Add `ApplyRunnerConfig` to `internal/config` with repository URL, remote name, commit title prefix, cleanup flag, Git path, and Codex path.
- [ ] 1.2 Load `APPLY_*` keys from `.env` and process environment while preserving current environment-over-dotenv precedence.
- [ ] 1.3 Validate required apply configuration and return contextual errors for missing repository URL or empty required values.
- [ ] 1.4 Update `.env.example` with all apply runner keys, leaving values empty.
- [ ] 1.5 Add table-driven config tests for apply defaults, missing required repository URL, invalid cleanup flag, and environment precedence.

## 2. Shared Workflow Helpers

- [ ] 2.1 Identify repeated proposal runner logic that can be safely shared without changing proposal behavior.
- [ ] 2.2 Extract small helpers for Codex argv construction, writer fallback, git status capture, git add/commit, git push, and temp cleanup where reuse is practical.
- [ ] 2.3 Keep proposal-specific branch creation, PR creation, and open-questions comment logic in proposal runner.
- [ ] 2.4 Update existing proposal runner tests if helper extraction changes package boundaries, preserving current command order expectations.

## 3. Apply Runner Flow

- [ ] 3.1 Create `internal/applyrunner` with a `Runner` type and a public method that accepts a proposal branch name and returns the updated branch name.
- [ ] 3.2 Reject empty branch names, branch names containing whitespace, and branch names starting with `-` before filesystem or command side effects.
- [ ] 3.3 Create a unique temp directory per apply run and clone the configured repository with `git clone --branch <proposal-branch> --single-branch <repo-url> <clone-dir>`.
- [ ] 3.4 Build and log the Codex prompt containing the `openspec-apply` skill instruction, proposal branch name, and instruction to implement the OpenSpec change in the current branch.
- [ ] 3.5 Run Codex CLI in the cloned repository with `codex exec --sandbox danger-full-access --cd <clone-dir> -`, passing the prompt through stdin and streaming stdout/stderr.
- [ ] 3.6 Inspect `git status --short` after Codex and stop with a contextual error when no changes were produced.
- [ ] 3.7 Run `git add -A`, commit with the configured apply title prefix, and push with `git push <remote> HEAD:<proposal-branch>`.
- [ ] 3.8 Ensure the apply runner never runs `git checkout -b`, `gh pr create`, or any GitHub CLI command.
- [ ] 3.9 Preserve the temp directory by default, log its path, and support the apply cleanup flag.

## 4. CLI and Documentation

- [ ] 4.1 Extend `cmd/orchv3` parsing with explicit `apply <proposal-branch>` and `proposal <task description>` modes.
- [ ] 4.2 Preserve legacy proposal invocation through positional task description and stdin without a subcommand.
- [ ] 4.3 Route apply workflow logs to stderr and print only the updated branch name to stdout on success.
- [ ] 4.4 Document apply usage, runtime variables, prerequisites, and the "push to existing branch, no new PR" behavior.
- [ ] 4.5 Update existing proposal documentation if CLI examples change because of explicit subcommands.

## 5. Verification

- [ ] 5.1 Add apply runner unit tests for happy path, invalid branch input, missing config, clone failure, Codex failure, no changes, commit failure, push failure, default temp retention, and cleanup.
- [ ] 5.2 Add tests for exact clone args, exact Codex argv, stdin prompt contents, absence of `git checkout -b`, absence of `gh pr create`, and push target `HEAD:<proposal-branch>`.
- [ ] 5.3 Add CLI tests for `apply`, explicit `proposal`, and legacy proposal invocation.
- [ ] 5.4 Run `go fmt ./...`.
- [ ] 5.5 Run `go test ./...`.
