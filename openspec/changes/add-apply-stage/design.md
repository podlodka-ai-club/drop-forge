## Context

Текущий runtime запускает только proposal-stage. `CoreOrch` читает managed Linear tasks через `TaskManager`, фильтрует `Ready to Propose`, переводит задачу в `Proposing in Progress`, вызывает `ProposalRunner`, прикрепляет PR URL и переводит задачу в `Need Proposal Review`.

Конфигурация уже содержит state IDs для code-route: `LINEAR_STATE_READY_TO_CODE_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`. `TaskManager.ManagedStateIDs()` уже включает ready-to-code в набор входных очередей, но `CoreOrch` сейчас эти задачи только пропускает.

Apply-stage должна работать с уже созданной веткой proposal-задачи: во временной директории клонировать репозиторий, переключаться на ветку задачи, запускать OpenSpec Apply skill, затем коммитить и пушить изменения в ту же ветку. В отличие от proposal-stage, новый PR создавать не нужно.

## Goals / Non-Goals

**Goals:**

- Обработать задачи из `Ready to Code` в том же orchestration runtime, что и proposal-stage.
- Переводить задачу в `Code in Progress` до запуска реализации и в `Need Code Review` после успешного push.
- Изолировать apply workflow за тестируемым executor-контрактом, чтобы `CoreOrch` не знал деталей git, Codex CLI или OpenSpec skill.
- Получать branch/ref задачи из данных `TaskManager`, а не вычислять его повторно из title/identifier.
- Сохранить структурные логи и контекстные ошибки на уровне proposal-stage.

**Non-Goals:**

- Не добавлять параллельное выполнение задач.
- Не менять внутренний workflow proposal runner, кроме возможного переиспользования общих helper'ов.
- Не создавать новый PR на Apply-этапе.
- Не реализовывать archive-stage в рамках этого изменения.
- Не выделять полноценный `GitManager`, если минимальная реализация Apply не требует этого прямо сейчас.

## Decisions

### D1. Apply executor как отдельный контракт

В `coreorch` добавить зависимость уровня:

```go
type ApplyRunner interface {
	Run(ctx context.Context, input ApplyInput) error
}
```

`ApplyInput` должен содержать task identity, agent prompt/context и branch/ref задачи. Реальная реализация может жить в новом пакете `internal/applyrunner` или в соседнем пакете с proposal runner, но orchestration layer зависит только от интерфейса.

Альтернатива: встроить apply git-команды прямо в `coreorch`. Отброшено, потому что нарушает текущую границу ответственности: `CoreOrch` координирует статусы и executor'ы, но не управляет git/agent runtime.

### D2. Один monitor pass обрабатывает proposal и apply routes

Сохранить текущий default CLI runtime как долгоживущий monitor, но расширить проход: он загружает managed tasks один раз и последовательно маршрутизирует задачи по state ID:

- `Ready to Propose` -> proposal flow;
- `Ready to Code` -> apply flow;
- остальные managed states -> skip log.

Для совместимости имена `RunProposalsOnce`/`RunProposalsLoop` можно оставить как алиасы или переименовать аккуратно только внутри кода и тестов. Публичный CLI по-прежнему запускается без аргументов.

Альтернатива: добавить второй отдельный loop для apply. Отброшено на текущем этапе: два loop'а будут независимо дергать один TaskManager и усложнят порядок обработки без подтвержденной необходимости.

### D3. Branch/ref приходит из TaskManager payload

Apply-stage не должна заново угадывать branch name из title/identifier, потому что proposal branch содержит timestamp и slug, а PR URL уже создается и прикрепляется к Linear-задаче proposal-stage. Минимальное расширение `taskmanager.Task`:

```go
type PullRequest struct {
	URL    string
	Branch string
}
```

`Task` получает `PullRequests []PullRequest` или эквивалентное поле с последним/основным PR. Linear client должен читать attachment'ы Pull Request, прикрепленные через `AddPR`, и возвращать URL. Branch можно извлечь executor'ом через `gh pr view <url> --json headRefName` либо TaskManager может заполнить `Branch`, если API позволяет получить head ref без дополнительного git/GitHub запроса.

Для первого slice достаточно нормативно требовать валидный PR URL у ready-to-code задачи и извлекать branch в apply runner через GitHub CLI после clone. Если PR URL отсутствует, Apply не запускается и задача не переводится в code review.

Альтернатива: хранить branch в отдельной Linear comment convention. Отброшено как хрупкий текстовый контракт.

### D4. Apply runner повторяет минимальный git lifecycle без PR creation

Workflow:

1. Validate config and input.
2. Create temp dir.
3. Clone configured repository.
4. Resolve checkout branch from input branch or PR URL.
5. Checkout branch.
6. Run Codex/OpenSpec Apply agent with task context in clone dir.
7. Check `git status --short`; fail if no changes were produced unless implementation explicitly decides to allow no-op later.
8. `git add -A`, `git commit -m "<identifier/title apply message>"`, `git push <remote> <branch>`.
9. Preserve or cleanup temp dir according to config.

Commit message can use `Apply: <Identifier>: <Title>` with the same display-name normalization as proposal runner.

Альтернатива: run apply directly in the repository working tree. Отброшено: proposal runner уже использует isolated temp clone, and this avoids mutating the operator's checkout.

### D5. OpenSpec Apply запускается через agent executor

Apply runner should use the existing Codex CLI executor pattern, but with prompt/instructions that explicitly tell the agent to use the local OpenSpec Apply skill for the change already present in the checked-out branch. The exact command remains implementation detail; the observable contract is that the executor runs implementation from specs and returns success/failure.

If the checked-out branch contains multiple pending OpenSpec changes, the prompt should include task context and let the Apply skill determine the relevant change from repository state; if that is ambiguous, the run should fail with context rather than applying an arbitrary change.

## Risks / Trade-offs

- [PR attachment does not expose branch] -> Mitigation: require PR URL in task payload and resolve head branch through GitHub CLI in Apply runner.
- [Ready-to-code task without proposal PR] -> Mitigation: fail before moving to `Need Code Review`; after moving to `Code in Progress`, return contextual error and keep the task there for operator intervention.
- [No-op apply run] -> Mitigation: treat empty git status as an error in the first implementation, matching proposal runner behavior.
- [Shared proposal/apply git code duplication] -> Mitigation: accept small duplication initially; extract `GitManager` only after both flows make the common shape clear.
- [Monitor name remains proposal-focused] -> Mitigation: implementation may keep old method names temporarily for compatibility, but docs/specs should describe the runtime as orchestration monitor once Apply is included.

## Migration Plan

1. Extend task payload and Linear query/tests to expose PR attachment URL for ready-to-code tasks.
2. Add apply runner contract and fakeable orchestration route in `coreorch`.
3. Implement real apply runner with temp clone, branch checkout, OpenSpec Apply execution, commit, and push.
4. Wire apply runner in CLI default runtime.
5. Update `architecture.md` because component interactions and executor responsibilities change.
6. Run `go fmt ./...` and `go test ./...`.

Rollback: disable Apply by not configuring or not moving tasks into `Ready to Code`; code-level rollback is reverting this change, because existing proposal route remains separate.

## Open Questions

- Should the first implementation require `gh pr view` to resolve the branch from PR URL, or should `TaskManager` query Linear/GitHub metadata deeply enough to return branch directly?
  - Recommended option: resolve via `gh pr view` in Apply runner for the first slice, because the runner already owns GitHub CLI interactions and this keeps TaskManager focused on task tracker data.
