## Context

TaskManager уже централизованно загружает Linear runtime-настройки и умеет переводить задачу в произвольный workflow state ID. Proposal orchestration сейчас фильтрует `Ready to Propose`, запускает proposal runner, прикрепляет PR и только после этого двигает задачу в `Need Proposal Review`. Из-за этого во время долгого Proposing задача остается в исходной колонке и может выглядеть свободной для повторной обработки.

Запрос добавляет один используемый сейчас in-progress статус (`Proposing in Progress`) и два конфигурационных статуса на будущие стадии (`Code in Progress`, `Archiving in Progress`). Реализация должна оставаться простой: новые ENV читаются тем же механизмом, а переход перед Proposing делается через существующий `TaskManager.MoveTask`.

## Goals / Non-Goals

**Goals:**

- Добавить в конфигурацию обязательные Linear state IDs для `Proposing in Progress`, `Code in Progress`, `Archiving in Progress`.
- Синхронизировать `.env.example` с новыми ключами без значений.
- Перед запуском proposal runner переводить ready-to-propose задачу в `Proposing in Progress`.
- После успешного runner-а сохранить текущий порядок: прикрепить PR, затем перевести задачу в `Need Proposal Review`.
- Покрыть новую конфигурацию и порядок переходов unit-тестами.

**Non-Goals:**

- Не реализовывать code-stage и archive-stage orchestration.
- Не менять Linear API-клиент: он уже принимает target state ID через `MoveTask`.
- Не добавлять locking, lease, retry policy или идемпотентность proposal runner-а.
- Не менять OpenSpec proposal runner workflow, git/PR поведение и CLI single-run режим.

## Decisions

### Использовать отдельные ENV-переменные для in-progress статусов

Добавить ключи:

- `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`
- `LINEAR_STATE_CODE_IN_PROGRESS_ID`
- `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`

Они будут полями `LinearTaskManagerConfig` и частью `Validate`. Это делает отсутствие настроенных колонок явной ошибкой запуска, как и для текущих managed/review states.

Альтернатива: сделать Code/Archiving in-progress ключи необязательными до появления соответствующих стадий. Это уменьшает миграционную нагрузку сейчас, но противоречит задаче “добавить эти строки в ENV” и оставляет конфигурацию будущих колонок расхожей с кодом.

### Не добавлять in-progress states в managed selection

`ManagedStateIDs()` должен по-прежнему возвращать только состояния, из которых оркестратор берет задачи в работу: ready-to-propose, ready-to-code, ready-to-archive. In-progress колонки являются target states, а не входными очередями для нового запуска.

Альтернатива: включить in-progress states в выборку Linear. Это может привести к повторной обработке уже взятых задач без отдельной логики восстановления.

### Двигать задачу в Proposing in Progress до запуска runner-а

Для каждой ready-to-propose задачи порядок станет таким:

1. `MoveTask(task.ID, ProposingInProgressStateID)`.
2. `ProposalRunner.Run(ctx, BuildProposalInput(task))`.
3. `AddPR(task.ID, prURL)`.
4. `MoveTask(task.ID, NeedProposalReviewStateID)`.

Если первый переход не удался, runner не запускается. Если runner или PR attachment падает после успешного первого перехода, задача остается в `Proposing in Progress`; ошибка возвращается с task context.

Альтернатива: запускать runner до первого transition и двигать в progress только после успешного старта subprocess. В текущем интерфейсе `ProposalRunner.Run` синхронный, поэтому отдельного “после старта” хука нет; transition до вызова runner-а лучше отражает занятость задачи.

## Risks / Trade-offs

- Новые обязательные ENV могут сломать локальный запуск до обновления `.env` -> `.env.example` фиксирует полный набор ключей, а ошибка валидации указывает конкретное имя переменной.
- Runner failure оставит задачу в `Proposing in Progress` -> это осознанный сигнал частичного сбоя; автоматический rollback в `Ready to Propose` не вводится, чтобы не скрыть неизвестный результат долгого внешнего процесса.
- Code/Archiving in-progress ключи пока не используются -> это небольшая конфигурационная подготовка под будущие стадии без новых runtime-путей.
- Повторный запуск не подбирает in-progress задачи -> это предотвращает дублирование работы; восстановление зависших задач остается ручным до отдельной задачи.

## Migration Plan

1. Добавить новые ключи в `.env.example` и локальные `.env` при настройке окружения.
2. Расширить `LinearTaskManagerConfig`, `Load`, `Validate` и config-тесты.
3. Расширить `coreorch.Config`, validation и wiring в CLI.
4. Обновить `RunProposalsOnce`/`processTask`, чтобы делать первый transition перед runner.
5. Обновить unit-тесты на порядок вызовов и failure cases.
6. Запустить `go fmt ./...`, `go test ./...`, `openspec validate update-taskmanager-progress-columns --strict`.
