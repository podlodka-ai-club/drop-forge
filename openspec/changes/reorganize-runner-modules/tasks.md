## 1. Baseline And Package Move

- [ ] 1.1 Run the current runner and orchestration test suite to capture baseline failures before refactor.
- [ ] 1.2 Create a dedicated internal runner directory and move proposal, apply, and archive runner packages under it without changing behavior.
- [ ] 1.3 Update imports in `cmd`, `coreorch`, tests, and any runner callers to use the new package paths.
- [ ] 1.4 Run `go test ./...` after the mechanical move and fix package/import breakage before extracting shared code.

## 2. Shared Runner Components

- [ ] 2.1 Extract shared agent execution input, result, and interface types used by proposal, apply, and archive runners.
- [ ] 2.2 Extract shared logged-command execution and writer fallback behavior, then remove duplicate `logged_command.go` implementations from stage packages.
- [ ] 2.3 Extract shared Codex CLI executor behavior with explicit stage profile fields for prompt builder, last-message file, error label, service/module, and final-message capture.
- [ ] 2.4 Extract shared metadata helpers for display name, title prefixing, slug generation, and commit message construction.
- [ ] 2.5 Replace apply/archive imports of proposal runner metadata helpers with the shared metadata package.

## 3. Stage Workflows

- [ ] 3.1 Keep proposal runner stage logic explicit for new branch, commit, push, PR creation, and optional final-response comment.
- [ ] 3.2 Consolidate apply/archive existing-branch workflow if the common helper remains simpler than the duplicated code after shared component extraction.
- [ ] 3.3 Preserve stage-specific prompts for `openspec-propose`, `openspec-apply-change`, and `openspec-archive-change`.
- [ ] 3.4 Preserve no-change errors, branch resolution precedence, commit prefixes, stdout/stderr logging, temp workspace cleanup behavior, and GitManager delegation.

## 4. Tests And Documentation

- [ ] 4.1 Update runner unit tests for new package paths and shared agent execution types.
- [ ] 4.2 Add or update table-driven tests for shared Codex executor command args, prompt stdin, logging, and final-message capture.
- [ ] 4.3 Add or update tests proving apply/archive do not create PRs and proposal still creates PR/comment through GitManager.
- [ ] 4.4 Update `architecture.md` with the new runner directory, shared runner components, and current actor-to-code mapping.
- [ ] 4.5 Run `go fmt ./...` and `go test ./...`.
