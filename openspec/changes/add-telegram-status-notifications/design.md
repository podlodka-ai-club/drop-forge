## Context

Оркестратор уже имеет централизованный путь смены статуса задач через `TaskManager.MoveTask`, а `CoreOrch` использует этот путь для proposal, apply и archive переходов. Сейчас у смены статуса нет доменного события: чтобы добавить Telegram-уведомление, пришлось бы напрямую встраивать отправку сообщения в task manager или orchestration flow.

Изменение вводит локальную Event Driven точку расширения: компоненты публикуют внутренние события, подписчики реагируют на них. Первым подписчиком будет Telegram notifier для события смены статуса задачи.

## Goals / Non-Goals

**Goals:**
- Добавить внутренний event dispatcher без внешней инфраструктуры.
- Публиковать событие `TaskStatusChanged` после успешного перехода задачи в Linear.
- Отправлять Telegram-сообщение в настроенный чат по событию смены статуса.
- Держать Telegram-настройки в `.env` / environment variables и синхронизировать `.env.example`.
- Сохранить тестируемость без реальных Linear и Telegram API вызовов.

**Non-Goals:**
- Не добавлять Kafka, NATS, RabbitMQ, Redis Streams или другую durable queue.
- Не гарантировать доставку уведомлений после рестарта процесса.
- Не добавлять пользовательские шаблоны сообщений.
- Не менять workflow статусов, набор Linear колонок или логику proposal/apply/archive runner-ов.

## Decisions

### Локальный синхронный dispatcher

Добавить небольшой внутренний пакет, например `internal/events`, с типами:
- `Event` с полями `Type`, `OccurredAt`, `Payload`.
- `TaskStatusChanged` как payload для смены статуса.
- `Subscriber` / `Handler` interface.
- `Dispatcher`, который регистрирует handlers и вызывает их при `Publish`.

Альтернатива с внешним брокером отклонена: для текущей задачи нужен один процесс и один Telegram-подписчик, а отдельная инфраструктура добавит настройку, эксплуатационные отказы и тестовую сложность до появления реальной потребности.

### Событие публикуется после успешного MoveTask

`TaskManager.MoveTask` остается центральной точкой фактической смены статуса. После успешного ответа Linear manager публикует `TaskStatusChanged` с обязательными `TaskID`, `TargetStateID`, `OccurredAt` и опциональными человекочитаемыми полями, если они доступны текущему вызывающему коду или будущим расширениям.

Если публикация события или подписчик вернул ошибку, `MoveTask` логирует ошибку уведомления, но не отменяет уже выполненный переход и не возвращает ошибку вызывающему orchestration flow. Это сохраняет текущее публичное поведение: статус задачи важнее best-effort уведомления.

### Telegram notifier как подписчик

Добавить внутренний пакет или адаптер, например `internal/notifications/telegram`, который реализует handler для `TaskStatusChanged`. Handler строит короткое сообщение с fallback-значениями:
- task identifier или task ID;
- title, если доступен;
- target state name или target state ID.

Отправка выполняется через стандартный `net/http` в Telegram Bot API `sendMessage`. HTTP client и API URL должны быть подменяемыми в тестах.

### Runtime wiring в CLI

`cmd/orchv3` при старте загружает конфигурацию, создает dispatcher, регистрирует Telegram notifier только когда уведомления включены, и передает dispatcher в `TaskManager`. Если уведомления выключены, используется пустой dispatcher или nil-safe publisher.

Новые переменные:
- `TELEGRAM_NOTIFICATIONS_ENABLED`
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`
- `TELEGRAM_API_URL`
- `TELEGRAM_TIMEOUT`

Token, chat ID и API URL валидируются только при включенных уведомлениях. В `.env.example` добавляются только ключи без значений.

## Risks / Trade-offs

- Потеря уведомлений при рестарте или сетевой ошибке -> принято как best-effort trade-off; durable delivery не требуется для DRO-45 и может быть добавлена позже через другой dispatcher implementation.
- Ошибка Telegram скрыта от orchestration result -> ошибка логируется структурно, а смена статуса не ломается после успешной мутации Linear.
- Событие из `TaskManager.MoveTask` на первом этапе может содержать только task ID и target state ID -> Telegram message использует fallback, а event payload оставляет опциональные поля для будущего обогащения без изменения типа события.
- Синхронный вызов подписчиков добавляет latency к `MoveTask` -> для одного Telegram-запроса это приемлемо; timeout ограничивается конфигурацией.

## Migration Plan

1. Добавить event dispatcher и тесты.
2. Добавить Telegram config, notifier и тесты HTTP-запроса.
3. Подключить publisher к `TaskManager` и покрыть успешную публикацию / ошибку подписчика.
4. Подключить dispatcher и Telegram notifier в CLI wiring.
5. Обновить `.env.example` и `architecture.md`, потому что появляется новая точка расширения между внутренними компонентами.
6. Запустить `go fmt ./...` и `go test ./...`.

Rollback: выключить `TELEGRAM_NOTIFICATIONS_ENABLED`; при необходимости удалить регистрацию notifier-а, оставив dispatcher без подписчиков.

## Open Questions

Нет открытых вопросов для начала реализации. Шаблон Telegram-сообщения предлагается зафиксировать минимальным и не делать runtime-настраиваемым до появления реального требования.
