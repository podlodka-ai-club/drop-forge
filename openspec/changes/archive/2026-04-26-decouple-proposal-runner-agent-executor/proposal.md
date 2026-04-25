## Why

`internal/proposalrunner` сейчас напрямую зависит от протокола Codex CLI: знает argv `codex exec`, переменную `PROPOSAL_CODEX_PATH`, флаг `--output-last-message`, Codex-specific prompt builder и модуль логирования `codex`. Это противоречит архитектурной границе `AgentExecutor`, где orchestration-слой должен запускать агентный proposal-step без знания конкретного coding-agent runtime.

## What Changes

- Ввести внутренний контракт `AgentExecutor` для запуска proposal-step в подготовленном clone workspace.
- Изолировать текущий Codex CLI протокол в первой реализации контракта, например `CodexCLIExecutor`.
- Сохранить текущий внешний happy path: CLI принимает описание задачи, runner клонирует репозиторий, агент создает OpenSpec artifacts, затем runner делает branch/commit/push/PR и публикует финальное сообщение агента.
- Обобщить Codex-specific имена в orchestration-слое там, где это безопасно, не добавляя поддержку второго агента в рамках этой задачи.
- Обновить unit-тесты и документацию так, чтобы они проверяли границу agent executor и текущую Codex-реализацию отдельно.

## Capabilities

### New Capabilities

Нет.

### Modified Capabilities

- `codex-proposal-pr-runner`: runner должен зависеть от внутреннего agent executor контракта, а Codex CLI должен остаться поддержанным как конкретная реализация этого контракта.

## Impact

- `internal/proposalrunner`: разделение orchestration workflow и agent execution деталей.
- `internal/config`: возможное обобщение Codex-specific runtime-настроек с сохранением совместимости там, где это нужно для текущего workflow.
- `internal/commandrunner`: остается техническим адаптером запуска процессов и может использоваться Codex-реализацией.
- `cmd/orchv3`: должен сохранить существующий CLI behavior.
- `README.md`, `docs/proposal-runner.md`, `.env.example`: документация и шаблон конфигурации должны отражать новую границу и текущую Codex-реализацию.
- `architecture.md`: нужно обновить маппинг текущего кода, потому что `AgentExecutor` станет явной внутренней границей.
