# Architecture

## Назначение

Этот документ фиксирует целевую архитектурную рамку оркестратора и текущее состояние реализации.
Его нужно обновлять при нетривиальных изменениях взаимодействия компонентов, появлении новых внутренних сервисов или изменении границ ответственности.

## Внутренние Акторы

- Внутренними акторами проекта считаем только `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, `Logger`.
- `TaskManager` изолирует проект от конкретного task backend. Сегодня это может быть `Linear`, но код проекта должен зависеть от своего контракта, а не от SDK конкретной системы.
- `AgentExecutor` изолирует проект от конкретного coding-agent runtime. Сегодня это `CodexCLI`, но orchestration-слой не должен быть жестко привязан к нему.

## Целевой Поток Proposal-Stage

1. `CoreOrch` регулярно запрашивает у `TaskManager` задачи, готовые к proposal-обработке.
2. `TaskManager` читает задачи из внешнего tracker-а и возвращает внутреннюю модель задач.
3. `CoreOrch` передает задачу в `AgentExecutor`.
4. `AgentExecutor` через `GitManager` поднимает isolated workspace, запускает агент и получает результат.
5. Во время выполнения `AgentExecutor` пишет шаги и stdout/stderr событий в `Logger`.
6. После изменений `GitManager` оформляет branch, commit, push и provider-specific review request: GitHub PR или GitLab MR.
7. `AgentExecutor` возвращает `CoreOrch` результат выполнения, включая review request URL.
8. `CoreOrch` просит `TaskManager` обновить статус задачи и прикрепить ссылку на review request.
9. Если человек отклоняет proposal, задача возвращается в `Ready to Propose` и цикл повторяется.
10. Если человек принимает proposal, задача переводится в `Ready to Code`.

## Целевой Поток Apply-Stage

1. `CoreOrch` в том же проходе monitor-а получает managed tasks от `TaskManager`.
2. Задачи в `Ready to Code` маршрутизируются в Apply-stage.
3. `TaskManager` возвращает вместе с задачей источник ветки: review request URL, branch name или оба значения.
4. `CoreOrch` переводит задачу в `Code in Progress` до запуска executor-а.
5. `ApplyRunner` через `GitManager` клонирует репозиторий во временную директорию, определяет ветку из branch name или review request URL, переключается на нее и запускает Codex с OpenSpec Apply-инструкцией.
6. Если агент создал изменения, `GitManager` выполняет `git add`, `commit` и `push` в ту же ветку без создания нового PR.
7. После успешного push `CoreOrch` переводит задачу в `Need Code Review`.

## Целевой Поток Archive-Stage

1. `CoreOrch` в том же проходе monitor-а получает managed tasks от `TaskManager`.
2. Задачи в `Ready to Archive` маршрутизируются в Archive-stage.
3. `TaskManager` возвращает вместе с задачей источник ветки: review request URL, branch name или оба значения.
4. `CoreOrch` переводит задачу в `Archiving in Progress` до запуска executor-а.
5. `ArchiveRunner` через `GitManager` клонирует репозиторий во временную директорию, определяет ветку из branch name или review request URL, переключается на нее и запускает Codex с OpenSpec Archive-инструкцией.
6. Если агент создал archive-изменения, `GitManager` выполняет `git add`, `commit` и `push` в ту же ветку без создания нового PR.
7. После успешного push `CoreOrch` переводит задачу в `Need Archive Review`.

## Границы Ответственности

- `CoreOrch` координирует сценарий, но не должен содержать детали `git`, `gh`, `glab`, `codex` или API task tracker-а.
- `TaskManager` отвечает за поиск задач, смену статусов, комментарии и привязку артефактов задачи.
- `AgentExecutor` отвечает за lifecycle агентного запуска: подготовка input, запуск, сбор результата, публикация логов, возврат статуса.
- `GitManager` отвечает за операции с repository/workspace: `clone`, ветки, commit, push и provider-specific review request operations.
- `Logger` отвечает за единый формат структурных событий и за доставку логов в sink-и.

## Маппинг На Текущий Код

- При создании, выделении или существенном изменении сервисов агент обязан обновлять эту секцию, чтобы статус реализации и маппинг на код оставались актуальными.
- `Logger` уже реализован в `internal/steplog`. Это текущий готовый сервис с явным контрактом JSON Lines.
- `AgentExecutor` реализован как явный контракт внутри `internal/proposalrunner`, `internal/applyrunner` и `internal/archiverunner`. Текущие реализации `CodexCLIExecutor` изолируют протокол `codex exec` и stage-specific prompt.
- `GitManager` реализован в `internal/gitmanager`: он управляет isolated clone workspace, cleanup, `git status/checkout/add/commit/push` и provider-specific CLI операциями. В GitHub-режиме он вызывает `gh pr view/create/comment`; в GitLab-режиме `glab mr view/create` и `glab mr note create`. `internal/proposalrunner`, `internal/applyrunner` и `internal/archiverunner` используют его через узкие интерфейсы, сохраняя stage-specific построение prompt, branch name, commit message и review request metadata внутри runner-пакетов.
- `CoreOrch` реализован в `internal/coreorch`: он получает managed tasks через контракт `TaskManager`, последовательно маршрутизирует `ReadyToProposeStateID` в `ProposalRunner`, `ReadyToCodeStateID` в `ApplyRunner`, а `ReadyToArchiveStateID` в `ArchiveRunner`. Proposal-route прикрепляет review request URL и переводит задачу в `NeedProposalReviewStateID`; Apply-route переводит задачу через `CodeInProgressStateID` в `NeedCodeReviewStateID`; Archive-route переводит задачу через `ArchivingInProgressStateID` в `NeedArchiveReviewStateID`.
- `cmd/orchv3/main.go` запускает orchestration monitor как default runtime без аргументов CLI. Прямой single-run запуск `proposalrunner.Run` по task description из args/stdin удален; непустые args/stdin считаются unsupported manual input.
- `TaskManager` реализован в `internal/taskmanager`: сервис читает managed Linear tasks, возвращает внутреннюю модель задачи с идентификаторами, описанием, состоянием, комментариями и PR attachment URL, а также выполняет `AddPR` и `MoveTask`.
- `internal/commandrunner` — это не отдельный доменный актор, а технический адаптер для запуска внешних команд, который переиспользуется `AgentExecutor` и `GitManager`.

## Текущее Архитектурное Чтение Репозитория

- Сегодня проект покрывает proposal, apply и archive slice `CoreOrch -> TaskManager -> AgentExecutor -> GitManager -> Logger` и запускает его из CLI как долгоживущий orchestration monitor с конфигурируемым polling interval.
- `AgentExecutor` и `GitManager` уже выделены как внутренние границы: stage-specific agent implementations находятся в runner-пакетах, а repository lifecycle сосредоточен в `internal/gitmanager`.
- Следующий естественный шаг роста стоит выбирать по фактической боли в orchestration flow; более сложный scheduler/backoff стоит добавлять только при подтвержденной необходимости.
- До появления реальной потребности не выделять новые сервисы сверх этих пяти ролей.
