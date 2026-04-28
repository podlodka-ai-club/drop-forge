## 1. Configuration Contract

- [x] 1.1 Update config structs and loader to use `DROP_FORGE_REPOSITORY_URL`, `DROP_FORGE_BASE_BRANCH`, `DROP_FORGE_REMOTE_NAME`, `DROP_FORGE_CLEANUP_TEMP`, `DROP_FORGE_POLL_INTERVAL`, `DROP_FORGE_GIT_PATH`, `DROP_FORGE_CODEX_PATH`, and `DROP_FORGE_GH_PATH` for shared runtime settings.
- [x] 1.2 Keep `PROPOSAL_BRANCH_PREFIX` and `PROPOSAL_PR_TITLE_PREFIX` as proposal-specific metadata settings.
- [x] 1.3 Change default application identity to `Drop Forge` when `APP_NAME` is absent.
- [x] 1.4 Add validation errors for removed shared `PROPOSAL_*` keys that name the corresponding supported `DROP_FORGE_*` key.
- [x] 1.5 Update config tests for environment precedence, missing required keys, invalid poll interval, default app name, and legacy key rejection.

## 2. Runtime Wiring

- [x] 2.1 Update CLI wiring to pass shared Drop Forge config fields into CoreOrch, GitManager, proposal runner, Apply runner, and Archive runner.
- [x] 2.2 Update monitor naming, startup logs, cancellation logs, and unsupported manual input errors to refer to Drop Forge orchestration instead of proposal-only monitor wording.
- [x] 2.3 Update proposal PR body and generated metadata text to identify Drop Forge while preserving proposal-specific PR title and branch derivation.
- [x] 2.4 Update runner and GitManager tests that assert command paths, temp cleanup, repository URL, PR body, service name, or log messages.

## 3. Environment Template And Docs

- [x] 3.1 Update `.env.example` to list supported `DROP_FORGE_*`, `APP_*`, `PROPOSAL_BRANCH_PREFIX`, `PROPOSAL_PR_TITLE_PREFIX`, `LINEAR_*`, and logging keys without values.
- [x] 3.2 Remove replaced shared `PROPOSAL_*` keys from `.env.example` and documentation.
- [x] 3.3 Update README and proposal runner docs to describe Drop Forge, shared configuration keys, stage-specific keys, and local migration from old proposal-only keys.
- [x] 3.4 Update ELK/logging docs where service name examples still use `orchv3`.
- [x] 3.5 Update `architecture.md` to describe Drop Forge runtime identity and shared configuration ownership.

## 4. Spec And Test Verification

- [x] 4.1 Run OpenSpec validation/status for `rename-to-drop-forge-and-migrate-config` and fix any proposal/spec/task issues.
- [x] 4.2 Run `go fmt ./...`.
- [x] 4.3 Run `go test ./...`.
- [x] 4.4 Document any remaining intentional non-renames, especially `cmd/orchv3` or Go module path, in the final implementation report.
