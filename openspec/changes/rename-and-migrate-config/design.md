## Context

Сейчас `orchv3` используется как рабочее имя Go module, CLI entrypoint, дефолтный `APP_NAME`, часть README/документации, prefix временных директорий и пользовательское описание orchestration monitor. Конфигурация также исторически использует `PROPOSAL_*`, хотя одни и те же настройки репозитория, `git`, `codex`, `gh`, cleanup и polling уже используются proposal/apply/archive flow.

Задача DRO-37 просит рефакторинг названия и миграцию конфигурации, а также 3 веселых варианта с ассоциацией "коты + подлодка". Предлагаемые варианты: `Catmarine`, `Purriscope`, `Meowrine`. Для реализации в proposal выбран `Catmarine`; если владелец задачи выберет другой вариант, implementation должен заменить только централизованные identity constants и связанные имена ключей/документов.

## Goals / Non-Goals

**Goals:**
- Сделать `Catmarine` новым публичным именем приложения в CLI/help, README, docs, service label и дефолтном `APP_NAME`.
- Перенести общие runner/orchestration env-ключи из устаревшего `PROPOSAL_*` namespace в `CATMARINE_*`.
- Сохранить совместимость со старым `.env`: если новый ключ отсутствует, loader читает legacy `PROPOSAL_*`.
- Задокументировать миграцию и покрыть приоритеты ключей тестами.
- Сохранить простую структуру проекта без новых внешних зависимостей.

**Non-Goals:**
- Не менять Linear namespace `LINEAR_*`, потому что он описывает внешний сервис, а не бренд приложения.
- Не менять OpenAI/Logstash ключи, если их смысл не связан с новым именем.
- Не удалять legacy `PROPOSAL_*` в рамках этой change; удаление требует отдельной breaking-change спеки.
- Не менять orchestration flow proposal/apply/archive и не выделять новый `GitManager`.

## Decisions

1. Основное имя для реализации: `Catmarine`.

   Альтернативы:
   - `Purriscope`: лучше обыгрывает подлодочный перископ, но хуже читается как имя CLI.
   - `Meowrine`: веселее, но менее очевидно при быстром чтении.

   Решение: использовать `Catmarine` как дефолтное публичное имя и basis для env namespace `CATMARINE_*`.

2. Конфигурационный namespace мигрирует с `PROPOSAL_*` на `CATMARINE_*`.

   `PROPOSAL_*` больше не отражает фактическую роль настроек: repository URL, branch, command paths, cleanup и poll interval используются не только proposal-runner'ом. Новые ключи:
   - `CATMARINE_REPOSITORY_URL`
   - `CATMARINE_BASE_BRANCH`
   - `CATMARINE_REMOTE_NAME`
   - `CATMARINE_BRANCH_PREFIX`
   - `CATMARINE_PR_TITLE_PREFIX`
   - `CATMARINE_CLEANUP_TEMP`
   - `CATMARINE_POLL_INTERVAL`
   - `CATMARINE_GIT_PATH`
   - `CATMARINE_CODEX_PATH`
   - `CATMARINE_GH_PATH`

   Альтернатива: использовать нейтральный namespace `ORCH_*`. Он менее привязан к бренду, но задача прямо просит новое название; для раннего проекта читаемость и простота важнее дополнительной абстракции.

3. Loader читает новый ключ первым, затем legacy alias.

   Для каждой мигрируемой настройки порядок такой:
   - если `CATMARINE_*` задан и не пустой, использовать его;
   - иначе использовать соответствующий `PROPOSAL_*`;
   - иначе использовать существующий дефолт или validation error.

   Если оба ключа заданы, новый ключ имеет приоритет. Это позволяет обновлять `.env` постепенно и не ломает старые окружения.

4. Внутренняя структура конфигурации остается минимальной.

   Можно переименовать `ProposalRunnerConfig` в более общий `RepositoryRunnerConfig`, потому что один config используется proposal/apply/archive runners. Переименование должно быть механическим и сопровождаться тестами, но без нового пакета или интерфейса.

5. CLI entrypoint получает новое имя без жесткого удаления старого.

   Основной entrypoint должен стать `cmd/catmarine`. Если `cmd/orchv3` сохраняется как compatibility wrapper, он должен вызывать тот же runtime и не иметь отдельной конфигурационной логики. Полное удаление старого entrypoint не требуется в этой change.

## Risks / Trade-offs

- [Двойной namespace env-ключей усложняет loader] -> Mitigation: реализовать маленькие helper-функции `stringFromEnvAliases`, `boolFromEnvAliases`, `durationFromEnvAliases` и покрыть приоритеты table-driven tests.
- [Название может измениться после review] -> Mitigation: держать имя в одном месте и не размазывать brand strings по пакетам.
- [Переименование Go module может затронуть все imports] -> Mitigation: делать механически, запускать `go fmt ./...` и `go test ./...`; если внешний import path еще не стабилен, это приемлемо.
- [Legacy ключи могут остаться надолго] -> Mitigation: README должен явно пометить `PROPOSAL_*` как deprecated aliases и указать, что удаление будет отдельной breaking change.

## Migration Plan

1. Ввести application identity constants/defaults для `Catmarine` и заменить пользовательские строки `orchv3`, где они являются публичным именем, а не исторической ссылкой.
2. Добавить alias-aware config helpers и перевести runner/orchestration settings на чтение `CATMARINE_*` с fallback на `PROPOSAL_*`.
3. Обновить `.env.example`: основные ключи `CATMARINE_*`, legacy aliases только в документации, без значений по умолчанию.
4. Переименовать CLI entrypoint в `cmd/catmarine`; при необходимости оставить `cmd/orchv3` как thin wrapper.
5. Обновить README/docs/architecture в местах, где меняется публичное имя, CLI-команды или конфигурационный namespace.
6. Добавить/обновить тесты `internal/config`, CLI wiring и runner temp-prefix/name expectations.
7. Запустить `go fmt ./...` и `go test ./...`.

Rollback: вернуть `.env` на legacy `PROPOSAL_*` возможно без отката кода, потому что aliases сохраняются. Если новое CLI имя вызывает проблемы, compatibility wrapper `cmd/orchv3` остается рабочим до отдельного удаления.

## Open Questions

- Финальное название нужно подтвердить владельцу задачи. Recommended option: `Catmarine`, потому что это самый понятный и короткий вариант из трех.
- Нужно ли в этой же change переименовывать Go module path с `orchv3` на `catmarine`? Recommended option: да, если репозиторий еще не публикуется как external Go module; иначе оставить module path до отдельного релиза.
