## Why

В проекте появились три stage runner-а (`proposalrunner`, `applyrunner`, `archiverunner`), но их структура выросла копированием: повторяются контракты agent executor-а, запуск Codex CLI, логирование команд, wiring `GitManager` и часть apply/archive workflow. Это уже мешает безопасно менять orchestration flow: любое исправление runner-инфраструктуры нужно дублировать в нескольких пакетах и проверять вручную.

## What Changes

- Перенести stage runner-ы в отдельную общую область модулей, чтобы все runner-related пакеты были сгруппированы рядом и имели понятные границы ответственности.
- Выделить переиспользуемые runner-компоненты: общий контракт agent execution, общий Codex CLI executor с stage-specific prompt/metadata, общий helper для logged command output и общий wiring clone/workspace/git dependencies.
- Сохранить публичное поведение proposal/apply/archive stage: proposal создает PR и комментирует финальный ответ агента, apply/archive пушат изменения в существующую ветку без создания нового PR.
- Убрать прямую зависимость apply/archive от `proposalrunner` для построения display name/commit message, вынеся общую metadata-логику в runner-common пакет.
- Обновить `architecture.md`, чтобы маппинг текущего кода отражал новую структуру runner-модулей.
- Не добавлять новые runtime-переменные, внешние зависимости или новый orchestration scheduler.

## Capabilities

### New Capabilities

- `runner-modules`: внутренняя архитектурная capability для размещения, общих компонентов и границ ответственности proposal/apply/archive runner-ов.

### Modified Capabilities

- `codex-proposal-pr-runner`: требования к proposal runner должны сохраниться после переноса в новую структуру и выделения общих runner-компонентов.

## Impact

- Затронутые пакеты: `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`, `internal/commandrunner`, возможно новые подпакеты под общей runner-директорией.
- Затронутые тесты: unit-тесты runner-ов, Codex executor-ов, command/logging helpers и интеграционный контракт `CoreOrch -> runner`.
- Затронута документация: `architecture.md`.
- Внешнее поведение CLI, Linear state flow, GitHub PR flow и `.env`-конфигурация не должны измениться.
