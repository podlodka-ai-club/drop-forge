## Why

Текущие Telegram-уведомления отправляются на каждую смену статуса и содержат в основном технические ID. Из-за этого чат получает шум от промежуточных in-progress переходов, а человеку приходится вручную искать задачу и связанный PR, когда реально нужен review.

## What Changes

- Отправлять Telegram-сообщения по `task.status_changed` только для целевых review-состояний:
  - `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`
  - `LINEAR_STATE_NEED_CODE_REVIEW_ID`
  - `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`
- Обогащать событие смены статуса данными задачи, доступными в оркестрации: readable identifier, title и PR URL/branch source.
- Обновить формат Telegram-сообщения так, чтобы оно показывало человекочитаемую задачу и ссылку на PR, если ссылка доступна.
- Сохранить best-effort модель доставки: ошибка Telegram не откатывает успешный переход статуса.

## Capabilities

### New Capabilities

Нет.

### Modified Capabilities

- `telegram-notifications`: фильтрация уведомлений по review-состояниям и новый человекочитаемый формат сообщения с PR ссылкой.
- `orchestration-events`: payload `task.status_changed` должен поддерживать optional PR URL/branch source для downstream уведомлений.
- `linear-task-manager`: публикация status change event должна поддерживать передачу расширенного контекста задачи, когда caller им располагает.

## Impact

- `internal/events`: расширение `TaskStatusChanged` optional-полями для PR URL/branch.
- `internal/taskmanager`: расширение контракта публикации status event без нарушения существующего `MoveTask`.
- `internal/coreorch`: передача task metadata и PR URL при переходах в review-состояния.
- `internal/notifications/telegram`: фильтр review-состояний и форматирование сообщения.
- Тесты: unit tests для event payload, task manager publishing, orchestration review moves и Telegram formatter/handler.
