## Why

Сейчас proposal runner жестко запускает только Codex CLI, хотя общий workflow уже отделен от конкретного агента: клонирование репозитория, генерация OpenSpec-артефактов, commit/push и PR. Поддержка Claude CLI позволит использовать тот же проверенный сценарий с альтернативным агентным CLI без дублирования orchestration flow.

## What Changes

- Добавить выбор агентного CLI для proposal runner через runtime-конфигурацию.
- Сохранить Codex CLI как поведение по умолчанию, чтобы существующие установки не требовали миграции.
- Добавить Claude CLI как поддерживаемый backend с собственным путем к бинарю, argv-builder и prompt contract.
- Обобщить Codex-специфичные имена там, где runner работает с любым поддержанным agent CLI.
- Сохранять публикацию финального ответа агента отдельным PR-комментарием, если выбранный CLI позволяет получить такой ответ.
- Обновить `.env.example`, конфигурацию и тесты для нового выбора агента.

## Capabilities

### New Capabilities

Пока нет новых capabilities.

### Modified Capabilities

- `codex-proposal-pr-runner`: Расширить requirement запуска Codex CLI до выбора поддержанного agent CLI, включая Claude CLI, при сохранении текущего Codex workflow по умолчанию.

## Impact

- `internal/config`: новые переменные для выбора agent CLI и пути к Claude CLI; сохранение совместимости с текущим `PROPOSAL_CODEX_PATH`.
- `internal/proposalrunner`: выделение agent CLI backend-ов, переименование Codex-специфичных частей там, где они стали общими, и добавление Claude argv/prompt execution.
- `.env.example`: добавление новых ключей без значений.
- `docs/proposal-runner.md` и README при необходимости: документирование выбора Codex/Claude и prerequisites.
- Тесты proposal runner и config: happy path для Claude, сохранение Codex default, ошибки неизвестного agent CLI и пустых обязательных путей.
