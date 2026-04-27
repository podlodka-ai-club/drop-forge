## 1. Prompt Builders

- [ ] 1.1 Verify `coreorch.BuildApplyInput` uses the shared task prompt builder and includes Linear comments in `AgentPrompt`.
- [ ] 1.2 Verify `coreorch.BuildArchiveInput` uses the shared task prompt builder and includes Linear comments in `AgentPrompt`.
- [ ] 1.3 Ensure tasks without comments produce an explicit `No comments available.` comments section for Apply and Archive prompts.

## 2. Tests

- [ ] 2.1 Add or update Apply input builder tests for ready-to-code tasks with comments, including comment body, author fallback, and task identity fields.
- [ ] 2.2 Add or update Apply input builder tests for ready-to-code tasks without comments.
- [ ] 2.3 Add or update Archive input builder tests for ready-to-archive tasks with comments, including comment body, author fallback, and task identity fields.
- [ ] 2.4 Add or update Archive input builder tests for ready-to-archive tasks without comments.
- [ ] 2.5 Add or update Apply/Archive runner tests to assert the agent executor receives `input.AgentPrompt` unchanged.
- [ ] 2.6 Add or update TaskManager/Linear client tests if current coverage does not verify comments on ready-to-code and ready-to-archive payloads.

## 3. Documentation And Verification

- [ ] 3.1 Update `architecture.md` only if implementation changes component responsibilities or data flow beyond prompt formatting.
- [ ] 3.2 Run `go fmt ./...`.
- [ ] 3.3 Run `go test ./...`.
- [ ] 3.4 Run `openspec status --change pull-linear-comments-into-stage-prompts` and verify the change is apply-ready.
