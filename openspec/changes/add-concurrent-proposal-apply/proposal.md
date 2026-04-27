## Why

Сейчас один проход оркестратора обрабатывает найденные задачи последовательно: пока proposal-задача выполняет долгий Codex/OpenSpec workflow, apply-задача из того же polling pass ждет своей очереди. Из-за этого нельзя одновременно вести две независимые работы, хотя proposal и apply не требуют общего executor-состояния.

## What Changes

- Изменить orchestration pass так, чтобы задачи proposal и apply запускались в отдельных горутинах и могли выполняться одновременно.
- Сохранить stage-specific lifecycle: каждая задача все еще сначала переводится в свой in-progress state, затем запускает соответствующий runner, затем переводится в review state только после успеха.
- Собирать ошибки из параллельных задач и возвращать из pass контекстную ошибку, не отменяя автоматически другую уже запущенную задачу из-за failure соседней.
- Сохранить последовательную обработку внутри одного stage на этом этапе: одна proposal-задача и одна apply-задача могут идти одновременно, но две proposal-задачи между собой и две apply-задачи между собой не распараллеливаются.
- Archive-route остается без изменения и продолжает выполняться последовательно, если в pass есть archive-задача.

## Capabilities

### New Capabilities

Нет.

### Modified Capabilities

- `proposal-orchestration`: изменить требование о последовательной маршрутизации proposal/apply так, чтобы proposal и apply могли выполняться параллельно в одном orchestration pass.

## Impact

- `internal/coreorch`: планирование задач в `RunProposalsOnce`, запуск stage handlers в горутинах, сбор ошибок, тесты на concurrency и сохранение lifecycle-порядка внутри задачи.
- `architecture.md`: обновить описание orchestration flow, потому что меняется взаимодействие между `CoreOrch` и stage executors.
- Публичные CLI-флаги, runtime config, TaskManager API и runner API не меняются.
