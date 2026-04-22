## Why

Сейчас orchestrator автоматизирует только подготовку OpenSpec proposal: по описанию задачи он клонирует репозиторий, запускает Codex со skill `openspec-propose` и публикует PR. Следующий естественный шаг workflow - автоматизировать реализацию принятого proposal через `openspec-apply`, чтобы оператор мог передать ветку с proposal и получить PR с кодом без ручного клонирования и запуска Codex.

## What Changes

- Добавить режим запуска OpenSpec apply, который принимает имя ветки с proposal как основной вход.
- Для apply клонировать целевой репозиторий и переключаться на переданную proposal-ветку до запуска Codex.
- Запускать Codex CLI со skill `openspec-apply` в клоне proposal-ветки.
- Не создавать новую ветку до выполнения apply: реализация должна стартовать из переданной proposal-ветки, а ветка для PR создается только для результатов implementation workflow.
- По возможности переиспользовать существующий код proposal runner: конфигурацию, создание temp workspace, запуск внешних команд, логирование, проверку изменений, commit/push/PR и тестовые подмены.
- При необходимости выполнить небольшой рефакторинг, чтобы общие шаги proposal/apply не дублировались и различались только входом, подготовкой git state, Codex prompt и метаданными PR.
- Покрыть новый apply workflow тестами с fake command runner.

## Capabilities

### New Capabilities

Пока нет новых capabilities.

### Modified Capabilities

- `codex-proposal-pr-runner`: Добавить apply-сценарий к существующему PR runner: входом становится proposal branch, clone выполняется от этой ветки, Codex запускается со skill `openspec-apply`, а результаты публикуются в отдельный PR.

## Impact

- `internal/proposalrunner`: новый apply entrypoint или обобщенный runner workflow с переиспользованием существующих шагов.
- `cmd/orchv3`: публичный CLI/API должен уметь выбирать proposal или apply режим без поломки текущего proposal поведения.
- `internal/config` и `.env.example`: возможное добавление apply-specific префиксов веток/заголовков PR, если текущие proposal-настройки недостаточно универсальны.
- `docs/proposal-runner.md`: обновить документацию по запуску proposal/apply workflow.
- Тесты для runner, config и CLI parsing.
