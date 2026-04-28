# Proposal runner

Drop Forge запускает proposal runner только из orchestration runtime: `CoreOrch` берет Linear-задачи из `Ready to Propose`, строит `ProposalInput`, а внутренний `AgentExecutor` создает OpenSpec proposal во внешнем репозитории. Сейчас единственная реализация `AgentExecutor` запускает Codex CLI.

## Запуск

```bash
orchv3
```

CLI запускается без аргументов и стартует постоянный Drop Forge orchestration monitor. `stdout` остается пустым. Логи monitor-а и шагов `temp`, `git`, `codex` и `github` печатаются в stderr или настроенный log sink. Модуль `codex` относится к текущей реализации agent executor. В этом же monitor-е задачи из `Ready to Code` обрабатывает Apply runner, а задачи из `Ready to Archive` обрабатывает Archive runner. Все runner-ы используют общие `DROP_FORGE_*` настройки репозитория и CLI-путей; proposal-specific `PROPOSAL_*` ключи отвечают только за metadata нового proposal PR.

Ручной direct-режим удален: аргументы командной строки и непустой stdin считаются unsupported manual input и возвращают usage error.

После создания pull request runner пытается опубликовать отдельный PR comment из последнего непустого сообщения agent executor. Для текущей Codex-реализации это сообщение сохраняется через `codex exec --output-last-message`. Если финальное сообщение пустое или состоит только из whitespace, дополнительный комментарий не создается.

## Внешние prerequisites

- `git` должен быть установлен и доступен по пути `DROP_FORGE_GIT_PATH` или через `PATH`.
- `codex` нужен для текущей реализации `AgentExecutor` и должен поддерживать non-interactive формат `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`; prompt передается через stdin.
- `gh` должен быть установлен, доступен по пути `DROP_FORGE_GH_PATH` или через `PATH`, и заранее аутентифицирован для целевого GitHub-репозитория.
- `.env` загружается через `github.com/joho/godotenv`: поддерживаются кавычки, комментарии и multiline-значения из godotenv, при этом process environment имеет приоритет над `.env`.

## Контракт вызова

Внутренний контракт `proposalrunner.Runner.Run(ctx, ProposalInput)` принимает структуру с явными полями:

- `Title` — человекочитаемое название задачи. Используется для построения PR title, имени ветки и сообщения коммита. Обязательное; пустое значение приводит к ошибке до начала workflow.
- `Identifier` — опциональный Linear-идентификатор задачи (например, `ZIM-42`). Если задан, PR title и slug ветки получают вид `<Identifier>: <Title>`.
- `AgentPrompt` — полный prompt для agent executor (для orchestrate-режима — task identifier, title, description, comments). Обязательное.

Правило формирования метаданных PR детерминированное: `displayName = "<Identifier>: <Title>"` (или `<Title>` при пустом `Identifier`), и затем PR title — `displayName` с префиксом `PROPOSAL_PR_TITLE_PREFIX`, усечённый до 72 рун. Содержимое `AgentPrompt` в title/branch/commit не попадает.

В orchestration runtime `coreorch.BuildProposalInput` заполняет все три поля из Linear-задачи. CLI больше не строит `ProposalInput` из args/stdin и не вызывает `proposalrunner.Run` напрямую.

## Runtime-настройки

Доступные переменные перечислены в `.env.example` без значений. Для запуска proposal/apply/archive runner обязательно указать `DROP_FORGE_REPOSITORY_URL`; остальные shared runtime поля имеют безопасные значения по умолчанию в коде и могут быть переопределены через environment. `DROP_FORGE_POLL_INTERVAL` задает паузу между проходами orchestration monitor. `DROP_FORGE_CODEX_PATH` остается путем к Codex CLI для текущей реализации agent executor.

По умолчанию временная директория сохраняется для диагностики. Чтобы удалять ее после workflow, включите `DROP_FORGE_CLEANUP_TEMP`.

Миграция со старых shared proposal keys breaking: `PROPOSAL_REPOSITORY_URL`, `PROPOSAL_BASE_BRANCH`, `PROPOSAL_REMOTE_NAME`, `PROPOSAL_CLEANUP_TEMP`, `PROPOSAL_POLL_INTERVAL`, `PROPOSAL_GIT_PATH`, `PROPOSAL_CODEX_PATH` и `PROPOSAL_GH_PATH` больше не принимаются. Используйте соответствующие `DROP_FORGE_*` ключи. `PROPOSAL_BRANCH_PREFIX` и `PROPOSAL_PR_TITLE_PREFIX` остаются proposal-specific настройками.
