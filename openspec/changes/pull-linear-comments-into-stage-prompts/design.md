## Context

`TaskManager` уже возвращает `taskmanager.Task` с `Description` и `Comments`, а orchestration layer строит input для proposal, Apply и Archive runner-ов из этого payload. Proposal-контракт явно требует включать комментарии в `AgentPrompt`, но для Apply и Archive это зафиксировано слабее: требования говорят про task context, не определяя, что Linear comments обязательно попадают в prompt агента.

Для DRO-27 важно именно поведение downstream стадий: когда человек оставляет замечания в Linear перед переводом задачи в `Ready to Code` или `Ready to Archive`, следующий agent run должен получить эти замечания как часть prompt.

## Goals / Non-Goals

**Goals:**

- Сделать комментарии Linear обязательной частью Apply и Archive `AgentPrompt`.
- Единообразно форматировать identity, title, description и comments для всех stage runner inputs.
- Явно представлять отсутствие комментариев строкой `No comments available.`
- Покрыть builder-контракт тестами без сетевых вызовов, Codex CLI, GitHub CLI или Linear API.

**Non-Goals:**

- Не менять Linear API схему, если текущий `TaskManager` уже возвращает comments для managed tasks.
- Не добавлять отдельное хранилище комментариев или convention в Linear comments.
- Не менять git workflow Apply/Archive runner-ов.
- Не добавлять повторное чтение задачи после перевода в in-progress в рамках этого изменения.

## Decisions

### D1. Comments форматируются в orchestration input builder'ах

`coreorch` остается местом, где Linear task payload превращается в prompt для агента. `BuildApplyInput` и `BuildArchiveInput` должны передавать runner-ам `AgentPrompt`, построенный тем же formatter'ом, что и proposal input: `Linear task`, `Description`, `Comments`.

Альтернатива: читать комментарии непосредственно в Apply/Archive runner-ах. Отброшено, потому что runner-ы должны оставаться исполнителями git/agent workflow и не зависеть от Linear.

### D2. Отсутствие comments является явным prompt context

Если `Task.Comments` пустой, formatter должен включать `No comments available.`. Это сохраняет запуск валидным и делает отсутствие ревью-контекста видимым для агента.

Альтернатива: пропускать секцию comments. Отброшено, потому что агенту и логам сложнее отличить отсутствие комментариев от ошибки сборки prompt.

### D3. Актуальность comments обеспечивается обычным чтением managed tasks

Перед каждой orchestration pass `TaskManager.GetTasks` читает managed tasks из Linear. Требование к `linear-task-manager` уточняется: comments должны возвращаться для всех managed states, включая `Ready to Code` и `Ready to Archive`, чтобы downstream stages получали последние доступные комментарии из task payload.

Альтернатива: делать дополнительный Linear lookup внутри каждой стадии. Отброшено для первого slice, потому что это увеличивает сетевые вызовы и дублирует ответственность TaskManager.

## Risks / Trade-offs

- [Большие цепочки комментариев могут раздувать prompt] -> Mitigation: на первом этапе передаем все доступные comments как текущий источник правды; лимиты/сжатие добавлять только при реальной проблеме.
- [Комментарии изменились после `GetTasks`, но до запуска runner] -> Mitigation: orchestration pass использует консистентный snapshot task payload; дополнительный refresh можно добавить позже, если появится требование.
- [Пустые или безымянные comments ухудшают читаемость] -> Mitigation: formatter использует `(empty comment)` и `Unknown author`, сохраняя порядок и диагностируемость.

## Migration Plan

1. Уточнить delta specs для `proposal-orchestration` и `linear-task-manager`.
2. Проверить/обновить `coreorch` builder'ы Apply и Archive input, чтобы `AgentPrompt` включал comments.
3. Добавить unit-тесты на Apply и Archive input с комментариями и без комментариев.
4. При необходимости обновить taskmanager/linear tests для ready-to-code и ready-to-archive payload comments.
5. Запустить `go fmt ./...`, `go test ./...` и `openspec status --change pull-linear-comments-into-stage-prompts`.

Rollback: вернуть builder'ы к прежнему prompt без comments и откатить delta specs; внешний runtime workflow и state transitions не меняются.

## Open Questions

- Нужно ли ограничивать количество comments в prompt?
  - Recommended option: не ограничивать в рамках DRO-27, потому что задача формулирует требование подтягивать комментарии, а политика сжатия может привести к потере важного human feedback.
