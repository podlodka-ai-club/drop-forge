## 1. CLI Contract

- [ ] 1.1 Add tests for `orchv3 apply <proposal-branch>` routing to apply workflow.
- [ ] 1.2 Add tests for `orchv3 proposal <task description>` routing to the existing proposal workflow.
- [ ] 1.3 Preserve and test legacy `orchv3 <task description>` proposal invocation.
- [ ] 1.4 Implement CLI parsing changes with clear errors for missing apply branch or missing proposal description.

## 2. Shared Workflow Refactor

- [ ] 2.1 Identify duplicated proposal/apply steps: temp lifecycle, command runner defaults, clone preparation, Codex invocation, git status, commit/push, PR creation, and logging.
- [ ] 2.2 Extract only the shared helpers needed by both workflows while keeping proposal-specific prompt, branch, title, body, and PR base logic separate.
- [ ] 2.3 Update existing proposal runner tests to prove command order and public behavior did not change.

## 3. Apply Runner Implementation

- [ ] 3.1 Add apply runner config with `APPLY_BRANCH_PREFIX` and `APPLY_PR_TITLE_PREFIX`, reusing repository URL, remote, cleanup, and command path settings.
- [ ] 3.2 Add input validation that rejects empty proposal branch names before temp creation or external commands.
- [ ] 3.3 Prepare temp clone from the provided proposal branch before Codex runs, returning branch-specific git errors with context.
- [ ] 3.4 Build and log a Codex prompt that instructs Codex to use the `openspec-apply` skill and includes the proposal branch name.
- [ ] 3.5 Run Codex through `codex exec --sandbox danger-full-access --cd <clone-dir> -` with the prompt on stdin.
- [ ] 3.6 After Codex succeeds, run `git status --short` and return a no-changes error before branch/commit/PR steps when no changes exist.
- [ ] 3.7 Create the implementation branch only after Codex produced changes, then add, commit, push, and create a PR with base equal to the proposal branch.
- [ ] 3.8 Return and log the apply PR URL.

## 4. Tests

- [ ] 4.1 Add apply happy-path unit test with fake command runner asserting git, Codex, push, and `gh pr create --base <proposal-branch>` command order.
- [ ] 4.2 Add tests for empty proposal branch rejection and config validation before side effects.
- [ ] 4.3 Add tests for proposal branch preparation failure, Codex apply failure, no-changes failure, and PR creation failure.
- [ ] 4.4 Add tests for apply temp cleanup/retention behavior.
- [ ] 4.5 Add helper tests for apply prompt, implementation branch name, PR title, and PR body builders.

## 5. Documentation and Configuration

- [ ] 5.1 Update `.env.example` with apply-specific keys without default values.
- [ ] 5.2 Update runner documentation with `orchv3 apply <proposal-branch>`, stacked PR behavior, and apply prerequisites.
- [ ] 5.3 Document that apply PR base is the proposal branch and that legacy proposal invocation remains supported.

## 6. Verification

- [ ] 6.1 Run `go fmt ./...`.
- [ ] 6.2 Run `go test ./...`.
- [ ] 6.3 Review `openspec/changes/add-openspec-apply-runner` artifacts for consistency before implementation.
