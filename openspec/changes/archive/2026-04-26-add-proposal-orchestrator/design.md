## Context

`architecture.md` определяет `CoreOrch` как координатор между `TaskManager`, `AgentExecutor`, `GitManager` и `Logger`. В текущем коде вертикальный slice proposal-run уже есть в `internal/proposalrunner`, а Linear-facing слой реализован в `internal/taskmanager`. Недостающая часть - минимальный orchestration layer, который выбирает задачи из Linear, передает их в существующий proposal runner и записывает результат обратно в Linear.

Текущее ограничение: proposal runner менять не нужно. Он остается модулем, который принимает один task description и возвращает PR URL. Новый слой должен адаптировать Linear task payload к этому входу и использовать результат runner-а для обновления задачи.

## Goals / Non-Goals

**Goals:**

- Добавить `CoreOrch` для proposal-stage как отдельный внутренний пакет с тестируемыми зависимостями.
- Обрабатывать задачи только из состояния `LINEAR_STATE_READY_TO_PROPOSE_ID`.
- Для каждой ready-задачи запускать существующий `proposalrunner.Run`.
- После успешного запуска прикреплять PR URL через `TaskManager.AddPR` и переводить задачу в `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.
- Логировать начало, успех, пропуск и ошибку обработки задач структурными событиями.
- Сохранить существующий single-run запуск proposal runner по тексту задачи.

**Non-Goals:**

- Не выделять `GitManager` из `proposalrunner` в рамках этой задачи.
- Не менять prompt, git workflow, commit/push/PR creation и comment behavior внутри proposal runner.
- Не реализовывать code-stage и archive-stage orchestration.
- Не добавлять distributed locking, daemon scheduler или параллельную обработку задач.

## Decisions

### Ввести `internal/coreorch`

Новый пакет должен содержать orchestration use case, а не детали Linear, GitHub или Codex. Минимальный публичный контракт:

- `RunProposalsOnce(ctx context.Context) error` - один проход по текущим managed tasks.
- `TaskManager` interface с методами, которые уже есть у `taskmanager.Manager`: `GetTasks`, `AddPR`, `MoveTask`.
- `ProposalRunner` interface с методом `Run(ctx, taskDescription) (string, error)`.

Альтернатива - встроить orchestration прямо в `cmd/orchv3/main.go`. Это быстрее, но смешивает CLI parsing, wiring и бизнес-поток. Отдельный пакет лучше соответствует `architecture.md` и проще тестируется без Linear/Codex/GitHub.

### Фильтровать `Ready to Propose` внутри CoreOrch

`TaskManager.GetTasks` сейчас возвращает задачи из всех managed states. `CoreOrch` должен явно выбирать только задачи, у которых `task.State.ID == cfg.TaskManager.ReadyToProposeStateID`.

Альтернатива - добавить в `TaskManager` отдельный метод для ready-to-propose. Это сужает API под один stage и дублирует уже имеющуюся выборку managed states. На текущем этапе достаточно фильтра в orchestration layer.

### Последовательная обработка задач

Первый вариант должен обрабатывать ready-задачи последовательно в порядке, возвращенном `TaskManager`. Ошибка одной задачи логируется с task context и возвращается как итоговая ошибка прохода; задачи, уже успешно обработанные до ошибки, не откатываются.

Альтернатива - параллельная обработка. Она раньше времени усложнит git/codex resource usage, rate limits и восстановление после частичных сбоев.

### Состояние менять только после PR attachment

Успешный порядок действий для задачи:

1. Сформировать proposal input из Linear task title, identifier, description и comments.
2. Запустить `ProposalRunner.Run`.
3. Вызвать `TaskManager.AddPR`.
4. Вызвать `TaskManager.MoveTask` в `Need Proposal Review`.

Если runner упал, PR не прикрепляется и статус не меняется. Если `AddPR` упал, статус не меняется, чтобы задача не попала в review без ссылки на артефакт. Если `MoveTask` упал после успешного `AddPR`, ошибка возвращается с контекстом; повторный проход может создать новый PR, поэтому в реализации стоит логировать такой частичный успех явно.

Альтернатива - сначала двигать статус, потом прикреплять PR. Это хуже для reviewer workflow: задача может оказаться в review без основного артефакта.

### CLI wiring через `orchestrate-proposals`

Добавить явный режим запуска `orchv3 orchestrate-proposals`, который загружает общий config, создает `taskmanager.Manager`, `proposalrunner.Runner` и запускает `coreorch.RunProposalsOnce`. Текущий запуск с текстом задачи через args/stdin остается путем прямого single-run proposal runner.

Альтернатива - запускать orchestration при пустом stdin и отсутствии args. Это меняет уже зафиксированное поведение CLI startup и может неожиданно обращаться к Linear.

## Risks / Trade-offs

- Частичный успех после `AddPR`, но до `MoveTask` -> логировать отдельное событие с task ID и PR URL; не скрывать ошибку от процесса.
- Повторный запуск после частичного сбоя может создать дополнительный proposal PR -> принять как временный trade-off до появления идемпотентности/locking; review state не должен выставляться без подтвержденного transition.
- `TaskManager.GetTasks` возвращает несколько managed states -> фильтрация в `CoreOrch` должна покрываться unit tests, чтобы code/archive задачи не попали в proposal runner.
- Комментарии Linear могут быть длинными -> input builder должен быть простым и детерминированным, без потери исходного description; обрезание или summarization не вводить на этом этапе.

## Migration Plan

1. Добавить `internal/coreorch` с интерфейсами, input builder и unit tests на happy path, фильтрацию states и ошибки runner/task manager.
2. Подключить новый режим в `cmd/orchv3`, сохранив existing single-run behavior.
3. Обновить `architecture.md`, потому что `CoreOrch` становится реализованным кодовым компонентом.
4. Запустить `go fmt ./...` и `go test ./...`.

Rollback: удалить CLI-команду и пакет `internal/coreorch`; существующий proposal runner и task manager не требуют миграции данных.

## Open Questions

- Нужно ли продолжать обработку следующих задач после ошибки одной задачи? Рекомендуемый вариант для первой версии: остановиться и вернуть ошибку, чтобы не скрывать системные сбои интеграций.
