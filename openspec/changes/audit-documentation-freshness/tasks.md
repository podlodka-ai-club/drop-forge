## 1. Подготовка аудита

- [x] 1.1 Составить список проверяемых документов: `README.md`, `docs/*.md`, `architecture.md`, `.env.example`, `openspec/specs/**/*.md`.
- [x] 1.2 Сверить текущие CLI-сценарии по `cmd/orchv3` и пакетам `internal/coreorch`, `internal/proposalrunner`, `internal/taskmanager`.
- [x] 1.3 Сверить поддерживаемые runtime-переменные по `internal/config` и `.env.example`.

## 2. Проверка и правки документации

- [x] 2.1 Проверить `README.md` на соответствие текущим CLI-режимам, зависимостям, stdout/stderr contract и ссылкам на подробные документы.
- [x] 2.2 Проверить `docs/proposal-runner.md` на соответствие текущему `proposalrunner.Runner`, `ProposalInput`, `AgentExecutor` и GitHub PR workflow.
- [x] 2.3 Проверить `docs/linear-task-manager.md` на соответствие текущему `TaskManager`, managed state IDs и review-state responsibility.
- [x] 2.4 Проверить `docs/elk-demo.md` и deploy-файлы на актуальность инструкций запуска, health-check и smoke-test.
- [x] 2.5 Проверить `architecture.md` на соответствие текущим границам `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, `Logger`.
- [x] 2.6 Проверить активные `openspec/specs/**/*.md` на явные расхождения с текущей реализацией и документацией.
- [x] 2.7 Внести минимальные исправления в документы, где найдены устаревшие, неполные или вводящие в заблуждение утверждения.

## 3. Валидация

- [x] 3.1 Убедиться, что `.env.example` содержит только поддерживаемые ключи без секретов и значений по умолчанию.
- [x] 3.2 Запустить `openspec validate audit-documentation-freshness --strict`.
- [x] 3.3 Запустить `go fmt ./...`.
- [x] 3.4 Запустить `go test ./...`.
- [x] 3.5 Зафиксировать в итоговом отчете, какие документы проверены, какие расхождения исправлены и какие проверки выполнены.

## Отчет аудита

- Проверены `README.md`, `docs/proposal-runner.md`, `docs/linear-task-manager.md`, `docs/elk-demo.md`, `architecture.md`, `.env.example`, active specs в `openspec/specs/**/*.md`, а также соответствующие участки `cmd/orchv3`, `internal/config`, `internal/coreorch`, `internal/proposalrunner`, `internal/taskmanager`, `internal/steplog` и `deploy`.
- Исправлены устаревшие формулировки про proposal orchestration: README, Linear-док и architecture теперь явно отражают переход в `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID` перед запуском runner и последующий переход в `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.
- Исправлена active spec `codex-proposal-pr-runner`: `ProposalInput.AgentPrompt` передается в `AgentExecutor`, а default `CodexCLIExecutor` добавляет инструкцию `openspec-propose` при построении Codex prompt.
- `.env.example` сверена с `internal/config`: поддерживаемые ключи присутствуют, значений по умолчанию и секретов в шаблоне нет.
- Выполнены проверки: `openspec validate audit-documentation-freshness --strict`, `go fmt ./...`, `go test ./...`.
