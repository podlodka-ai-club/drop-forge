## 1. Конфиг и контракт TaskManager

- [x] 1.1 Расширить `internal/config` настройками Linear, обязательным project filter и маппингом управляемых state ID `ready to propose`, `ready to code` и `ready to archive`.
- [x] 1.2 Обновить `.env.example` всеми новыми `LINEAR_*` ключами без значений и добавить unit-тесты на загрузку, валидацию, приоритет process env и ошибку при отсутствии project filter/state mapping.
- [x] 1.3 Описать минимальные внутренние типы `TaskManager`: модель задачи, включающую описание, текущий state и комментарии, а также операции `GetTasks`, `MoveTask`, `AddComment` и `AddPR`.
- [x] 1.4 Расширить `internal/config` и `.env.example` отдельными review target state ID для `Need Proposal Review`, `Need Code Review` и `Need Archive Review`.

## 2. Чтение задач из Linear

- [x] 2.1 Добавить пакет `internal/taskmanager` и/или внутренний Linear adapter для загрузки задач только из настроенного проекта и управляемых state'ов.
- [x] 2.2 Реализовать возврат данных задачи в едином payload для `CoreOrch`: идентификатор, key, описание, state и комментарии.
- [x] 2.3 Покрыть чтение задач тестами на project-scoped фильтрацию, фильтрацию по state, стабильный payload без description/comments и возврат обновленных комментариев после повторного получения задачи.
- [x] 2.4 Добавить отдельный тест сценария повторного чтения задачи: после reject и новых human comments следующий вызов `GetTasks` через `CoreOrch -> TaskManager` возвращает ту же задачу с обновленным feedback.

## 3. Запись обратно в Linear

- [x] 3.1 Реализовать в `TaskManager` операцию перевода задачи в новый state.
- [x] 3.2 Реализовать в `TaskManager` операцию публикации комментария в задаче.
- [x] 3.3 Реализовать в `TaskManager` операцию привязки PR к задаче.
- [x] 3.4 Покрыть write-операции тестами на корректное формирование запросов, привязку данных к правильной задаче и обработку ошибок Linear API.
- [x] 3.5 Добавить негативные тесты на невалидный PR URL, частично заполненный ответ Linear и сетевые/transport ошибки клиента.
- [x] 3.6 Добавить тесты и wiring для использования review target state ID в вызовах `MoveTask`, чтобы `CoreOrch` мог переводить задачи в `Need Proposal Review`, `Need Code Review`, `Need Archive Review`.

## 4. Wiring и документация

- [x] 4.1 Добавить structured logging для чтения и записи задач через `taskmanager` и `linear` модули.
- [x] 4.2 Зафиксировать в коде и документации, что `TaskManager` является Linear-facing слоем для будущего `CoreOrch`, а не orchestration-loop.
- [x] 4.3 Обновить README и связанную документацию описанием project-scoped TaskManager и передачи комментариев вместе с описанием задачи.
- [x] 4.4 Добавить unit-тесты на structured logging ошибок и операций чтения/записи с контекстом project/state/task.
- [x] 4.5 Прогнать `go fmt ./...` и `go test ./...`.
