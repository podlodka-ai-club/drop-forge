# Linear TaskManager

`TaskManager` в этом репозитории — это внутренний Linear-facing слой для будущего `CoreOrch`, а не отдельный orchestration loop.

## Scope первой итерации

`TaskManager` умеет:

- читать задачи только из одного настроенного проекта Linear;
- выбирать только задачи из управляемых state'ов `ready to propose`, `ready to code`, `ready to archive`;
- возвращать данные задачи в форме, пригодной для `CoreOrch`: `id`, `identifier`, `title`, `description`, текущий `state`, `comments`;
- выполнять write-операции обратно в Linear: `MoveTask`, `AddComment`, `AddPR`.

Для review-этапов `TaskManager` не выбирает target state сам. `CoreOrch` должен взять нужный review state ID из конфига и явно вызвать `MoveTask(...)`, например для:

- `Need Proposal Review`
- `Need Code Review`
- `Need Archive Review`

In-progress state IDs также являются только target-переходами. `TaskManager` валидирует и хранит их в конфиге, но не добавляет в managed input queues. Текущий proposal-pass в `CoreOrch` перед запуском runner вызывает `MoveTask(...)` с `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, а затем после успешного PR attachment переводит задачу в `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.

Для HITL-сценария `TaskManager` возвращает все comments задачи без дополнительной фильтрации по автору или релевантности. Отбор нужного feedback остается ответственностью `CoreOrch`.

## Что не входит в scope

Первая итерация `TaskManager` не делает:

- polling задач по расписанию;
- dispatch задач в executor'ы;
- orchestration retry / lease / locking;
- чтение существующих PR association как обязательную часть read-модели.

Эти обязанности должны появиться в `CoreOrch` или в следующих change'ах.

## Конфигурация

`TaskManager` использует runtime-параметры из `.env` / process environment:

- `LINEAR_API_URL`
- `LINEAR_API_TOKEN`
- `LINEAR_PROJECT_ID`
- `LINEAR_STATE_READY_TO_PROPOSE_ID`
- `LINEAR_STATE_READY_TO_CODE_ID`
- `LINEAR_STATE_READY_TO_ARCHIVE_ID`
- `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- `LINEAR_STATE_CODE_IN_PROGRESS_ID`
- `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`
- `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`
- `LINEAR_STATE_NEED_CODE_REVIEW_ID`
- `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`

`.env.example` содержит все ключи без значений по текущему контракту проекта.
