## 1. Context Validation

- [ ] 1.1 Inspect current proposal orchestration input builder and task processing order.
- [ ] 1.2 Add a small helper that treats trimmed description or at least one trimmed comment as sufficient proposal context.
- [ ] 1.3 Add table-driven tests for description-only, comment-only, whitespace-only, title-only, and fully empty task payloads.

## 2. Proposal Orchestration Flow

- [ ] 2.1 Run the context preflight before moving ready-to-propose tasks to proposing-in-progress.
- [ ] 2.2 For insufficient context, skip proposal runner execution, PR attachment, and proposal-review transition.
- [ ] 2.3 Publish deterministic Linear feedback asking for goal, expected behavior, and acceptance criteria.
- [ ] 2.4 Avoid duplicate feedback comments when the same feedback already exists on the task.
- [ ] 2.5 Emit a structured skip log with task identity and insufficient-context reason.
- [ ] 2.6 Return a contextual error when feedback comment publication fails.

## 3. Tests

- [ ] 3.1 Add orchestration tests proving title-only tasks are skipped before state transition and runner execution.
- [ ] 3.2 Add orchestration tests proving tasks with description or comments still enter the normal proposal flow.
- [ ] 3.3 Add orchestration tests for feedback comment publication, duplicate suppression, and feedback publication failure.
- [ ] 3.4 Keep tests isolated from Linear API, GitHub CLI, Codex CLI, git, and network access through existing fake dependencies.

## 4. Documentation and Verification

- [ ] 4.1 Update `architecture.md` only if implementation changes component responsibilities beyond the local preflight branch.
- [ ] 4.2 Run `go fmt ./...`.
- [ ] 4.3 Run `go test ./...`.
- [ ] 4.4 Run `openspec status --change dro-43-handle-empty-linear-task-proposals` and verify the change is apply-ready.
