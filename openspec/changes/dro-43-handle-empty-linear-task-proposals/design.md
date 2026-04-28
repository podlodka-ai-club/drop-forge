## Context

Proposal orchestration currently treats every task in `LINEAR_STATE_READY_TO_PROPOSE_ID` as executable. The proposal input builder can produce a non-empty prompt from identifier/title even when the task has no real requirements, as in `DRO-43` (`ТЕст 3`, no description, no comments).

This keeps the runner mechanically healthy but moves ambiguity into Codex execution, where the generated OpenSpec proposal can become arbitrary. The better boundary is before runner execution: orchestration already owns task selection, task state changes, comments, structured logs, and fakeable dependencies.

## Goals / Non-Goals

**Goals:**

- Stop proposal execution for ready-to-propose tasks that do not contain enough human-authored context.
- Leave actionable feedback on the Linear task using the existing comment capability.
- Preserve current behavior for tasks that have a meaningful description or meaningful comments.
- Keep the implementation unit-testable without Linear, GitHub, git, Codex CLI, or network access.

**Non-Goals:**

- Do not infer product requirements from short placeholder titles.
- Do not introduce NLP, LLM classification, or a new external validation service.
- Do not add a new Linear workflow state unless a future product decision requires it.
- Do not change Apply or Archive routing.

## Decisions

### D1. Validate proposal context inside orchestration

The preflight check should live in the proposal route before moving a task to proposing-in-progress. It can inspect the existing `Task` fields and decide whether to call the runner.

Alternative: validate inside `proposalrunner`. Rejected because the runner should stay focused on git/Codex/PR execution; it should not need Linear comment or state semantics.

### D2. Use description/comments as the minimum meaningful context

A task is proposal-ready when at least one of these fields contains meaningful non-whitespace text:

- description
- any comment body

Title and identifier remain useful for traceability, but they are not enough to define behavior by themselves. This directly handles tasks like `DRO-43`, where only a placeholder title exists.

Alternative: require both description and comments. Rejected because many valid new tasks have a complete description and no discussion yet.

Alternative: use a minimum character count. Rejected for the first implementation because it can reject concise but valid requirements and requires subjective thresholds.

### D3. Comment and skip without changing task state

When context is insufficient, orchestration should publish a Linear comment explaining what is missing, emit a structured log event, skip the runner, and keep the task out of proposal review. The task can remain in `Ready to Propose` so humans can update it and re-run the normal workflow.

Alternative: move the task to a separate blocked/rejected state. Rejected because no such configured state is part of the current runtime contract, and adding one would require new config and workflow rollout.

### D4. Treat comment publish failure as a contextual orchestration error

If Linear rejects the feedback comment, orchestration should return an error identifying the task and operation. It should still avoid running the proposal runner, because the task remains under-specified.

Alternative: silently skip when feedback cannot be posted. Rejected because operators would lose the reason the task did not progress.

## Risks / Trade-offs

- [False negative for title-only valid tasks] -> Mitigation: require humans to place actual requirements in description/comments; title-only tasks are not reliable enough for autonomous proposal generation.
- [Repeated comments on every polling pass] -> Mitigation: implementation can make the comment text deterministic and avoid reposting if the same feedback comment is already present in returned task comments.
- [Ready-to-propose queue can retain blocked tasks] -> Mitigation: structured logs and Linear feedback make the blocking reason visible without adding new workflow states.
- [Validation rules may need tightening later] -> Mitigation: keep the check small and local so thresholds or additional fields can be added without changing runner contracts.

## Migration Plan

1. Add a small helper for proposal-context sufficiency, with table-driven unit tests.
2. Insert the preflight before the proposing-in-progress transition.
3. Publish deterministic Linear feedback for insufficient context and skip runner execution.
4. Add orchestration tests for skip, comment failure, and no-runner behavior.
5. Run `go fmt ./...` and `go test ./...`.

Rollback: remove the preflight branch and tests; the existing proposal flow remains otherwise unchanged.

## Open Questions

- Should repeated polling suppress duplicate feedback comments by exact text matching?
  - Recommended option: yes, because the monitor is continuous and duplicate comments would make Linear tasks noisy.
