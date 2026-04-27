# orchv3

`orchv3` — Go CLI для proposal/apply/archive оркестрации. Утилита запускает постоянный мониторинг Linear-задач, создает OpenSpec proposal PR для задач, готовых к proposal, применяет принятые OpenSpec changes для задач, готовых к code, и архивирует завершенные changes для задач, готовых к archive.

Детали proposal runner workflow и prerequisites вынесены в [docs/proposal-runner.md](docs/proposal-runner.md). Детали Linear-facing слоя описаны в [docs/linear-task-manager.md](docs/linear-task-manager.md).

## Что умеет CLI сейчас

- запустить постоянный orchestration monitor без аргументов CLI;
- запустить proposal workflow во внешнем репозитории через `AgentExecutor`;
- для Linear-задач в `Ready to Propose` создать proposal PR, прикрепить PR URL и перевести задачу в `Need Proposal Review`;
- для Linear-задач в `Ready to Code` взять attached PR URL или branch, применить OpenSpec Apply в той же ветке и перевести задачу в `Need Code Review`;
- для Linear-задач в `Ready to Archive` взять attached PR URL или branch, выполнить OpenSpec Archive в той же ветке и перевести задачу в `Need Archive Review`;
- писать структурные JSON Lines логи workflow в `stderr` или настроенный sink.

Ручной запуск proposal по описанию задачи из args/stdin удален. Любые CLI-аргументы или непустой `stdin` возвращают usage error.

## Proposal Orchestration

Default runtime связывает `CoreOrch`, `TaskManager`, `proposalrunner`, `applyrunner` и `archiverunner` в долгоживущий polling loop:

- `TaskManager` читает managed Linear tasks из одного настроенного project;
- `CoreOrch` маршрутизирует `LINEAR_STATE_READY_TO_PROPOSE_ID` в proposal workflow, `LINEAR_STATE_READY_TO_CODE_ID` в Apply workflow и `LINEAR_STATE_READY_TO_ARCHIVE_ID` в Archive workflow;
- `CoreOrch` формирует input из `identifier`, `title`, `description`, `comments` и, для Apply/Archive, attached PR URL или branch;
- `proposalrunner` создает OpenSpec proposal PR во внешнем репозитории;
- `applyrunner` клонирует репозиторий, переключается на ветку задачи, запускает OpenSpec Apply через Codex CLI, коммитит и пушит изменения без создания нового PR;
- `archiverunner` клонирует репозиторий, переключается на ветку задачи, запускает OpenSpec Archive через Codex CLI, коммитит и пушит изменения без создания нового PR;
- после успеха `CoreOrch` вызывает `TaskManager.AddPR(...)`, затем `TaskManager.MoveTask(...)` в `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.
- после успешного Apply `CoreOrch` переводит задачу из `LINEAR_STATE_CODE_IN_PROGRESS_ID` в `LINEAR_STATE_NEED_CODE_REVIEW_ID`.
- после успешного Archive `CoreOrch` переводит задачу из `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` в `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`.
- после каждого прохода monitor ждет `PROPOSAL_POLL_INTERVAL` и запускает следующий проход до остановки процесса.

Если отдельный orchestration pass падает, monitor пишет structured error и продолжает следующий проход после polling interval. Если runner падает или Linear не смог прикрепить PR, задача не переводится в review state. Если PR уже прикреплен, но move task упал, ошибка логируется с контекстом задачи и PR URL.

## Зависимости

Для локального запуска нужны:

- Go `1.24.2` или совместимая версия для сборки и запуска проекта;
- `git`;
- `codex` для текущей реализации agent executor;
- `gh`;
- доступ к целевому GitHub-репозиторию и предварительная аутентификация `gh`;
- Linear API token и настроенные workflow state IDs для orchestration monitor;
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
- `PROPOSAL_POLL_INTERVAL` — интервал между проходами orchestration monitor, например `30s` или `1m`;
- `LINEAR_API_URL`, `LINEAR_API_TOKEN`, `LINEAR_PROJECT_ID` — подключение к Linear и фильтр по проекту;
- `LINEAR_STATE_READY_TO_PROPOSE_ID`, `LINEAR_STATE_READY_TO_CODE_ID`, `LINEAR_STATE_READY_TO_ARCHIVE_ID` — идентификаторы управляемых Linear state'ов для `TaskManager`;
- `LINEAR_STATE_PROPOSING_IN_PROGRESS_ID`, `LINEAR_STATE_CODE_IN_PROGRESS_ID`, `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` — target state IDs для in-progress переходов;
- `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` — target state IDs для review-этапов, которые `CoreOrch` использует при вызове `TaskManager.MoveTask(...)`;
- `APP_ENV`, `APP_NAME`, `LOG_LEVEL`, `HTTP_PORT`, `OPENAI_API_KEY` — общие runtime-параметры, поддерживаемые конфигом.

## Запуск

Перед первым запуском установите зависимости и подготовьте `.env`.

### Orchestration monitor

Этот режим сам берет задачи из Linear. `stdout` остается пустым; результат и ошибки видны в structured logs.

```bash
go run ./cmd/orchv3
```

Минимальная ручная проверка:

1. В Linear подготовьте задачу в state, чей ID указан в `LINEAR_STATE_READY_TO_PROPOSE_ID`.
2. Убедитесь, что `.env` заполнен для `PROPOSAL_*`, `LINEAR_*`, `git`, `codex` и `gh`.
3. Запустите `go run ./cmd/orchv3`.
4. Проверьте, что в Linear к задаче прикрепился PR URL, а state сменился на `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`.
5. Для Apply подготовьте задачу в state из `LINEAR_STATE_READY_TO_CODE_ID` с attached Pull Request URL; после успешного прохода state должен смениться на `LINEAR_STATE_NEED_CODE_REVIEW_ID`, а изменения появиться в ветке PR.
6. Для Archive подготовьте задачу в state из `LINEAR_STATE_READY_TO_ARCHIVE_ID` с attached Pull Request URL; после успешного прохода state должен смениться на `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`, а archive commit появиться в ветке PR.

## Ключевые директории

- [cmd/orchv3](cmd/orchv3) — точка входа CLI;
- [internal/config](internal/config) — загрузка и валидация конфигурации;
- [internal/coreorch](internal/coreorch) — orchestration layer для proposal, Apply и Archive stage;
- [internal/proposalrunner](internal/proposalrunner) — orchestration proposal workflow и текущая Codex-реализация `AgentExecutor`;
- [internal/applyrunner](internal/applyrunner) — Apply workflow для реализации OpenSpec changes в существующей ветке задачи;
- [internal/archiverunner](internal/archiverunner) — Archive workflow для архивирования OpenSpec changes в существующей ветке задачи;
- [internal/taskmanager](internal/taskmanager) — Linear-facing слой для чтения и обновления задач;
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
