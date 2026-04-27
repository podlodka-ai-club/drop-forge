## Why

`GitManager` уже описан как отдельная архитектурная ответственность, но фактическая реализация git/gh-операций размазана по `internal/proposalrunner`, `internal/applyrunner` и `internal/archiverunner`. Это усложняет поддержку runner-ов, дублирует сценарии `clone`, `checkout`, `add`, `commit`, `push` и PR-операций, а также мешает тестировать git lifecycle как отдельную границу.

## What Changes

- Выделить новый внутренний пакет `internal/gitmanager`, который инкапсулирует операции с repository/workspace, ветками, commit/push и GitHub PR через существующий command runner.
- Перевести `proposalrunner`, `applyrunner` и `archiverunner` на использование `GitManager` вместо прямой сборки команд `git` и `gh` внутри runner-ов.
- Сохранить текущее внешнее поведение proposal, apply и archive stages: isolated clone workspace, логирование шагов, ошибки с контекстом, commit/push/PR правила и отсутствие нового PR для Apply/Archive.
- Обновить архитектурное описание, чтобы маппинг текущего кода отражал фактически выделенный `GitManager`.
- Покрыть новый пакет unit-тестами с fake command runner и адаптировать runner-тесты под новую зависимость.

## Capabilities

### New Capabilities
- `git-manager`: внутренний сервис для управления clone workspace, branch lifecycle, commit/push и GitHub PR-операциями, переиспользуемый runner-ами.

### Modified Capabilities
- `codex-proposal-pr-runner`: proposal runner должен использовать выделенный `GitManager`, сохраняя прежний пользовательский workflow и тестируемость.
- `proposal-orchestration`: Apply и Archive runner-ы должны использовать выделенный `GitManager` для работы с task branch, сохраняя текущие orchestration-контракты.

## Impact

- Затронутые пакеты: `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`, новый `internal/gitmanager`, тесты этих пакетов.
- Техническая зависимость: существующий `internal/commandrunner` остается низкоуровневым адаптером запуска внешних команд и используется новым пакетом.
- Архитектура: требуется обновить `architecture.md`, потому что изменение выделяет ранее описанный внутренний сервис и меняет маппинг ответственности на код.
- Runtime-конфигурация и `.env.example` не должны измениться, если выделение пакета не добавит новых runtime-параметров.
