## 1. Event Core

- [ ] 1.1 Добавить внутренний пакет событий с типами event, publisher/handler interfaces и константой `task.status_changed`.
- [ ] 1.2 Реализовать локальный dispatcher с регистрацией подписчиков по типу события и синхронной публикацией.
- [ ] 1.3 Покрыть dispatcher unit-тестами для matching event, unrelated event, нескольких подписчиков и ошибки подписчика.

## 2. TaskManager Event Publishing

- [ ] 2.1 Добавить в `TaskManager` опциональный publisher без изменения поведения при nil publisher.
- [ ] 2.2 Публиковать `task.status_changed` после успешного `MoveTask` с task ID, target state ID и timestamp.
- [ ] 2.3 Логировать ошибку публикации события без возврата ошибки из успешного `MoveTask`.
- [ ] 2.4 Добавить тесты `TaskManager` для успешной публикации, отсутствия публикации при ошибке Linear и сохранения успеха при ошибке publisher-а.

## 3. Telegram Notifications

- [ ] 3.1 Добавить Telegram config: enable flag, bot token, chat ID, API URL и timeout.
- [ ] 3.2 Обновить `.env.example` Telegram-ключами без значений.
- [ ] 3.3 Реализовать Telegram subscriber для `task.status_changed` через стандартный `net/http` и `sendMessage`.
- [ ] 3.4 Добавить форматирование сообщения с human-readable полями и fallback на task ID / target state ID.
- [ ] 3.5 Покрыть Telegram config и notifier unit-тестами через локальный HTTP server или подменяемый transport.

## 4. Runtime Wiring

- [ ] 4.1 Подключить dispatcher в `cmd/orchv3` и передать publisher в `TaskManager`.
- [ ] 4.2 Регистрировать Telegram subscriber только при `TELEGRAM_NOTIFICATIONS_ENABLED=true`.
- [ ] 4.3 Проверить CLI wiring тестами для включенных и выключенных Telegram-уведомлений без реального Telegram API.

## 5. Documentation And Verification

- [ ] 5.1 Обновить `architecture.md`, описав event dispatcher и Telegram subscriber как расширение текущего orchestration flow.
- [ ] 5.2 Запустить `go fmt ./...`.
- [ ] 5.3 Запустить `go test ./...`.
