# Proposal runner

`orchv3` может принять описание задачи аргументами CLI или через stdin, создать OpenSpec proposal во внешнем репозитории через внутренний `AgentExecutor` и вернуть URL pull request. Сейчас единственная реализация `AgentExecutor` запускает Codex CLI.

## Запуск

```bash
orchv3 "Добавить сценарий ..."
```

или:

```bash
printf '%s\n' "Добавить сценарий ..." | orchv3
```

При запуске workflow итоговый PR URL печатается в stdout. Логи шагов `temp`, `git`, `codex` и `github` печатаются в stderr, чтобы stdout можно было использовать в скриптах. Модуль `codex` относится к текущей реализации agent executor.

После создания pull request runner пытается опубликовать отдельный PR comment из последнего непустого сообщения agent executor. Для текущей Codex-реализации это сообщение сохраняется через `codex exec --output-last-message`. Если финальное сообщение пустое или состоит только из whitespace, дополнительный комментарий не создается.

## Внешние prerequisites

- `git` должен быть установлен и доступен по пути `PROPOSAL_GIT_PATH` или через `PATH`.
- `codex` нужен для текущей реализации `AgentExecutor` и должен поддерживать non-interactive формат `codex exec --json --sandbox danger-full-access --output-last-message <path> --cd <clone-dir> -`; prompt передается через stdin.
- `gh` должен быть установлен, доступен по пути `PROPOSAL_GH_PATH` или через `PATH`, и заранее аутентифицирован для целевого GitHub-репозитория.
- `.env` загружается через `github.com/joho/godotenv`: поддерживаются кавычки, комментарии и multiline-значения из godotenv, при этом process environment имеет приоритет над `.env`.

## Runtime-настройки

Доступные переменные перечислены в `.env.example` без значений. Для запуска proposal runner обязательно указать `PROPOSAL_REPOSITORY_URL`; остальные поля имеют безопасные значения по умолчанию в коде и могут быть переопределены через environment. `PROPOSAL_CODEX_PATH` остается путем к Codex CLI для текущей реализации agent executor.

По умолчанию временная директория сохраняется для диагностики. Чтобы удалять ее после workflow, включите `PROPOSAL_CLEANUP_TEMP`.
