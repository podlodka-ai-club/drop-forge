# orchv3

`orchv3` — Go CLI для запуска proposal-runner workflow: утилита принимает описание задачи, создает OpenSpec proposal во внешнем репозитории через внутренний `AgentExecutor` и возвращает URL созданного pull request. Текущая реализация `AgentExecutor` использует Codex CLI.

README описывает только текущий подтвержденный сценарий. Детали workflow и prerequisites вынесены в [docs/proposal-runner.md](docs/proposal-runner.md).

## Что умеет CLI сейчас

- принять описание задачи аргументами командной строки;
- принять описание задачи через `stdin`;
- запустить proposal workflow во внешнем репозитории через `AgentExecutor`;
- вывести итоговый PR URL в `stdout`;
- писать пошаговые логи workflow в `stderr`.

Если запустить CLI без аргументов и без данных в `stdin`, proposal workflow не стартует.

## Что уже есть для Linear TaskManager

В репозитории появился внутренний пакет `TaskManager` для будущего `CoreOrch`, но он пока не подключен к публичному CLI workflow. Его роль строго ограничена интеграцией с Linear:

- читать задачи только из одного настроенного Linear project;
- фильтровать задачи по управляемым state'ам `ready to propose`, `ready to code`, `ready to archive`;
- возвращать payload задачи для `CoreOrch`, включая описание, текущий state и комментарии;
- записывать изменения обратно в Linear: move task, add comment, add PR.

Для HITL-сценария `TaskManager` возвращает все comments задачи без дополнительной фильтрации, чтобы `CoreOrch` мог использовать human feedback при повторном proposal/coding проходе.

## Зависимости

Для локального запуска нужны:

- Go `1.24.2` или совместимая версия для сборки и запуска проекта;
- `git`;
- `codex` для текущей реализации agent executor;
- `gh`;
- доступ к целевому GitHub-репозиторию и предварительная аутентификация `gh`;
- настроенный `.env` с runtime-параметрами.

Go-модуль и зависимости зафиксированы в [go.mod](go.mod). Подробные требования к proposal-runner workflow описаны в [docs/proposal-runner.md](docs/proposal-runner.md).

## Настройка окружения

1. Создайте локальный `.env` на основе [.env.example](.env.example).
2. Заполните значения переменных для вашей среды.
3. Убедитесь, что `git`, `codex` и `gh` доступны по путям из окружения или через `PATH`.

Полный список поддерживаемых переменных хранится в [.env.example](.env.example), а `.env` подхватывается через `godotenv`. Значения из process environment имеют приоритет над `.env`.

Практически важные переменные:

- `PROPOSAL_REPOSITORY_URL` — обязательный URL целевого репозитория;
- `PROPOSAL_BASE_BRANCH`, `PROPOSAL_REMOTE_NAME`, `PROPOSAL_BRANCH_PREFIX`, `PROPOSAL_PR_TITLE_PREFIX` — параметры git/GitHub workflow;
- `PROPOSAL_GIT_PATH`, `PROPOSAL_CODEX_PATH`, `PROPOSAL_GH_PATH` — пути к внешним CLI; `PROPOSAL_CODEX_PATH` относится к текущей Codex-реализации `AgentExecutor`;
- `PROPOSAL_CLEANUP_TEMP` — удалять ли временную директорию после выполнения;
- `LINEAR_API_URL`, `LINEAR_API_TOKEN`, `LINEAR_PROJECT_ID` — подключение к Linear и фильтр по проекту;
- `LINEAR_STATE_READY_TO_PROPOSE_ID`, `LINEAR_STATE_READY_TO_CODE_ID`, `LINEAR_STATE_READY_TO_ARCHIVE_ID` — идентификаторы управляемых Linear state'ов для `TaskManager`;
- `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` — target state IDs для review-этапов, которые `CoreOrch` использует при вызове `TaskManager.MoveTask(...)`;
- `APP_ENV`, `APP_NAME`, `LOG_LEVEL`, `HTTP_PORT`, `OPENAI_API_KEY` — общие runtime-параметры, поддерживаемые конфигом.

## Запуск

Перед первым запуском установите зависимости и подготовьте `.env`.

Запуск с описанием задачи в аргументах:

```bash
go run ./cmd/orchv3 "Добавить сценарий ..."
```

Запуск с передачей задачи через `stdin`:

```bash
printf '%s\n' "Добавить сценарий ..." | go run ./cmd/orchv3
```

При успешном выполнении:

- `stdout` содержит только URL созданного pull request, чтобы результат было удобно использовать в скриптах;
- `stderr` содержит пошаговые логи workflow (`temp`, `git`, `codex`, `github`) и сообщения CLI.

## Ключевые директории

- [cmd/orchv3](cmd/orchv3) — точка входа CLI;
- [internal/config](internal/config) — загрузка и валидация конфигурации;
- [internal/proposalrunner](internal/proposalrunner) — orchestration proposal workflow и текущая Codex-реализация `AgentExecutor`;
- [internal/taskmanager](internal/taskmanager) — Linear-facing слой для будущего `CoreOrch`;
- [docs](docs) — дополнительная документация;
- [openspec](openspec) — спецификации и changes.

## Проверка изменений

Минимальные команды перед завершением работы:

```bash
go fmt ./...
go test ./...
```

## Дополнительная документация

- [docs/proposal-runner.md](docs/proposal-runner.md) — подробное описание proposal-runner workflow и prerequisites;
- [docs/linear-task-manager.md](docs/linear-task-manager.md) — описание текущего scope `TaskManager` и его места в целевой архитектуре;
- [.env.example](.env.example) — шаблон поддерживаемых переменных окружения;
- [openspec](openspec) — текущие и архивные изменения по OpenSpec.
