## 1. Identity and Naming

- [ ] 1.1 Add centralized application identity constants/defaults for `Catmarine`.
- [ ] 1.2 Replace user-facing `orchv3` naming in CLI text, service labels, temp workspace prefixes, README/docs references, and tests where it represents the active product name.
- [ ] 1.3 Rename or add the primary CLI entrypoint as `cmd/catmarine`, preserving `cmd/orchv3` only as a compatibility wrapper if needed.
- [ ] 1.4 Decide whether to rename the Go module path from `orchv3` to `catmarine`; if yes, update imports mechanically and verify no stale imports remain.

## 2. Configuration Migration

- [ ] 2.1 Add alias-aware config helper functions for strings, booleans, integers, and durations where migrated keys need fallback behavior.
- [ ] 2.2 Migrate shared runner/orchestration settings from `PROPOSAL_*` to primary `CATMARINE_*` keys with `PROPOSAL_*` fallback.
- [ ] 2.3 Ensure `CATMARINE_*` keys take precedence when both new and legacy keys are set.
- [ ] 2.4 Keep validation errors contextual for missing or invalid migrated settings, including legacy-only invalid values.
- [ ] 2.5 Update `.env.example` to list primary `CATMARINE_*` keys without defaults or secrets.

## 3. Documentation

- [ ] 3.1 Update `README.md` title, introduction, run commands, key directories, and configuration section for `Catmarine`.
- [ ] 3.2 Document the full migration mapping from `PROPOSAL_*` to `CATMARINE_*` and the precedence rule.
- [ ] 3.3 Update `docs/*` references that describe active CLI commands, runtime config, or product identity.
- [ ] 3.4 Update `architecture.md` if implementation changes CLI entrypoint mapping, config ownership, or module boundaries.

## 4. Tests

- [ ] 4.1 Add table-driven config tests for loading from `CATMARINE_*`.
- [ ] 4.2 Add config tests for fallback from legacy `PROPOSAL_*`.
- [ ] 4.3 Add config tests for conflict precedence where `CATMARINE_*` wins over `PROPOSAL_*`.
- [ ] 4.4 Update existing CLI, runner, and docs-related tests that assert old names, paths, temp prefixes, or env keys.

## 5. Verification

- [ ] 5.1 Run `openspec validate rename-and-migrate-config --strict`.
- [ ] 5.2 Run `go fmt ./...`.
- [ ] 5.3 Run `go test ./...`.
