# Linear TaskManager

`TaskManager` в этом репозитории — это внутренний Linear-facing слой для будущего `CoreOrch`, а не отдельный orchestration loop.

## Scope первой итерации

`TaskManager` умеет:

- читать задачи только из одного настроенного проекта Linear;
- выбирать только задачи из управляемых state'ов `ready to propose`, `ready to code`, `ready to archive`;
- возвращать данные задачи в форме, пригодной для `CoreOrch`: `id`, `identifier`, `title`, `description`, текущий `state`, `comments`, attached Pull Request URL для Apply и Archive;
- выполнять write-операции обратно в Linear: `MoveTask`, `AddComment`, `AddPR`.

Для review-этапов `TaskManager` не выбирает target state сам. `CoreOrch` должен взять нужный review state ID из конфига и явно вызвать `MoveTask(...)`, например для:

- `Need Proposal Review`
- `Need Code Review`
- `Need Archive Review`

Для HITL-сценария `TaskManager` возвращает все comments задачи без дополнительной фильтрации по автору или релевантности. Отбор нужного feedback остается ответственностью `CoreOrch`.

## Что не входит в scope

Первая итерация `TaskManager` не делает:

- polling задач по расписанию;
- dispatch задач в executor'ы;
- orchestration retry / lease / locking;
- получение branch name из GitHub metadata.

Branch name для Apply и Archive при необходимости определяют stage-specific runner'ы через `gh pr view` по attached PR URL.

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
