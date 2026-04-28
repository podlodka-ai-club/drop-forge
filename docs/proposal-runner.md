# Proposal runner

`orchv3` запускает proposal runner только из orchestration runtime: `CoreOrch` берет Linear-задачи из `Ready to Propose`, строит `ProposalInput`, а внутренний `AgentExecutor` создает OpenSpec proposal во внешнем репозитории. Сейчас единственная реализация `AgentExecutor` запускает Codex CLI.

## Запуск

```bash
orchv3
```

CLI запускается без аргументов и стартует постоянный orchestration monitor. `stdout` остается пустым. Логи monitor-а и шагов `temp`, `git`, `codex`, `github` или `gitlab` печатаются в stderr или настроенный log sink. Модуль `codex` относится к текущей реализации agent executor. В этом же monitor-е задачи из `Ready to Code` обрабатывает Apply runner, а задачи из `Ready to Archive` обрабатывает Archive runner. Оба runner-а используют те же `PROPOSAL_*` настройки репозитория и CLI-путей, но пушат изменения в существующую ветку review request без создания нового PR/MR.

Ручной direct-режим удален: аргументы командной строки и непустой stdin считаются unsupported manual input и возвращают usage error.

После создания review request runner пытается опубликовать отдельный comment/note из последнего непустого сообщения agent executor. Для текущей Codex-реализации это сообщение сохраняется через `codex exec --output-last-message`. Если финальное сообщение пустое или состоит только из whitespace, дополнительный комментарий не создается.

## Внешние prerequisites

- `git` должен быть установлен и доступен по пути `PROPOSAL_GIT_PATH` или через `PATH`.
- `codex` нужен для текущей реализации `AgentExecutor` и должен поддерживать non-interactive формат `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`; prompt передается через stdin.
- Для GitHub-режима (`PROPOSAL_GIT_PROVIDER=github` или переменная не задана) `gh` должен быть установлен, доступен по пути `PROPOSAL_GH_PATH` или через `PATH`, и заранее аутентифицирован для целевого GitHub-репозитория.
- Для GitLab-режима (`PROPOSAL_GIT_PROVIDER=gitlab`) `glab` должен быть установлен, доступен по пути `PROPOSAL_GLAB_PATH` или через `PATH`, и заранее аутентифицирован для целевого GitLab-репозитория. Для self-managed GitLab настройте `glab auth login --hostname <host>` вне приложения.
- `.env` загружается через `github.com/joho/godotenv`: поддерживаются кавычки, комментарии и multiline-значения из godotenv, при этом process environment имеет приоритет над `.env`.

## Контракт вызова

Внутренний контракт `proposalrunner.Runner.Run(ctx, ProposalInput)` принимает структуру с явными полями:

- `Title` — человекочитаемое название задачи. Используется для построения PR title, имени ветки и сообщения коммита. Обязательное; пустое значение приводит к ошибке до начала workflow.
- `Identifier` — опциональный Linear-идентификатор задачи (например, `ZIM-42`). Если задан, PR title и slug ветки получают вид `<Identifier>: <Title>`.
- `AgentPrompt` — полный prompt для agent executor (для orchestrate-режима — task identifier, title, description, comments). Обязательное.

Правило формирования метаданных review request детерминированное: `displayName = "<Identifier>: <Title>"` (или `<Title>` при пустом `Identifier`), и затем title — `displayName` с префиксом `PROPOSAL_PR_TITLE_PREFIX`, усечённый до 72 рун. Содержимое `AgentPrompt` в title/branch/commit не попадает.

В orchestration runtime `coreorch.BuildProposalInput` заполняет все три поля из Linear-задачи. CLI больше не строит `ProposalInput` из args/stdin и не вызывает `proposalrunner.Run` напрямую.

## Runtime-настройки

Доступные переменные перечислены в `.env.example` без значений. Для запуска proposal/apply/archive runner обязательно указать `PROPOSAL_REPOSITORY_URL`; остальные поля имеют безопасные значения по умолчанию в коде и могут быть переопределены через environment. `PROPOSAL_GIT_PROVIDER` выбирает provider review request operations: `github` по умолчанию или `gitlab`. В GitHub-режиме используются команды `gh pr view/create/comment`; в GitLab-режиме используются `glab mr view/create` и `glab mr note create`. `PROPOSAL_GH_PATH` валидируется только для GitHub, `PROPOSAL_GLAB_PATH` только для GitLab. `PROPOSAL_POLL_INTERVAL` задает паузу между проходами orchestration monitor. `PROPOSAL_CODEX_PATH` остается путем к Codex CLI для текущей реализации agent executor.

По умолчанию временная директория сохраняется для диагностики. Чтобы удалять ее после workflow, включите `PROPOSAL_CLEANUP_TEMP`.
