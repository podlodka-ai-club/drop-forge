## 1. Duplicate Audit

- [ ] 1.1 Inventory production and test Go files with focus on `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`, `internal/reviewrunner`, `internal/coreorch`, and shared helper packages.
- [ ] 1.2 Compare likely duplicate clusters across runner lifecycle code, Codex executor plumbing, command logging helpers, input validation, commit message builders, and test fakes/builders.
- [ ] 1.3 Create `docs/duplicate-code-audit.md` with audit date, scope, methodology, limitations, duplicate cluster table, recommendation, rationale, and current decision status for each cluster.

## 2. Safe Refactors

- [ ] 2.1 Evaluate the duplicated `runLoggedCommand` helper across stage runner packages and extract it to a natural shared package if the API remains simple and behavior-preserving.
- [ ] 2.2 Evaluate duplicated Codex executor plumbing (`codexArgs`, last-message handling, prompt execution flow) and either implement a low-risk extraction or document why stage-specific code should stay explicit.
- [ ] 2.3 Evaluate `applyrunner`/`archiverunner` runner lifecycle duplication and document whether it should remain as explicit stage symmetry or be reduced through a small helper.
- [ ] 2.4 Update or add focused tests for every refactor that moves behavior into a shared helper.

## 3. Documentation And Architecture

- [ ] 3.1 Ensure `docs/duplicate-code-audit.md` explains all deferred or intentionally accepted duplicates with a future revisit signal.
- [ ] 3.2 Update `architecture.md` only if implementation introduces a new shared package or changes component responsibility boundaries.
- [ ] 3.3 Confirm no new mandatory external duplicate-detection dependency was added to `go.mod`, CI, or required verification commands.

## 4. Verification

- [ ] 4.1 Run `go fmt ./...`.
- [ ] 4.2 Run `go test ./...`.
- [ ] 4.3 Run `openspec status --change audit-code-duplicates` and verify the change is apply-ready.
