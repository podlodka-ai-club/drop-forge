## Why

Apply и Archive запускаются после ревью человеком, поэтому последние комментарии в Linear часто содержат уточнения, замечания и решения, которые агент должен учитывать при реализации или архивировании. Сейчас для этих стадий нужно явно закрепить контракт: комментарии задачи подтягиваются из Linear и попадают в prompt выполнения, а отсутствие комментариев представляется явно и не ломает запуск.

## What Changes

- Уточнить построение Apply prompt: он должен включать идентификатор, заголовок, описание и актуальные комментарии Linear-задачи.
- Уточнить построение Archive prompt: он должен включать идентификатор, заголовок, описание и актуальные комментарии Linear-задачи.
- Для задач без комментариев prompt должен содержать явное указание, что комментарии отсутствуют, чтобы агент не ожидал скрытого контекста.
- Сохранить существующие требования к branch source, state transitions, логированию и тестируемым зависимостям.
- Добавить тестовые сценарии, которые предотвращают регресс: комментарии из task payload должны доходить до Apply и Archive runner input.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: уточнить требования к построению Apply и Archive input/prompt из Linear task payload с обязательным включением комментариев.
- `linear-task-manager`: уточнить, что комментарии возвращаются актуальными для downstream стадий `Ready to Code` и `Ready to Archive`, а не только для proposal retry.

## Impact

- `internal/coreorch`: builder'ы Apply и Archive input должны форматировать task comments в agent prompt и покрываться unit-тестами.
- `internal/applyrunner` и `internal/archiverunner`: контракты input остаются совместимыми, но тесты должны подтверждать получение prompt с комментариями.
- `internal/taskmanager`: при необходимости обновить тесты Linear payload, чтобы комментарии сохранялись для ready-to-code и ready-to-archive задач.
- `openspec/specs/proposal-orchestration/spec.md` и `openspec/specs/linear-task-manager/spec.md`: обновление требований.
