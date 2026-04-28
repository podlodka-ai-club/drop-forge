## Context

Текущий runtime уже обрабатывает несколько стадий: proposal, apply и archive. При этом часть конфигурации и внешних сообщений осталась proposal-only: `PROPOSAL_REPOSITORY_URL`, `PROPOSAL_POLL_INTERVAL`, пути к `git/codex/gh`, cleanup временных директорий и название `orchv3`. Эти параметры фактически используются шире одной стадии, поэтому новые стадии будут усиливать путаницу и дублирование.

Целевое внешнее имя приложения - `Drop Forge`. Внутренний Go module path можно оставить `orchv3`, если его переименование не требуется для пользовательского поведения: полное переименование module path затронет все imports и не дает runtime-пользы на этом этапе.

## Goals / Non-Goals

**Goals:**

- Зафиксировать внешнее имя приложения как `Drop Forge` в CLI, логах, документации и generated metadata.
- Разделить ENV на общие `DROP_FORGE_*` runtime-параметры и stage-specific параметры.
- Перенести общие `PROPOSAL_*` ключи на нейтральные `DROP_FORGE_*`.
- Оставить proposal-specific ключи там, где они действительно управляют proposal PR metadata.
- Обновить `.env.example`, config loading, validation и тесты как единый контракт.

**Non-Goals:**

- Не добавлять новый scheduler, retry/backoff или иной orchestration engine.
- Не менять Linear workflow states и не переименовывать сами стадии proposal/apply/archive.
- Не менять протокол Codex CLI, GitHub CLI или Linear API.
- Не требовать переименования Go module path, бинарного каталога или пакетов, если это не нужно для внешнего поведения Drop Forge.

## Decisions

1. Общие runtime-ключи получают префикс `DROP_FORGE_*`.

   Новая схема:
   - `DROP_FORGE_REPOSITORY_URL`
   - `DROP_FORGE_BASE_BRANCH`
   - `DROP_FORGE_REMOTE_NAME`
   - `DROP_FORGE_CLEANUP_TEMP`
   - `DROP_FORGE_POLL_INTERVAL`
   - `DROP_FORGE_GIT_PATH`
   - `DROP_FORGE_CODEX_PATH`
   - `DROP_FORGE_GH_PATH`

   Эти ключи описывают весь orchestration runtime и используются proposal/apply/archive runner-ами. Альтернатива - ввести `ORCHESTRATOR_*`, но `DROP_FORGE_*` лучше связывает runtime contract с новым названием продукта.

2. Proposal-specific metadata остается под `PROPOSAL_*`.

   `PROPOSAL_BRANCH_PREFIX` и `PROPOSAL_PR_TITLE_PREFIX` управляют только созданием нового proposal PR. Apply и Archive пушат в существующую ветку, поэтому перенос этих ключей в общий namespace сделал бы их менее точными.

3. `APP_NAME` становится необязательным override, но дефолт внешнего имени должен быть `Drop Forge`.

   Логи и сервисное имя должны использовать `Drop Forge`, когда `APP_NAME` не задан. Это сохраняет возможность локального override для инфраструктуры логирования, но убирает прежнее `orchv3` как публичный дефолт.

4. Старые общие `PROPOSAL_*` ключи не поддерживаются как fallback.

   Изменение намеренно breaking: одновременная поддержка двух namespaces увеличит неясность, ради устранения которой выполняется миграция. Ошибки валидации должны называть новые ключи и помогать быстро обновить `.env`.

5. Архитектурный документ нужно обновить.

   Изменение затрагивает runtime identity и границы общей конфигурации между CLI, CoreOrch, runner-ами и GitManager, поэтому `architecture.md` должен отражать Drop Forge и новую схему config ownership.

## Risks / Trade-offs

- [Risk] Локальные `.env` и окружения CI перестанут запускаться со старыми ключами. → Mitigation: явно перечислить новые ключи в `.env.example`, README/docs и ошибках валидации.
- [Risk] Часть текста `proposal` в коде должна остаться, потому что стадия действительно называется proposal. → Mitigation: переименовывать только общий runtime/config/identity слой; stage-specific API и требования оставлять точными.
- [Risk] Переименование Go module path может создать большой шум в diff. → Mitigation: не включать module path rename в этот change, если реализация не докажет необходимость.
- [Risk] `APP_NAME` и `DROP_FORGE_*` могут выглядеть как два источника имени. → Mitigation: считать `APP_NAME` только override для service/log identity, а продуктовый дефолт держать в коде как `Drop Forge`.

## Migration Plan

1. Обновить config structs и loader: заменить общие `PROPOSAL_*` keys на `DROP_FORGE_*`, сохранить proposal-specific metadata keys.
2. Обновить wiring CLI, CoreOrch, GitManager и runner-ов на новые поля конфигурации без изменения stage behavior.
3. Обновить `.env.example`, README, docs и `architecture.md`.
4. Обновить unit/integration tests для новой схемы ENV и сообщений Drop Forge.
5. Запустить `go fmt ./...` и `go test ./...`.

Rollback: вернуть предыдущие ENV names и документацию в одном revert commit. Данные во внешних системах не мигрируются, поэтому runtime rollback сводится к возврату старой конфигурации приложения.

## Open Questions

- Нужно ли в рамках реализации переименовывать CLI path `cmd/orchv3` и имя бинаря?

  Recommended option: оставить `cmd/orchv3` пока без изменения, потому что пользовательский запуск в Go уже документируется и переименование каталога добавит много механического шума без изменения runtime-контракта. Отдельное переименование бинаря можно сделать позже, если появится packaging/release flow.
