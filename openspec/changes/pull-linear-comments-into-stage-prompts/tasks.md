## 1. Verify Current Prompt Assembly

- [ ] 1.1 Review `internal/coreorch` Apply and Archive input builders to confirm where `AgentPrompt` is assembled.
- [ ] 1.2 Confirm `TaskManager` already returns comments for managed tasks and no new Linear query/configuration is needed.

## 2. Core Implementation

- [ ] 2.1 Ensure `BuildApplyInput` uses a task prompt that includes task ID, identifier, title, description, and Linear comments.
- [ ] 2.2 Ensure `BuildArchiveInput` uses a task prompt that includes task ID, identifier, title, description, and Linear comments.
- [ ] 2.3 Preserve explicit prompt markers for missing description, missing comments, missing author, and empty comment bodies.
- [ ] 2.4 Keep Apply and Archive runners unchanged unless tests show their prompt handoff drops the prepared context.

## 3. Tests

- [ ] 3.1 Add or update `coreorch` tests for Apply input with comments, including author/time metadata.
- [ ] 3.2 Add or update `coreorch` tests for Archive input with comments, including author/time metadata.
- [ ] 3.3 Add or update fallback tests for Apply and Archive prompts without comments and without descriptions.
- [ ] 3.4 Add or update tests proving empty comment bodies are represented instead of dropped.

## 4. Validation

- [ ] 4.1 Run `go fmt ./...`.
- [ ] 4.2 Run `go test ./...`.
- [ ] 4.3 Run `openspec status --change pull-linear-comments-into-stage-prompts`.
- [ ] 4.4 Confirm all OpenSpec artifacts exist and the change is ready for implementation.
