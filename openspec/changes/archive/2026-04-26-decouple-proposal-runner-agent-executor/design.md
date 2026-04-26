## Context

Текущий `internal/proposalrunner.Run` одновременно координирует proposal workflow и содержит детали запуска Codex CLI. В одном методе находятся clone workspace, сборка Codex prompt, argv `codex exec`, чтение `codex-last-message.txt`, проверка `git status`, commit/push, создание PR и публикация комментария.

В `architecture.md` уже зафиксирована целевая граница: `AgentExecutor` должен изолировать orchestration-слой от конкретного coding-agent runtime. Сейчас эта граница существует только как архитектурное намерение, а не как явный контракт в коде.

## Goals / Non-Goals

**Goals:**

- Сделать `proposalrunner` зависимым от внутреннего `AgentExecutor`, а не от Codex CLI argv/protocol.
- Оставить Codex CLI первой и единственной runtime-реализацией в рамках этой задачи.
- Сохранить внешний CLI behavior и текущий happy path создания OpenSpec proposal PR.
- Сохранить тестируемость без реального Codex CLI, GitHub и сетевых вызовов.
- Обновить документацию и архитектурный маппинг под новую границу ответственности.

**Non-Goals:**

- Не добавлять поддержку Cursor, Claude Code, Gemini CLI или другого agent runtime.
- Не вводить plugin registry, динамический выбор агента или очередь задач.
- Не выделять полноценный `GitManager`; git/GitHub шаги остаются внутри текущего workflow.
- Не менять формат OpenSpec artifacts, PR title/body и базовый CLI-вход.

## Decisions

### Ввести `AgentExecutor` как контракт proposal-step

Добавить внутренний контракт, который принимает описание задачи и путь к clone workspace, запускает агент и возвращает результат выполнения. Минимальная модель результата должна включать финальное сообщение агента для PR comment.

Ожидаемая форма ответственности:

- `proposalrunner.Run` отвечает за валидацию input, temp-dir, clone, git status, branch/commit/push, PR и comment.
- `AgentExecutor` отвечает за подготовку agent prompt, запуск конкретного runtime, потоковое логирование agent stdout/stderr и сбор agent result.
- `CodexCLIExecutor` реализует текущий Codex protocol: `codex exec`, `--json`, `--sandbox danger-full-access`, `--output-last-message`, `--cd <clone-dir>`, stdin prompt.

Альтернатива - оставить `BuildCodexPrompt`, `CodexArgs` и чтение last-message в runner, а интерфейс сделать только вокруг `commandrunner.Run`. Это почти не снижает связность: orchestration по-прежнему будет знать Codex-specific protocol.

### Сохранить Codex как реализацию по умолчанию без нового runtime switch

Конфигурация должна продолжать строить рабочий proposal runner с Codex CLI executor по умолчанию. `PROPOSAL_CODEX_PATH` можно оставить как путь к текущей реализации, потому что в этой задаче нет второго агента и нет подтвержденной потребности в `PROPOSAL_AGENT_TYPE`.

Если при реализации потребуется переименование внутренних полей, внешние `.env` keys должны меняться только при явной пользе. Безопаснее оставить существующий env contract и обобщить только внутренние имена там, где это не ломает пользователей.

Альтернатива - сразу ввести `PROPOSAL_AGENT_TYPE=codex` и `PROPOSAL_AGENT_PATH`. Это выглядит расширяемо, но добавляет конфигурационную поверхность без второго backend-а и усложняет миграцию раньше реальной необходимости.

### Перенести Codex-specific helpers к Codex executor

`BuildCodexPrompt`, `CodexArgs`, `ReadLastCodexMessage` и `lastMessageFile` должны стать деталями Codex executor или быть переименованы так, чтобы runner не вызывал их напрямую. Тесты Codex argv и prompt должны остаться, но должны проверять Codex реализацию, а не общий orchestration workflow.

Альтернатива - оставить helper-функции экспортированными в `proposalrunner`. Это проще механически, но сохраняет публичную поверхность с Codex-specific деталями и делает будущую замену runtime более рискованной.

### Логи остаются наблюдаемыми, но module выбирает executor

Runner больше не должен хардкодить module `codex` для agent-step. Codex executor может продолжить писать module `codex`, чтобы не ухудшить диагностику и не ломать текущие ожидания логов. Обобщенный orchestration-слой должен оперировать понятием agent-step и не знать конкретное имя runtime.

Альтернатива - сразу заменить все agent логи на module `agent`. Это чище с точки зрения абстракции, но ухудшает читаемость текущих логов и может создать ненужное изменение поведения для операторов.

## Risks / Trade-offs

- [Интерфейс получится слишком широким] -> начать с минимального input/result, который нужен текущему proposal workflow: task description, clone workspace, temp/work paths для runtime artifacts, writers/logger, final message.
- [Смешение ответственности runner и executor сохранится] -> тестами закрепить, что runner вызывает executor как зависимость, а Codex argv/prompt проверяются в тестах Codex executor.
- [Переименование конфигурации может сломать локальные `.env`] -> не менять внешние env keys без отдельной необходимости; если переименование все же потребуется, явно описать migration и обновить `.env.example`.
- [Документация станет слишком общей] -> писать, что сейчас поддержан только Codex CLI, но он изолирован за `AgentExecutor`.

## Migration Plan

1. Добавить `AgentExecutor` input/result контракт и Codex CLI реализацию.
2. Перевести `Runner` на зависимость от `AgentExecutor`, сохранив default constructor с Codex реализацией.
3. Перенести Codex prompt/argv/last-message детали из orchestration path в Codex executor.
4. Разделить тесты: orchestration tests проверяют порядок workflow через fake executor, Codex tests проверяют конкретный CLI protocol.
5. Обновить README, `docs/proposal-runner.md`, `.env.example` при изменении поддерживаемых ключей и `architecture.md` с текущим маппингом.

Rollback прост: если новая граница окажется неудачной во время разработки, можно оставить Codex implementation и вернуть прямой вызов в runner до архивации change. После релиза rollback потребует сохранить прежний env contract, поэтому внешние ключи лучше не ломать.

## Open Questions

Нет открытых вопросов. В рамках этой задачи Codex остается единственным runtime backend, а новый контракт нужен для разделения ответственности, не для выбора агента пользователем.
