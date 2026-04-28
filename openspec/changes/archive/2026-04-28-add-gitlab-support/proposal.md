## Why

Сейчас оркестратор умеет создавать pull request, комментировать его и резолвить branch source только через GitHub CLI `gh`. Это блокирует использование того же workflow для репозиториев в GitLab, хотя основные стадии proposal/apply/archive не зависят от GitHub как продукта.

## What Changes

- Добавить runtime-настройку Git provider-а для выбора между GitHub и GitLab без изменения orchestration flow.
- Расширить `GitManager`, чтобы он выполнял операции pull/merge request через provider-specific CLI: текущий `gh` для GitHub и `glab` для GitLab.
- Поддержать создание GitLab merge request после proposal runner, публикацию финального комментария и резолв ветки по MR URL для Apply/Archive.
- Сохранить текущие GitHub-настройки и поведение как совместимый default.
- Обновить `.env.example`, конфигурацию, README/docs и тесты для GitLab-настроек.

## Capabilities

### New Capabilities
- `git-provider`: выбор и использование GitHub/GitLab provider-а для pull/merge request операций.

### Modified Capabilities
- `git-manager`: `GitManager` должен поддерживать provider-specific PR/MR операции, а не только GitHub CLI.
- `codex-proposal-pr-runner`: proposal runner должен создавать review request через выбранный provider и возвращать URL.
- `proposal-orchestration`: Apply/Archive должны использовать выбранный provider при резолве ветки из URL review request-а.
- `project-readme`: документация должна описывать GitHub и GitLab prerequisites/configuration.

## Impact

- Код: `internal/config`, `internal/gitmanager`, `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`, CLI wiring/tests.
- Конфигурация: новая переменная выбора provider-а и путь к GitLab CLI; существующие `PROPOSAL_*` переменные остаются совместимыми.
- Внешние инструменты: для GitLab-режима нужен установленный и аутентифицированный `glab`.
- Документация: `.env.example`, README и профильные docs должны отражать оба provider-а.
