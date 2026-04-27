## Context

Текущий `orchv3` уже содержит вертикальный proposal-stage: `cmd/orchv3` умеет запускать прямой `proposalrunner.Run` по тексту из args/stdin и отдельный режим `orchestrate-proposals`, который делает один проход через `CoreOrch`. `CoreOrch` получает managed Linear tasks, фильтрует `Ready to propose`, запускает proposal runner, прикрепляет PR и переводит задачу в `Need Proposal Review`.

Задача DRO-31 меняет модель использования: оператор больше не должен вручную передавать текст задачи в CLI или запускать первую тестовую команду. Приложение должно работать как долгоживущий процесс, который сам мониторит Linear-столбец `Ready to propose` и повторяет proposal orchestration.

## Goals / Non-Goals

**Goals:**

- Сделать основной CLI-запуск долгоживущим proposal polling loop.
- Убрать публичный прямой запуск proposal runner по args/stdin.
- Убрать тестовый one-shot CLI command `orchestrate-proposals` как пользовательский режим.
- Сохранить существующий единичный проход `RunProposalsOnce` как внутренний тестируемый building block, если он удобен для реализации цикла.
- Добавить конфигурируемый polling interval через `.env` и централизованный loader.
- Корректно завершать цикл по отмене контекста/сигналу процесса.
- Обновить README и `architecture.md`, потому что меняется публичный flow запуска и взаимодействие entrypoint -> `CoreOrch`.

**Non-Goals:**

- Не добавлять обработку `Ready to code` или `Ready to archive`.
- Не менять внутренний workflow `proposalrunner`: clone, Codex/OpenSpec propose, commit, push, PR и comment остаются тем же executor behavior.
- Не добавлять новый task backend вместо Linear.
- Не выделять отдельный scheduler/service сверх существующего `CoreOrch`, если простой loop в текущих границах решает задачу.
- Не добавлять параллельную обработку задач; текущая последовательная обработка остается безопаснее для первого долгоживущего режима.

## Decisions

### 1. Основной запуск CLI стартует polling loop без дополнительных команд

`orchv3` без task description должен загружать конфиг, собирать `TaskManager`, `ProposalRunner`, `CoreOrch` и входить в бесконечный цикл. Это убирает необходимость помнить отдельную тестовую команду и делает production-поведение дефолтным.

Альтернатива: оставить `orchestrate-proposals` и сделать его бесконечным. Это сохраняет лишний ручной режим и хуже соответствует требованию убрать запуск через CLI первой тестовой команды.

### 2. Args/stdin больше не являются task input

Если пользователь передает произвольные args или pipe в stdin, CLI должен завершаться с ошибкой использования и не запускать proposal runner напрямую. Это явно ломает старый ручной путь, но предотвращает обход Linear-state workflow.

Альтернатива: молча игнорировать args/stdin и стартовать loop. Это рискованно: оператор может думать, что запустил конкретную задачу, а процесс начнет обрабатывать очередь Linear.

### 3. `RunProposalsOnce` остается внутренней операцией, над ней добавляется loop

В `internal/coreorch` стоит добавить метод уровня `RunProposalLoop(ctx)` или отдельную функцию, которая вызывает существующий один проход, затем ждет interval. `RunProposalsOnce` остается полезным для unit-тестов и для ограничения blast radius: логика выбора и обработки задач уже покрыта.

Альтернатива: встроить бесконечный цикл прямо в текущий `RunProposalsOnce`. Это ухудшит тестируемость и смешает одну итерацию с lifecycle процесса.

### 4. Polling interval настраивается через `.env`

Добавить runtime-параметр, например `PROPOSAL_POLL_INTERVAL`, который парсится как `time.Duration`. В `.env.example` хранится только ключ без значения. В коде допустим безопасный default, чтобы локальный запуск не требовал дополнительной переменной, если текущие правила config loader позволяют default'ы для несекретных параметров.

Альтернатива: захардкодить interval в loop. Это проще, но нарушает проектное правило выносить runtime-параметры в `.env` и усложняет настройку частоты опроса без пересборки.

### 5. Ошибка одной итерации не должна автоматически убивать долгоживущий процесс

Loop должен логировать ошибку итерации и продолжать после interval, кроме случаев отмены контекста или ошибки стартовой конфигурации. Это важнее для мониторинга очереди: временная ошибка Linear/GitHub/Codex не должна требовать ручного рестарта.

Альтернатива: завершать процесс при первой ошибке обработки. Это проще и ближе к текущему one-shot поведению, но плохо подходит для постоянного мониторинга.

## Risks / Trade-offs

- [Breaking CLI behavior] -> Mitigation: явно обновить README/docs и добавить тесты, что args/stdin больше не запускают proposal runner.
- [Бесконечные тесты или зависающие процессы] -> Mitigation: loop принимает `context.Context`, использует injectable ticker/sleeper или короткий тестовый interval и покрывается тестами отмены.
- [Шумные повторы после постоянной ошибки] -> Mitigation: логировать structured event на каждую failed iteration; throttling/backoff оставить follow-up, если появится реальная проблема.
- [Неверный polling interval] -> Mitigation: config loader валидирует duration и возвращает ошибку до старта loop.
- [Архитектурный drift] -> Mitigation: обновить `architecture.md`, где уже описан целевой регулярный запрос `CoreOrch` к `TaskManager`.

## Migration Plan

1. Добавить config поле polling interval, загрузку из env и `.env.example`.
2. Добавить loop API в `internal/coreorch`, переиспользующий текущий `RunProposalsOnce`.
3. Переподключить `cmd/orchv3`: default startup запускает loop; direct args/stdin и `orchestrate-proposals` больше не запускают обработку.
4. Обновить README/docs и `architecture.md`.
5. Добавить/обновить тесты и прогнать `go fmt ./...`, `go test ./...`.

Rollback: вернуть предыдущий CLI branching на direct proposal runner и one-shot `orchestrate-proposals`, оставив новые config поля неиспользуемыми до удаления отдельным change.

## Open Questions

- Нужен ли отдельный exit code для usage error при args/stdin, или достаточно текущего общего `1`?
  - Why it matters: влияет на shell automation и тесты CLI.
  - Recommended option: использовать `1`, потому что в проекте уже применяется простой non-zero статус без отдельной taxonomy.
