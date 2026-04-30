## Context

Current orchestration runtime consists of a monitor-loop in `CoreOrch` that loads managed Linear tasks through `TaskManager` and routes them by state ID into one of three stage runners: `ProposalRunner`, `ApplyRunner`, `ArchiveRunner`. Each runner follows the same pattern: temp clone → checkout/branch → run `AgentExecutor` → commit → push → move task to its human review state. The only existing `AgentExecutor` implementation is `CodexCLIExecutor`, encapsulating a `codex exec --json --output-last-message ...` invocation.

Producer artefacts (`proposal.md`/`design.md`/`specs`, implementation code, archive transitions) reach human reviewers without any automated quality control. The team is concurrently integrating Claude as a second coding agent, but it is not yet wired into code. The Review stage is therefore designed so it:

- works today with a single Codex backend across two slot identities (different models, different prompts, different trailer markers), and
- accepts a future second `AgentExecutor` (Claude) by registering it under a slot name, without any change to `ReviewRunner` or to the PR comment format.

## Goals / Non-Goals

**Goals:**

- Insert automatic, reviewer-driven quality control between "producer-runner pushed" and "human gets a review request".
- Reuse the existing architectural pattern: stage-runner + Linear state as a queue + isolated temp clone + structured logging + fakeable dependencies in tests.
- Make the producer marker machine-readable and self-contained on the artefact (no Linear-side metadata or comments).
- Publish PR review in CodeRabbit-style: summary + inline + ready-to-paste fix-prompt per finding.
- Keep the human as the sole decider of next transition; severity is informational and never auto-blocks.
- Make the AI review stage a feature flag controlled by environment: empty AI-review state IDs disable the fourth route entirely and producer-runners fall back to direct human review transitions.

**Non-Goals:**

- Do not integrate Claude as a real second executor in this change — only prepare the contract.
- Do not auto-apply fix-prompts; the human copy-pastes them.
- Do not introduce reviewer-side discussion (comment-on-comment), webhook responses, or model-quality statistics.
- Do not duplicate review output into Linear comments — comments live only on the PR.
- Do not parallelise review execution within a monitor pass — keep sequential like the three existing stages.
- Do not extract a dedicated `GitManager` package in this change; review continues to call git/gh through `commandrunner` directly, like the other runners.
- Do not introduce a manual CLI mode to run review for a specific task — the stage activates only through the monitor pass.

## Decisions

### D1. Review as the fourth orchestration stage with three AI-review state IDs

Extend `CoreOrch.RunProposalsOnce` route table with three entries:

```
NeedProposalAIReviewStateID  -> ReviewRunner(stage=Proposal)
NeedCodeAIReviewStateID      -> ReviewRunner(stage=Apply)
NeedArchiveAIReviewStateID   -> ReviewRunner(stage=Archive)
```

Producer-runners move tasks to `Need * AI Review` instead of `Need * Review`. `ReviewRunner` picks tasks up from the AI-review state, finishes its work, and moves them to `Need * Review` (the existing human review state). On failure, the task stays in the AI-review state and the next monitor pass picks it up.

`ReviewRunner` accepts `Stage` as an explicit parameter — it drives prompt template selection, allowed finding categories, and target collection. The runner itself is stage-agnostic; stage-specific knowledge lives in a `StageProfile` strategy object.

**Alternative:** a single `Need AI Review` state with the source stage carried in task comments. Rejected — it breaks the existing pattern where each stage owns its Linear state.

**Alternative:** inline review invocation inside each producer-runner after push. Rejected — couples review to specific runners, complicates restarting review without restarting the producer, smears `gh` and JSON parsing logic across three packages.

### D2. Producer marker as a git trailer in the commit message

Each producer-runner calls a single helper before `git commit`:

```go
agentmeta.AppendTrailer(message, agentmeta.Producer{
    By:    "codex",
    Model: "gpt-5-codex",
    Stage: agentmeta.StageProposal,
})
```

This appends a canonical git trailer block:

```
Produced-By: codex
Produced-Model: gpt-5-codex
Produced-Stage: proposal
```

`ReviewRunner` reads trailers via `git log -1 --format=%B` and parses keys case-insensitively. The format is part of the new `review-orchestration` spec contract.

**Alternative:** record producer in a Linear comment such as `[produced by codex]`. Rejected — couples review to Linear API and risks mixing with human comments.

**Alternative:** store producer in Linear custom-fields. Rejected — Linear API is used minimally in this project; we prefer not to expand surface area.

### D3. Reviewer slot is the opposite of the most recent producer

Configuration introduces two slots:

```env
REVIEW_ROLE_PRIMARY=codex
REVIEW_ROLE_SECONDARY=codex          # today also codex; future => claude
REVIEW_PRIMARY_MODEL=gpt-5-codex
REVIEW_SECONDARY_MODEL=gpt-5
REVIEW_PRIMARY_EXECUTOR_PATH=...
REVIEW_SECONDARY_EXECUTOR_PATH=...
```

`ReviewRunner` reads the trailer of the HEAD commit only and selects:

```
producer = trailer.Produced-By                       // "codex"
reviewer = config.OppositeOf(producer)               // primary <-> secondary
executor = registeredExecutors[reviewer.Slot]
```

The day Claude integrates becomes a single env edit (`REVIEW_ROLE_SECONDARY=claude`) plus registering `ClaudeCLIExecutor` in the executor factory.

**Edge cases:**

- Trailer absent → log warning, default producer to `unknown`, reviewer to `REVIEW_ROLE_SECONDARY`. Review still runs and the published summary marks "producer unknown".
- Trailer references a slot not in config → contextual configuration error, task remains in the AI-review state, log explains the missing `REVIEW_ROLE_*` setting.
- Multiple producer commits in history → only the trailer of the latest HEAD commit drives selection.

**Alternative:** strict alternation by run number. Rejected — manual pushes and human-driven fixes would desync alternation from reality.

**Alternative:** fixed reviewer (always secondary reviews primary). Rejected — does not satisfy "and vice versa" from the original brief.

### D4. Strict JSON contract for the reviewer response

The reviewer prompt mandates a strict JSON output conforming to:

```json
{
  "summary": {
    "verdict": "ship-ready | needs-work | blocked",
    "walkthrough": "markdown",
    "stats": { "findings": <int>, "by_severity": { "blocker": <int>, "major": <int>, "minor": <int>, "nit": <int> } }
  },
  "findings": [
    {
      "id": "F1",
      "category": "<closed enum, stage-specific>",
      "severity": "blocker | major | minor | nit",
      "file": "...",
      "line_start": <int> | null,
      "line_end": <int> | null,
      "title": "...",
      "message": "...",
      "fix_prompt": "..."
    }
  ]
}
```

The parser lives in `internal/reviewrunner/reviewparse`. On invalid JSON or schema mismatch, the runner attempts exactly one repair retry with a "your previous response was invalid, return strict JSON" prompt. A second failure returns a contextual error and leaves the task in the AI-review state without a partial publication.

`line_start` and `line_end` may be null for general findings; those go into the summary only, not into inline comments.

**Alternative:** markdown with regex anchors. Rejected — fragile, breaks on any prompt edit.

**Alternative:** two-pass freeform-then-normalise. Rejected — doubles cost and adds a second failure point for marginal benefit.

### D5. Closed stage-specific categories and informational severity

Each stage has its own closed category enum baked into both the prompt and the parser. Severity is a closed enum `blocker | major | minor | nit`. Severity controls only:

- summary `verdict` value (`blocked` if any blocker, `needs-work` if any major, otherwise `ship-ready`),
- ordering of findings in the summary,
- emoji icon prefix on each inline comment (`🛑 / ⚠️ / 💡 / 🪶`).

Severity does NOT change the task's transition. Auto-blocking is intentionally rejected: the human owns the merge decision.

### D6. Single atomic POST through GitHub Pull Request Reviews API

`gh pr comment` cannot create line-anchored review comments. The runner uses `gh api`:

```
POST /repos/{owner}/{repo}/pulls/{number}/reviews
{
  "commit_id": "<HEAD-sha>",
  "event": "COMMENT",
  "body": "<summary markdown>",
  "comments": [
    { "path": "...", "line": 18, "side": "RIGHT", "body": "<inline markdown>" }
  ]
}
```

`event: COMMENT` is intentional — review is informational, not approval/changes-requested.

Publication is encapsulated in `internal/reviewrunner/prcommenter` behind a `PRCommenter` interface. The concrete `GHPostReviewCommenter` uses `commandrunner` for `gh api -X POST ... --input -` with the JSON payload on stdin. `ReviewRunner` knows nothing about `gh` flags; `CoreOrch` knows nothing about `gh` at all.

**Alternative:** multiple separate POSTs (issue-comment summary plus per-finding review comments). Rejected — partial failure splits comments from summary.

### D7. Idempotency by (reviewer, stage, HEAD-sha)

Before posting, `ReviewRunner` calls `GET /pulls/{n}/reviews` and looks for an HTML-comment marker on the first line of any existing review body:

```
<!-- drop-forge-review-marker:<reviewer-slot>:<stage>:<HEAD-sha> -->
```

If a review with the same `(reviewer-slot, stage, HEAD-sha)` already exists, the runner skips publication and proceeds to move the task to the human review state. Force-push or new producer commit changes the HEAD sha and triggers a fresh review with a new marker. Existing reviews are never edited or deleted.

### D8. Targets and context budget

Per-stage targets:

- **Proposal:** all files under the new OpenSpec change directory (`openspec/changes/<change>/{proposal.md, design.md, tasks.md}` and everything under `specs/`). No diff — the change is new.
- **Apply:** `git diff <merge-base>..HEAD`, plus full text of touched `*.go` files, plus the corresponding OpenSpec change as "what should have been done".
- **Archive:** `git diff <merge-base>..HEAD` for the archive change, plus full text of archived spec files.

If the total exceeds `REVIEW_MAX_CONTEXT_BYTES`, the runner truncates by priority (auto-generated/lock files first, then test stub data) and records a "context truncated" tripwire in the summary.

### D9. Feature flag through environment

If any of the three AI-review state IDs is empty, `ReviewRunner` is not registered in `CoreOrch`, and producer-runners fall back to transitioning to the human review state directly. Configuration validation requires either all three AI-review state IDs and both reviewer slots populated, or all three AI-review state IDs empty. Partial configuration is a startup error.

## Risks / Trade-offs

- [Reviewer regularly returns invalid JSON] → Strict schema in prompt, one repair retry, explicit refusal otherwise. If the pattern recurs, add stage-specific few-shot examples to prompt templates.
- [Inline comments point to lines that no longer exist after force-push] → Review is anchored to a specific `commit_id`; new HEAD triggers a new review with a new marker; old review remains attached to the historical commit.
- [Producer and reviewer end up with the same Codex model] → Configuration contract requires distinct values for `REVIEW_PRIMARY_MODEL` vs `REVIEW_SECONDARY_MODEL`; warning logged when they match.
- [Context exceeds model limit] → `REVIEW_MAX_CONTEXT_BYTES` and deterministic truncation; truncation visible in summary tripwires.
- [GitHub Reviews API rate-limit] → Single POST per review and idempotency lookup minimise calls; rate-limit failures leave the task in AI-review and the next monitor pass retries.
- [Crash between POST and MoveTask leaves duplicates] → Idempotency by `(reviewer, stage, HEAD-sha)` makes the next pass find the existing review and proceed to MoveTask without re-posting.
- [Architectural blur] → `gh` logic only in `prcommenter`; JSON parser only in `reviewparse`; prompts are external `.tmpl` files; `CoreOrch` only sees the runner interface.

## Migration Plan

1. Add `internal/agentmeta` with `AppendTrailer`/`ParseTrailer` and table-driven tests.
2. Wire `agentmeta.AppendTrailer` into commit construction inside three existing runners; add tests asserting trailer presence.
3. Extend `internal/config` with three new state IDs and reviewer-slot variables; update `.env.example` with empty values.
4. Extend `LinearTaskManagerConfig.ManagedStateIDs` to include AI-review state IDs.
5. Implement `internal/reviewrunner` (runner + reviewparse + prcommenter + prompts) with full table-driven tests on fake executor / commenter / task manager.
6. Add the three AI-review routes to `CoreOrch` with feature-flag handling and tests.
7. Switch the post-push transition target inside the three existing runners from `Need * Review` to `Need * AI Review` when AI review is enabled.
8. Wire the executor factory (two slots) and `ReviewRunner` construction in `cmd/orchv3`.
9. Update `architecture.md` with a new "Целевой Поток Review-Stage" section and code mapping.
10. Update `docs/proposal-runner.md` to mention the AI review step.
11. `go fmt ./...` + `go test ./...`.

**Rollback:** clear the three AI-review state IDs in environment. The feature flag turns off automatically and producer runners revert to direct human review transitions, with no code revert required.

## Open Questions

- **PR review language.** All in-project external communication is in Russian today. Recommended default: PR review body in Russian with English machine-readable keywords (`severity: blocker`, `category: bug`). Add an env switch when a non-RU reader joins the team.
- **Manual review CLI mode.** Out of scope for the first version. Add later only when a recurring diagnostic need appears.
