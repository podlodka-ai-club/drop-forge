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
6. После изменений `GitManager` оформляет branch, commit, push и PR.
7. `AgentExecutor` возвращает `CoreOrch` результат выполнения, включая `PR URL`.
8. `CoreOrch` просит `TaskManager` обновить статус задачи и прикрепить ссылку на PR.
9. Если человек отклоняет proposal, задача возвращается в `Ready to Propose` и цикл повторяется.
10. Если человек принимает proposal, задача переводится в `Ready to Code`.

## Границы Ответственности

- `CoreOrch` координирует сценарий, но не должен содержать детали `git`, `gh`, `codex` или API task tracker-а.
- `TaskManager` отвечает за поиск задач, смену статусов, комментарии и привязку артефактов задачи.
- `AgentExecutor` отвечает за lifecycle агентного запуска: подготовка input, запуск, сбор результата, публикация логов, возврат статуса.
- `GitManager` отвечает за операции с repository/workspace: `clone`, ветки, commit, push, PR.
- `Logger` отвечает за единый формат структурных событий и за доставку логов в sink-и.

## Маппинг На Текущий Код

- При создании, выделении или существенном изменении сервисов агент обязан обновлять эту секцию, чтобы статус реализации и маппинг на код оставались актуальными.
- `Logger` уже реализован в `internal/steplog`. Это текущий готовый сервис с явным контрактом JSON Lines.
- `AgentExecutor` частично реализован в `internal/proposalrunner`: именно этот модуль сейчас запускает `codex`, собирает последний ответ и управляет шагами workflow.
- `GitManager` пока не выделен в отдельный пакет, но его ответственность уже фактически присутствует внутри `internal/proposalrunner` через команды `git clone`, `checkout -b`, `add`, `commit`, `push` и `gh pr create`.
- `CoreOrch` для proposal-stage реализован в `internal/coreorch`: он получает managed tasks через контракт `TaskManager`, фильтрует задачи по `ReadyToProposeStateID`, последовательно запускает `ProposalRunner`, прикрепляет PR URL и переводит задачу в `NeedProposalReviewStateID`.
- `cmd/orchv3/main.go` поддерживает два proposal-сценария: прямой single-run запуск `proposalrunner.Run` по task description из args/stdin и явный режим `orchestrate-proposals` для одного прохода `CoreOrch`.
- `TaskManager` реализован в `internal/taskmanager`: сервис читает managed Linear tasks, возвращает внутреннюю модель задачи с идентификаторами, описанием, состоянием и комментариями, а также выполняет `AddPR` и `MoveTask`.
- `internal/commandrunner` — это не отдельный доменный актор, а технический адаптер для запуска внешних команд, который уже переиспользуется `AgentExecutor`/будущим `GitManager`.

## Текущее Архитектурное Чтение Репозитория

- Сегодня проект покрывает вертикальный slice `CoreOrch -> TaskManager -> AgentExecutor -> GitManager -> Logger` для одного proposal orchestration pass без постоянного polling loop.
- Следующий естественный шаг роста — отделить из `internal/proposalrunner` самостоятельные `GitManager` и `AgentExecutor`, а затем добавить долгоживущий scheduler/polling loop поверх `CoreOrch`.
- До появления реальной потребности не выделять новые сервисы сверх этих пяти ролей.
