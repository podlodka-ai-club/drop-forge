## 1. Подготовка аудита

- [ ] 1.1 Составить список проверяемых документов: `README.md`, `docs/*.md`, `architecture.md`, `.env.example`, `openspec/specs/**/*.md`.
- [ ] 1.2 Сверить текущие CLI-сценарии по `cmd/orchv3` и пакетам `internal/coreorch`, `internal/proposalrunner`, `internal/taskmanager`.
- [ ] 1.3 Сверить поддерживаемые runtime-переменные по `internal/config` и `.env.example`.

## 2. Проверка и правки документации

- [ ] 2.1 Проверить `README.md` на соответствие текущим CLI-режимам, зависимостям, stdout/stderr contract и ссылкам на подробные документы.
- [ ] 2.2 Проверить `docs/proposal-runner.md` на соответствие текущему `proposalrunner.Runner`, `ProposalInput`, `AgentExecutor` и GitHub PR workflow.
- [ ] 2.3 Проверить `docs/linear-task-manager.md` на соответствие текущему `TaskManager`, managed state IDs и review-state responsibility.
- [ ] 2.4 Проверить `docs/elk-demo.md` и deploy-файлы на актуальность инструкций запуска, health-check и smoke-test.
- [ ] 2.5 Проверить `architecture.md` на соответствие текущим границам `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, `Logger`.
- [ ] 2.6 Проверить активные `openspec/specs/**/*.md` на явные расхождения с текущей реализацией и документацией.
- [ ] 2.7 Внести минимальные исправления в документы, где найдены устаревшие, неполные или вводящие в заблуждение утверждения.

## 3. Валидация

- [ ] 3.1 Убедиться, что `.env.example` содержит только поддерживаемые ключи без секретов и значений по умолчанию.
- [ ] 3.2 Запустить `openspec validate audit-documentation-freshness --strict`.
- [ ] 3.3 Запустить `go fmt ./...`.
- [ ] 3.4 Запустить `go test ./...`.
- [ ] 3.5 Зафиксировать в итоговом отчете, какие документы проверены, какие расхождения исправлены и какие проверки выполнены.
