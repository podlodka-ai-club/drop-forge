## 1. Prompt Contract

- [ ] 1.1 Locate the proposal orchestration code that builds `ProposalInput.AgentPrompt` from a managed Linear task.
- [ ] 1.2 Ensure tasks without description render an explicit "no description provided" marker in the prompt context.
- [ ] 1.3 Ensure tasks without comments render an explicit "no comments available" marker in the prompt context.
- [ ] 1.4 Ensure the prompt keeps the Linear identifier and title visible for traceability.

## 2. Tests

- [ ] 2.1 Add or update a unit test for a ready-to-propose task with identifier `DRO-49`, title `Просто тест`, empty description, and no comments.
- [ ] 2.2 Assert the prepared `ProposalInput` has the expected identifier, title, non-empty prompt, missing-description marker, and missing-comments marker.
- [ ] 2.3 Assert the low-context task uses the normal proposal route and does not require any special state, CLI mode, or runtime configuration.

## 3. Verification

- [ ] 3.1 Run `go fmt ./...`.
- [ ] 3.2 Run `go test ./...`.
- [ ] 3.3 Run `openspec status --change dro-49-simple-test` and confirm the change remains apply-ready.
