## 1. Runner Refactoring

- [ ] 1.1 Extract the shared proposal workflow steps in `internal/proposalrunner` into a private reusable workflow path without changing `Runner.Run(ctx, taskDescription)` behavior.
- [ ] 1.2 Add workflow options/builders for Codex prompt, PR branch name, PR title, PR body, and optional pre-Codex git commands.
- [ ] 1.3 Keep existing proposal tests passing and add regression coverage proving the current proposal command order and prompt are unchanged.

## 2. Apply Workflow

- [ ] 2.1 Add `RunApply(ctx, proposalBranch string)` or an equivalent apply entrypoint that validates non-empty branch input before side effects.
- [ ] 2.2 Implement apply preparation so the clone checks out the caller-provided proposal branch before Codex runs.
- [ ] 2.3 Add `BuildApplyCodexPrompt` and reuse the existing Codex CLI argv format while instructing Codex to use `openspec-apply`.
- [ ] 2.4 Ensure no implementation branch is created before the Codex apply step.
- [ ] 2.5 After a successful apply with changes, create the implementation branch, commit, push, and create a PR with base set to the proposal branch.
- [ ] 2.6 Return contextual errors for proposal branch checkout failure, Codex apply failure, and no-change apply results.

## 3. Configuration And CLI

- [ ] 3.1 Add apply-specific branch/title prefix config with defaults in code and keys in `.env.example` without values.
- [ ] 3.2 Update config validation/tests so shared command paths remain required and apply-specific config is validated.
- [ ] 3.3 Add explicit CLI apply mode, for example `orchv3 apply <proposal-branch>`, while preserving existing proposal invocation through args/stdin.
- [ ] 3.4 Add CLI tests for legacy proposal invocation, apply invocation, and missing apply branch input.

## 4. Documentation And Verification

- [ ] 4.1 Update runner documentation with the apply command, branch behavior, PR base behavior, and required prerequisites.
- [ ] 4.2 Add unit tests for apply command order, prompt content, branch checkout failure, no-change behavior, and PR base branch selection.
- [ ] 4.3 Run `go fmt ./...`.
- [ ] 4.4 Run `go test ./...`.
