## Why

Стадии `archive` и `apply` принимают решение на основе prompt, собранного из задачи Linear. Если комментарии ревью не попадают в этот prompt, агент теряет актуальные указания от человека и может повторить уже отмеченные ошибки.

## What Changes

- Вход для `apply` и `archive` должен включать комментарии Linear вместе с идентификатором, заголовком и описанием задачи.
- При отсутствии комментариев prompt должен явно содержать `No comments available.`, чтобы downstream-агент не трактовал пустой блок как ошибку сборки контекста.
- Тесты должны проверять, что комментарии с автором, временем и телом доходят до `ApplyInput.AgentPrompt` и `ArchiveInput.AgentPrompt`.
- Новых runtime-переменных и внешних зависимостей не требуется.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: уточняется контракт подготовки входа для стадий `apply` и `archive`: комментарии Linear обязательны как часть agent prompt.

## Impact

- `internal/coreorch`: сборка `ApplyInput` и `ArchiveInput`, форматирование общего agent prompt, unit-тесты маршрутизации и подготовки входа.
- `internal/applyrunner` и `internal/archiverunner`: поведение prompt остается прежним, но тесты должны подтвердить, что переданный orchestration prompt без потерь попадает в Codex prompt.
- `openspec/specs/proposal-orchestration`: добавляется требование к передаче комментариев в prompt для `apply` и `archive`.
