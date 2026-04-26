## Why

Сейчас TaskManager не переводит Linear-задачу в промежуточный статус перед запуском Proposing, поэтому внешне не видно, что задача уже взята в работу. Также конфигурация статусов не готова к аналогичным промежуточным колонкам для будущих стадий Code и Archiving.

## What Changes

- Добавить обязательный runtime-параметр для Linear-статуса `Proposing in Progress`.
- Перед запуском Proposing переводить задачу из `Ready to Propose` в настроенный статус `Proposing in Progress`.
- Добавить runtime-параметры для будущих Linear-статусов `Code in Progress` и `Archiving in Progress`, чтобы `.env.example` и загрузка конфигурации уже знали эти колонки.
- Сохранить текущий финальный переход после успешного Proposing в `Need Proposal Review`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `linear-task-manager`: меняются требования к runtime-конфигурации Linear-статусов и сценарию перехода задачи при обработке Proposing.
- `proposal-orchestration`: меняется порядок state transitions для proposal-stage перед запуском proposal runner.

## Impact

- `internal/config`: структура конфигурации, загрузка ENV, валидация и тесты.
- `internal/coreorch`: порядок переходов при `RunProposalsOnce` и тесты на мутации TaskManager.
- `.env.example`: новые ключи без значений.
- `openspec/specs/linear-task-manager/spec.md`: требования к промежуточным статусам и proposal-flow.
