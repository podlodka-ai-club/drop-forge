# Architecture

## Назначение

Этот документ фиксирует целевую архитектурную рамку оркестратора и текущее состояние реализации.
Его нужно обновлять при нетривиальных изменениях взаимодействия компонентов, появлении новых внутренних сервисов или изменении границ ответственности.

## Внутренние Акторы

- Внутренними акторами проекта считаем `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, `EventDispatcher`, `NotificationSubscriber`, `Logger`.
- `TaskManager` изолирует проект от конкретного task backend. Сегодня это может быть `Linear`, но код проекта должен зависеть от своего контракта, а не от SDK конкретной системы.
- `AgentExecutor` изолирует проект от конкретного coding-agent runtime. Сегодня это `CodexCLI`, но orchestration-слой не должен быть жестко привязан к нему.
- `EventDispatcher` дает локальную точку расширения для доменных событий без внешнего брокера.
- `NotificationSubscriber` реагирует на доменные события и выполняет best-effort доставку уведомлений во внешние каналы.

## Целевой Поток Proposal-Stage

1. `CoreOrch` регулярно запрашивает у `TaskManager` задачи, готовые к proposal-обработке.
2. `TaskManager` читает задачи из внешнего tracker-а и возвращает внутреннюю модель задач.
3. `CoreOrch` передает задачу в `AgentExecutor`.
4. `AgentExecutor` через `GitManager` поднимает isolated workspace, запускает агент и получает результат.
5. Во время выполнения `AgentExecutor` пишет шаги и stdout/stderr событий в `Logger`.
6. После изменений `GitManager` оформляет branch, commit, push и PR.
7. `AgentExecutor` возвращает `CoreOrch` результат выполнения, включая `PR URL`.
8. `CoreOrch` просит `TaskManager` обновить статус задачи и прикрепить ссылку на PR.
9. После успешного обновления статуса `TaskManager` публикует событие `task.status_changed` в локальный `EventDispatcher`; для review-переходов `CoreOrch` передает в `TaskManager` snapshot задачи и PR-контекст для downstream уведомлений.
10. Если включены Telegram-уведомления, Telegram subscriber получает событие и отправляет best-effort сообщение в чат только для configured review state IDs; ошибка уведомления логируется, но не отменяет смену статуса.
11. Если человек отклоняет proposal, задача возвращается в `Ready to Propose` и цикл повторяется.
12. Если человек принимает proposal, задача переводится в `Ready to Code`.

## Целевой Поток Apply-Stage

1. `CoreOrch` в том же проходе monitor-а получает managed tasks от `TaskManager`.
2. Задачи в `Ready to Code` маршрутизируются в Apply-stage.
3. `TaskManager` возвращает вместе с задачей источник ветки: PR URL, branch name или оба значения.
4. `CoreOrch` переводит задачу в `Code in Progress` до запуска executor-а.
5. `ApplyRunner` через `GitManager` клонирует репозиторий во временную директорию, определяет ветку из branch name или PR URL, переключается на нее и запускает Codex с OpenSpec Apply-инструкцией.
6. Если агент создал изменения, `GitManager` выполняет `git add`, `commit` и `push` в ту же ветку без создания нового PR.
7. После успешного push `CoreOrch` переводит задачу в `Need Code Review`, передавая task identifier/title и PR URL или branch в контекст события.
8. Успешные переходы статусов публикуют `task.status_changed`; подключенные подписчики обрабатывают событие синхронно и best-effort.

## Целевой Поток Archive-Stage

1. `CoreOrch` в том же проходе monitor-а получает managed tasks от `TaskManager`.
2. Задачи в `Ready to Archive` маршрутизируются в Archive-stage.
3. `TaskManager` возвращает вместе с задачей источник ветки: PR URL, branch name или оба значения.
4. `CoreOrch` переводит задачу в `Archiving in Progress` до запуска executor-а.
5. `ArchiveRunner` через `GitManager` клонирует репозиторий во временную директорию, определяет ветку из branch name или PR URL, переключается на нее и запускает Codex с OpenSpec Archive-инструкцией.
6. Если агент создал archive-изменения, `GitManager` выполняет `git add`, `commit` и `push` в ту же ветку без создания нового PR.
7. После успешного push `CoreOrch` переводит задачу в `Need Archive Review`, передавая task identifier/title и PR URL или branch в контекст события.
8. Успешные переходы статусов публикуют `task.status_changed`; подключенные подписчики обрабатывают событие синхронно и best-effort.

## Целевой Поток Review-Stage

1. `CoreOrch` в том же проходе monitor-а получает managed tasks от `TaskManager`.
2. Задачи в `Need * AI Review` (Proposal/Code/Archive) маршрутизируются в Review-stage соответствующего этапа.
3. `ReviewRunner` клонирует ветку задачи во временную директорию, читает producer-trailer последнего HEAD-коммита и выбирает reviewer-слот, противоположный продьюсеру.
4. Reviewer-executor запускается с stage-specific prompt'ом и возвращает строго JSON по схеме review-ответа; при невалидном JSON выполняется один repair-retry.
5. Распарсенный review публикуется одним атомарным POST'ом через GitHub Pull Request Reviews API: summary в body PR review плюс inline-комментарии на каждую находку с собственным fix-prompt'ом.
6. Идемпотентность по HTML-маркеру `(reviewer, stage, HEAD-sha)`: повторный запуск review на том же коммите пропускает публикацию и сразу переходит к смене статуса.
7. После успешной публикации (или idempotent skip) `CoreOrch` переводит задачу в человеческий review-state соответствующей стадии.
8. При сбое публикации, невалидном JSON после repair или config-mismatch reviewer-слота задача остаётся в AI-review state, monitor подхватит её следующим тиком.

## Границы Ответственности

- `CoreOrch` координирует сценарий, но не должен содержать детали `git`, `gh`, `codex` или API task tracker-а.
- `TaskManager` отвечает за поиск задач, смену статусов, комментарии и привязку артефактов задачи.
- `AgentExecutor` отвечает за lifecycle агентного запуска: подготовка input, запуск, сбор результата, публикация логов, возврат статуса.
- `GitManager` отвечает за операции с repository/workspace: `clone`, ветки, commit, push, PR.
- `EventDispatcher` отвечает за регистрацию подписчиков по типу события и синхронную публикацию внутренних доменных событий.
- `NotificationSubscriber` отвечает за доставку уведомлений по событиям; сегодня это Telegram subscriber для `task.status_changed`.
- `Logger` отвечает за единый формат структурных событий и за доставку логов в sink-и.

## Маппинг На Текущий Код

- При создании, выделении или существенном изменении сервисов агент обязан обновлять эту секцию, чтобы статус реализации и маппинг на код оставались актуальными.
- `Logger` уже реализован в `internal/steplog`. Это текущий готовый сервис с явным контрактом JSON Lines.
- `EventDispatcher` реализован в `internal/events`: пакет определяет `Event`, `Publisher`, `Handler`, payload `TaskStatusChanged` и локальный синхронный dispatcher.
- `NotificationSubscriber` для Telegram реализован в `internal/notifications/telegram`: подписчик обрабатывает `task.status_changed`, фильтрует события по review state IDs из существующей Linear-конфигурации и отправляет `sendMessage` через стандартный `net/http`.
- `AgentExecutor` реализован как явный контракт внутри `internal/proposalrunner`, `internal/applyrunner` и `internal/archiverunner`. Текущие реализации `CodexCLIExecutor` изолируют протокол `codex exec` и stage-specific prompt.
- `GitManager` реализован в `internal/gitmanager`: он управляет isolated clone workspace, cleanup, `git status/checkout/add/commit/push` и GitHub CLI операциями `gh pr view/create/comment`. `internal/proposalrunner`, `internal/applyrunner` и `internal/archiverunner` используют его через узкие интерфейсы, сохраняя stage-specific построение prompt, branch name, commit message и PR metadata внутри runner-пакетов.
- `CoreOrch` реализован в `internal/coreorch`: он получает managed tasks через контракт `TaskManager`, последовательно маршрутизирует `ReadyToProposeStateID` в `ProposalRunner`, `ReadyToCodeStateID` в `ApplyRunner`, а `ReadyToArchiveStateID` в `ArchiveRunner`. Proposal-route прикрепляет PR URL и переводит задачу в `NeedProposalReviewStateID`; Apply-route переводит задачу через `CodeInProgressStateID` в `NeedCodeReviewStateID`; Archive-route переводит задачу через `ArchivingInProgressStateID` в `NeedArchiveReviewStateID`. Review-переходы используют расширенный move с task identifier/title, target state name и PR URL/branch.
- `cmd/orchv3/main.go` запускает orchestration monitor как default runtime без аргументов CLI. При старте он создает локальный dispatcher, регистрирует Telegram subscriber только при `TELEGRAM_NOTIFICATIONS_ENABLED=true` и передает publisher в `TaskManager`. Прямой single-run запуск `proposalrunner.Run` по task description из args/stdin удален; непустые args/stdin считаются unsupported manual input.
- `TaskManager` реализован в `internal/taskmanager`: сервис читает managed Linear tasks, возвращает внутреннюю модель задачи с идентификаторами, описанием, состоянием, комментариями и PR attachment URL/branch, а также выполняет `AddPR`, `MoveTask` и расширенный move с context snapshot. После успешного move он публикует `task.status_changed`; ошибка публикации логируется и не меняет результат уже выполненного перехода в Linear.
- `internal/commandrunner` — это не отдельный доменный актор, а технический адаптер для запуска внешних команд, который переиспользуется `AgentExecutor` и `GitManager`.
- `ReviewRunner` реализован в `internal/reviewrunner` как четвёртая stage-агностичная реализация над `AgentExecutor`-контрактом. Инкапсулирует строгий JSON-парсер review-ответа (`internal/reviewrunner/reviewparse`), формирование PR review (`internal/reviewrunner/prcommenter`) и stage-specific prompt-templates (`internal/reviewrunner/prompts/*.tmpl`). Активируется feature-flag'ом через тройку `LINEAR_STATE_NEED_*_AI_REVIEW_ID` плюс пары reviewer-слотов `REVIEW_ROLE_PRIMARY` / `REVIEW_ROLE_SECONDARY`.
- `internal/agentmeta` хранит контракт producer-trailer'а в commit message: `Produced-By`, `Produced-Model`, `Produced-Stage`. Используется тремя producer-runner'ами при коммите и `ReviewRunner` при чтении HEAD.

## Текущее Архитектурное Чтение Репозитория

- Сегодня проект покрывает proposal, apply и archive slice `CoreOrch -> TaskManager -> AgentExecutor -> GitManager -> Logger` и запускает его из CLI как долгоживущий orchestration monitor с конфигурируемым polling interval. Смена статуса дополнительно проходит через локальный `EventDispatcher`, чтобы подписчики могли реагировать без прямой связи с `TaskManager`.
- `AgentExecutor` и `GitManager` уже выделены как внутренние границы: stage-specific agent implementations находятся в runner-пакетах, а repository lifecycle сосредоточен в `internal/gitmanager`.
- Следующий естественный шаг роста стоит выбирать по фактической боли в orchestration flow; более сложный scheduler/backoff стоит добавлять только при подтвержденной необходимости.
- До появления реальной потребности не добавлять внешний брокер событий, durable delivery или runtime-шаблоны уведомлений.
