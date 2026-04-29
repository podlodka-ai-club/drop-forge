# Proposal runner

`orchv3` запускает proposal runner только из orchestration runtime: `CoreOrch` берет Linear-задачи из `Ready to Propose`, строит `ProposalInput`, а внутренний `AgentExecutor` создает OpenSpec proposal во внешнем репозитории. Сейчас единственная реализация `AgentExecutor` запускает Codex CLI.

## Запуск

```bash
orchv3
```

CLI запускается без аргументов и стартует постоянный orchestration monitor. `stdout` остается пустым. Логи monitor-а и шагов `temp`, `git`, `codex` и `github` печатаются в stderr или настроенный log sink. Модуль `codex` относится к текущей реализации agent executor. В этом же monitor-е задачи из `Ready to Code` обрабатывает Apply runner, а задачи из `Ready to Archive` обрабатывает Archive runner. Оба runner-а используют те же `PROPOSAL_*` настройки репозитория и CLI-путей, но пушат изменения в существующую ветку PR без создания нового pull request.

Ручной direct-режим удален: аргументы командной строки и непустой stdin считаются unsupported manual input и возвращают usage error.

После создания pull request runner пытается опубликовать отдельный PR comment из последнего непустого сообщения agent executor. Для текущей Codex-реализации это сообщение сохраняется через `codex exec --output-last-message`. Если финальное сообщение пустое или состоит только из whitespace, дополнительный комментарий не создается.

Если включена Cross-Agent Review-стадия (см. `LINEAR_STATE_NEED_*_AI_REVIEW_ID` и `REVIEW_ROLE_*` переменные в `.env.example`), proposal runner после push переводит задачу не сразу в `Need Proposal Review`, а в `Need Proposal AI Review`. Из этого state monitor подхватывает задачу следующим тиком и запускает `ReviewRunner`, который публикует автоматическое PR review «противоположной» моделью с inline-комментариями и fix-prompt'ами; только после публикации задача переходит к человеку. Apply и Archive стадии работают по тому же паттерну. При выключенной фиче (любой из трёх AI-review state ID пустой) поведение прежнее — переход напрямую в человеческий review-state. Каждый producer-runner также добавляет в commit message git-trailer `Produced-By:`/`Produced-Model:`/`Produced-Stage:` чтобы review-стадия выбрала «противоположную» модель.

## Внешние prerequisites

- `git` должен быть установлен и доступен по пути `PROPOSAL_GIT_PATH` или через `PATH`.
- `codex` нужен для текущей реализации `AgentExecutor` и должен поддерживать non-interactive формат `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`; prompt передается через stdin.
- `gh` должен быть установлен, доступен по пути `PROPOSAL_GH_PATH` или через `PATH`, и заранее аутентифицирован для целевого GitHub-репозитория.
- `.env` загружается через `github.com/joho/godotenv`: поддерживаются кавычки, комментарии и multiline-значения из godotenv, при этом process environment имеет приоритет над `.env`.

## Контракт вызова

Внутренний контракт `proposalrunner.Runner.Run(ctx, ProposalInput)` принимает структуру с явными полями:

- `Title` — человекочитаемое название задачи. Используется для построения PR title, имени ветки и сообщения коммита. Обязательное; пустое значение приводит к ошибке до начала workflow.
- `Identifier` — опциональный Linear-идентификатор задачи (например, `ZIM-42`). Если задан, PR title и slug ветки получают вид `<Identifier>: <Title>`.
- `AgentPrompt` — полный prompt для agent executor (для orchestrate-режима — task identifier, title, description, comments). Обязательное.

Правило формирования метаданных PR детерминированное: `displayName = "<Identifier>: <Title>"` (или `<Title>` при пустом `Identifier`), и затем PR title — `displayName` с префиксом `PROPOSAL_PR_TITLE_PREFIX`, усечённый до 72 рун. Содержимое `AgentPrompt` в title/branch/commit не попадает.

В orchestration runtime `coreorch.BuildProposalInput` заполняет все три поля из Linear-задачи. CLI больше не строит `ProposalInput` из args/stdin и не вызывает `proposalrunner.Run` напрямую.

## Runtime-настройки

Доступные переменные перечислены в `.env.example` без значений. Для запуска proposal/apply/archive runner обязательно указать `PROPOSAL_REPOSITORY_URL`; остальные поля имеют безопасные значения по умолчанию в коде и могут быть переопределены через environment. `PROPOSAL_POLL_INTERVAL` задает паузу между проходами orchestration monitor. `PROPOSAL_CODEX_PATH` остается путем к Codex CLI для текущей реализации agent executor.

По умолчанию временная директория сохраняется для диагностики. Чтобы удалять ее после workflow, включите `PROPOSAL_CLEANUP_TEMP`.
