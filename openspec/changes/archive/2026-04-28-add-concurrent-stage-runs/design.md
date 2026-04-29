## Context

Текущий `CoreOrch` загружает managed Linear tasks одним вызовом `TaskManager.GetTasks`, затем проходит по ним в порядке ответа и синхронно вызывает route для proposal, Apply или Archive. Каждый route уже имеет отдельную границу ответственности: перевод задачи в in-progress state, запуск своего runner'а, финальный переход в review state и структурные логи.

Проблема в том, что runner'ы запускают долгие Codex/git/OpenSpec workflows. Пока proposal-задача выполняется, готовые Apply и Archive задачи из других колонок ждут завершения текущего route, хотя они могут выполняться независимо во временных клонах и разных ветках.

## Goals / Non-Goals

**Goals:**

- Запускать каждую eligible task из `Ready to Propose`, `Ready to Code` и `Ready to Archive` в отдельной goroutine внутри одного orchestration pass.
- Сохранять существующие route contracts, state transitions и runner interfaces.
- Дожидаться завершения всех goroutine перед завершением pass и перед следующим poll interval.
- Агрегировать ошибки всех неуспешных goroutine, чтобы один failure не скрывал остальные.
- Покрыть конкурентное поведение unit-тестами без реального Linear, git, GitHub CLI и Codex CLI.

**Non-Goals:**

- Не менять внутренний workflow proposal, Apply и Archive runner'ов.
- Не добавлять новый scheduler, worker pool, очередь задач или внешний concurrency dependency.
- Не вводить runtime-настройку лимита параллелизма в первом slice.
- Не пытаться автоматически отменять уже запущенные задачи при ошибке соседней задачи.

## Decisions

### D1. Одна goroutine на eligible task в pass

После `GetTasks` orchestration pass классифицирует каждую задачу по state ID и для proposal/Apply/Archive запускает соответствующий `process*Task` в отдельной goroutine. Неизвестные или неуправляемые states продолжают логироваться как skipped без goroutine.

Альтернатива: по одному worker'у на колонку. Отброшено, потому что это оставляет задачи внутри одной колонки последовательными без явной необходимости и добавляет stage-level coordination, которой сейчас нет в модели.

### D2. Join перед следующим poll

`RunProposalsOnce` должен завершаться только после окончания всех запущенных goroutine. `RunProposalsLoop` остается прежним: он запускает pass, логирует ошибку pass, ждет `PROPOSAL_POLL_INTERVAL`, затем начинает следующий pass.

Это предотвращает повторный захват тех же Linear-задач следующим poll, пока предыдущий pass еще работает.

### D3. Ошибки собираются через стандартную библиотеку

Каждая goroutine возвращает contextual error в общий collector под mutex. После `WaitGroup.Wait()` pass возвращает `errors.Join(collected...)`, если были ошибки. Новые внешние зависимости не нужны: `sync.WaitGroup`, `sync.Mutex` и `errors.Join` из стандартной библиотеки Go покрывают задачу.

Альтернатива: `errgroup.Group`. Отброшено, потому что он добавит зависимость ради небольшого wrapper'а, а отмена остальных задач при первой ошибке здесь нежелательна.

### D4. Failure одной задачи не отменяет соседние

Если одна goroutine завершилась ошибкой, остальные уже запущенные proposal/Apply/Archive задачи должны продолжать выполняться. Это соответствует цели: задачи из других колонок не блокируются из-за долгого или упавшего route.

Альтернатива: общий `context.WithCancel` при первой ошибке. Отброшено, потому что это снова связывает независимые задачи и может прерывать валидную работу в другой колонке.

### D5. Последовательность сохраняется внутри одной task

Конкурентность применяется между tasks. Внутри `processProposalTask`, `processApplyTask` и `processArchiveTask` порядок остается прежним: in-progress transition, runner execution, финальная task mutation. Это сохраняет текущую модель восстановления после ошибок и не меняет требования к runner'ам.

## Risks / Trade-offs

- [Больше одновременных Codex/git процессов] -> Принято как цель изменения; первый slice не добавляет лимит, чтобы не усложнять runtime до появления реальной нагрузки.
- [Конкурентные вызовы `TaskManager`] -> Реализация должна избегать shared mutable state в orchestration collector; fake-объекты в тестах нужно сделать concurrency-safe. Реальный TaskManager должен продолжать использовать независимые API calls.
- [Порядок логов и mutation calls станет недетерминированным] -> Тесты не должны полагаться на глобальный порядок между разными tasks; проверять нужно факт вызовов и per-task ordering.
- [Один pass может вернуть несколько ошибок] -> Использовать `errors.Join`, чтобы вызывающий monitor получил один error, а логи сохранили task-level контекст.

## Migration Plan

1. Обновить `internal/coreorch.Orchestrator.RunProposalsOnce`: запускать eligible tasks через goroutine, собирать counts и errors, ждать завершения всех задач.
2. Сохранить `processProposalTask`, `processApplyTask` и `processArchiveTask` как последовательные per-task workflow.
3. Обновить fake task manager/runner'ы в `internal/coreorch` tests для безопасной конкурентной записи.
4. Заменить тесты, которые проверяли глобальный последовательный порядок, на тесты параллельного запуска, ожидания завершения и агрегации ошибок.
5. Запустить `go fmt ./...`, `go test ./...`, `openspec status --change add-concurrent-stage-runs`.

Rollback: вернуть `RunProposalsOnce` к синхронному циклу по tasks. Runner'ы, конфигурация и Linear state model не требуют миграции данных.

## Open Questions

Открытых вопросов нет. Лимит параллелизма сознательно оставлен вне первого slice: его стоит добавлять только после появления измеримой нагрузки или ограничений Linear/Codex окружения.
