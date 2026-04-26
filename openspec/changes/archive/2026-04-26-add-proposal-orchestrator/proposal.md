## Why

Сейчас proposal-runner умеет выполнить один запуск по переданному описанию задачи, а `TaskManager` умеет читать и обновлять задачи Linear, но в системе нет слоя, который связывает эти части в рабочий proposal-stage. Нужно добавить минимальный `CoreOrch`, который сам выбирает задачи, готовые к proposal, запускает по ним существующий runner и переводит задачи на ручной review.

## What Changes

- Добавить proposal orchestration loop, который через `TaskManager` получает задачи из состояния `LINEAR_STATE_READY_TO_PROPOSE_ID`.
- Для каждой найденной задачи запускать существующий proposal runner без изменения его внутреннего workflow.
- После успешного proposal-run прикреплять PR к задаче и переводить ее в состояние `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.
- При ошибках proposal-run или обновления Linear логировать контекст и не переводить задачу в review state.
- Сохранить текущий CLI single-run сценарий или совместимый путь запуска, чтобы существующий runner оставался доступен отдельно.

## Capabilities

### New Capabilities

- `proposal-orchestration`: Координация proposal-stage через `TaskManager`, существующий proposal runner и workflow transitions в Linear.

### Modified Capabilities

- `linear-task-manager`: Уточнение использования task manager оркестратором для выборки только `Ready to Propose`, прикрепления PR и перехода в `Need Proposal Review`.

## Impact

- Код: `cmd/orchv3`, новый или расширенный внутренний orchestration пакет, конфигурация и тесты.
- Интеграции: Linear через существующий `TaskManager`, GitHub PR URL через результат существующего proposal runner.
- Конфигурация: используются существующие Linear state IDs, включая `LINEAR_STATE_READY_TO_PROPOSE_ID` и `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`; новые переменные добавлять только если выбранный режим запуска потребует явного управления циклом.
- Архитектура: реализуется роль `CoreOrch` из `architecture.md`, не меняя границы `proposalrunner`.
