## Why

Сейчас `orchv3` умеет автоматизировать только подготовку OpenSpec proposal через Codex CLI и pull request. Следующий рабочий шаг в том же процессе - применить уже подготовленный proposal, запустив `openspec-apply` от ветки с proposal и опубликовав реализацию отдельным PR.

## What Changes

- Добавить apply runner, который принимает название ветки с proposal как основной вход.
- Для apply workflow клонировать целевой репозиторий, переключаться на переданную proposal-ветку и запускать Codex CLI с инструкцией использовать skill `openspec-apply`.
- Не создавать новую proposal-ветку перед запуском Codex apply; ветка с implementation PR должна создаваться только после появления изменений от Codex.
- По возможности переиспользовать существующие шаги proposal runner: конфигурацию, создание temp workspace, запуск внешних команд, проверку `git status`, commit, push, создание PR и логирование.
- Сохранить обратную совместимость текущего proposal workflow.
- Обновить документацию, конфигурацию и тесты для нового apply-сценария.

## Capabilities

### New Capabilities
- `codex-apply-pr-runner`: запуск Codex для применения OpenSpec proposal из указанной ветки и публикация результата в pull request

### Modified Capabilities
Нет.

## Impact

- `cmd/orchv3`: выбор режима proposal/apply и чтение входных данных CLI/stdin.
- `internal/proposalrunner` или новый внутренний пакет runner: выделение общих workflow-шагов и добавление apply workflow.
- `internal/config`: runtime-настройки для apply runner и синхронизация `.env.example`.
- `docs`: описание запуска apply runner и prerequisites.
- `openspec/specs`: новая спецификация apply runner.
- Тесты: unit tests на валидацию входа, последовательность git/Codex/gh команд, отсутствие создания proposal-ветки до apply и сохранение текущего proposal behavior.
