## 1. Project Overview Document

- [ ] 1.1 Create `docs/project-overview.md` with a concise explanation of the project purpose, the current proposal-runner scenario, and the question this repository solves today.
- [ ] 1.2 Document the current end-to-end workflow from task input through configuration loading, temporary clone creation, Codex execution, git push, and GitHub pull request creation.
- [ ] 1.3 Add a repository map and current scope boundaries covering `cmd/orchv3`, `internal/config`, `internal/proposalrunner`, supporting docs, and the fact that the project is currently a Go CLI rather than an HTTP service.

## 2. README Navigation

- [ ] 2.1 Update `README.md` so it links to `docs/project-overview.md` as the main introduction point for readers who want to understand the project before running it.
- [ ] 2.2 Review README wording and related doc links to keep `README.md`, `docs/project-overview.md`, and `docs/proposal-runner.md` complementary rather than duplicative.

## 3. Verification

- [ ] 3.1 Verify that the overview document matches the current behavior in `cmd/orchv3/main.go`, `internal/config/config.go`, `internal/proposalrunner`, `.env.example`, and `docs/proposal-runner.md`.
- [ ] 3.2 Run `go fmt ./...`.
- [ ] 3.3 Run `go test ./...`.
