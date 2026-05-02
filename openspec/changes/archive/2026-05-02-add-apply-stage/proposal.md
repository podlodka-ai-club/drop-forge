## Why

Оркестратор уже умеет проходить proposal-этап: брать Linear-задачи из `Ready to Propose`, запускать агента во временном клоне и переводить задачу на review. Следующий практический этап цикла - Apply: после принятия спеки задача должна автоматически перейти из `Ready to Code` в реализацию, получить изменения в ветке задачи и уйти в `Need Code Review`.

## What Changes

- Добавить Apply-стадию оркестрации для задач в `LINEAR_STATE_READY_TO_CODE_ID`.
- Apply-стадия должна переводить задачу в `LINEAR_STATE_CODE_IN_PROGRESS_ID` до запуска реализации.
- Добавить executor для Apply, который во временной директории клонирует репозиторий задачи, переключается на ветку задачи, запускает агентскую реализацию через OpenSpec Apply skill, затем коммитит и пушит изменения в ту же ветку.
- После успешного Apply оркестратор должен переводить Linear-задачу в `LINEAR_STATE_NEED_CODE_REVIEW_ID`.
- Сохранить подход proposal-цикла к логированию, контекстным ошибкам, cleanup/preserve временной директории и тестируемым зависимостям, но без создания нового PR.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: добавить Apply-стадию в общий orchestration runtime рядом с proposal-мониторингом.
- `linear-task-manager`: уточнить, что `Ready to Code`, `Code in Progress` и `Need Code Review` используются Apply-стадией как input, in-progress и review transitions.

## Impact

- `internal/coreorch`: новый Apply route, интерфейс executor'а, обработка состояний и тесты.
- `internal/proposalrunner` или новый пакет executor'а: переиспользование паттерна временного клона/git workflow для реализации без создания PR.
- `cmd/orchv3`: wiring Apply-стадии в runtime рядом с proposal loop.
- `internal/config` и `.env.example`: ожидается использование уже существующих Linear state переменных для code-route; при необходимости добавить отдельные runtime-настройки для Apply executor'а без значений по умолчанию в `.env.example`.
- `openspec/specs/proposal-orchestration/spec.md` и `openspec/specs/linear-task-manager/spec.md`: обновление требований.
