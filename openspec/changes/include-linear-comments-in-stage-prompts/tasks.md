## 1. Orchestration Prompt Tests

- [ ] 1.1 Add or update `coreorch` unit tests for `BuildApplyInput` so a ready-to-code task with Linear comments produces an `AgentPrompt` containing `Comments:`, comment body, author, and timestamp.
- [ ] 1.2 Add or update `coreorch` unit tests for `BuildArchiveInput` so a ready-to-archive task with Linear comments produces an `AgentPrompt` containing `Comments:`, comment body, author, and timestamp.
- [ ] 1.3 Add or update tests for ready-to-code and ready-to-archive tasks without comments so both prompts contain `No comments available.` while still validating branch source handling.

## 2. Implementation

- [ ] 2.1 Reuse or adjust the shared task prompt builder so `ApplyInput.AgentPrompt` and `ArchiveInput.AgentPrompt` include the Linear comments block without diverging from proposal prompt formatting.
- [ ] 2.2 Ensure empty comment body, missing author, and zero timestamp fallbacks remain stable for apply and archive prompts.
- [ ] 2.3 Confirm `applyrunner` and `archiverunner` Codex prompt construction preserves the full `AgentPrompt` received from orchestration.

## 3. Verification

- [ ] 3.1 Run `go fmt ./...`.
- [ ] 3.2 Run `go test ./...`.
- [ ] 3.3 Review the OpenSpec delta for `proposal-orchestration` and ensure all new scenarios are covered by implementation or tests.
