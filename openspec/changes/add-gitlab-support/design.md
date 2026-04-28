## Context

Оркестратор уже разделяет stage-specific логику и инфраструктурные git-операции: proposal/apply/archive runner-ы используют `internal/gitmanager` через узкие интерфейсы. При этом `GitManager` сейчас совмещает обычные `git` операции с GitHub-specific командами `gh pr view/create/comment`, а конфигурация содержит только `PROPOSAL_GH_PATH`.

GitLab support должен сохранить текущий workflow: Linear выбирает задачу, runner создает или обновляет ветку, review request прикрепляется к Linear, Apply/Archive продолжают работу по branch name или URL review request-а. Меняется только слой, который выполняет provider-specific операции pull/merge request.

## Goals / Non-Goals

**Goals:**
- Добавить выбор Git provider-а через `.env` без изменения orchestration state machine.
- Сохранить GitHub как default и не ломать существующие `PROPOSAL_*` настройки.
- Поддержать GitLab merge request операции через `glab`: resolve source branch, create MR, publish final agent comment.
- Сохранить тестируемость без реальных `git`, `gh`, `glab`, сети и внешних API.
- Обновить пользовательскую документацию и `.env.example`.

**Non-Goals:**
- Не добавлять прямую интеграцию с GitLab API в этой итерации.
- Не переносить Linear task management на GitLab Issues.
- Не менять модель стадий proposal/apply/archive и Linear state transitions.
- Не реализовывать provider auto-detection по URL репозитория как обязательное поведение.

## Decisions

1. Ввести явный `PROPOSAL_GIT_PROVIDER` со значениями `github` и `gitlab`.

   Rationale: URL репозитория может быть self-hosted, SSH/HTTPS и не всегда надежно указывает provider. Явный provider проще тестировать и диагностировать.

   Alternatives considered:
   - Auto-detection по `PROPOSAL_REPOSITORY_URL`: меньше конфигурации, но хуже для self-hosted инсталляций и неявных remote alias.
   - Отдельные runner-конфиги для GitHub/GitLab: больше дублирования при том, что workflow общий.

2. Сохранить `PROPOSAL_GH_PATH` и добавить `PROPOSAL_GLAB_PATH`.

   Rationale: это обратно совместимо и не требует переименования существующей переменной. Валидация должна требовать только CLI выбранного provider-а.

   Alternatives considered:
   - Универсальный `PROPOSAL_PROVIDER_CLI_PATH`: чище для новых provider-ов, но ломает текущие настройки или требует миграционного alias.

3. Оставить provider-specific операции внутри `GitManager`, но выделить внутренний provider dispatcher/adapter.

   Rationale: runner-ы уже зависят от `GitManager` как от инфраструктурной границы. Им не нужно знать, используется `gh` или `glab`; они передают title/body/base/head и получают URL или branch.

   Alternatives considered:
   - Создать новый пакет `vcsprovider`: может стать полезным позже, но сейчас добавляет архитектуру без необходимости.
   - Разнести GitHub/GitLab manager-ы по разным типам: усложнит wiring и тесты runner-ов при одинаковом внешнем контракте.

4. Использовать термин "review request" в документации и внутренних описаниях там, где поведение одинаково для GitHub PR и GitLab MR.

   Rationale: публичный Go-контракт может временно сохранить типы `PullRequest` для минимального изменения, но новые требования и docs должны быть provider-neutral.

   Alternatives considered:
   - Полностью переименовать типы и методы на `ReviewRequest`: чище, но увеличивает blast radius. Можно сделать позже отдельным refactor change.

5. GitLab mode использует `glab` CLI, а не GitLab REST API.

   Rationale: текущая система уже работает через внешние CLI и testable command runner. `glab` сохраняет этот подход и не добавляет токены/API-клиенты в код.

   Alternatives considered:
   - GitLab REST API: более контролируемый JSON contract, но потребует отдельной аутентификации, HTTP-клиента, моделей ошибок и документации токена.

## Risks / Trade-offs

- [Risk] `glab mr note create` помечен в документации GitLab CLI как experimental → Mitigation: обернуть ошибку понятным контекстом, покрыть command contract unit-тестами и оставить публикацию пустого комментария skip-поведением.
- [Risk] Разные форматы вывода `gh` и `glab` для URL → Mitigation: сохранить parser URL из plain/JSON/mixed output и добавить GitLab URL fixtures.
- [Risk] Терминология PR/MR может остаться смешанной в коде → Mitigation: docs/specs используют provider-neutral термины; кодовые переименования делать только там, где они нужны для ясности.
- [Risk] GitLab self-managed инсталляции требуют предварительной настройки `glab auth login --hostname` вне приложения → Mitigation: README/docs явно фиксируют этот prerequisite.

## Migration Plan

1. Добавить конфиг provider-а с default `github`, чтобы существующие `.env` продолжили работать.
2. Добавить `PROPOSAL_GLAB_PATH` в `.env.example` без значения.
3. Реализовать GitLab adapter в `GitManager` и покрыть его unit-тестами с fake command runner.
4. Обновить runner/config tests для GitHub default и GitLab mode.
5. Обновить README/docs с GitHub и GitLab setup.

Rollback: установить `PROPOSAL_GIT_PROVIDER=github` или удалить новую переменную, чтобы система использовала существующий GitHub path. Изменения не требуют миграции данных.

## Open Questions

- Нужен ли отдельный режим отключения финального комментария для GitLab, если `glab mr note create` окажется нестабильным в целевой инсталляции?
  - Why it matters: comment failure сейчас делает proposal workflow failed после создания review request-а.
  - Recommended option: оставить поведение одинаковым с GitHub в первой итерации и добавить feature flag только при реальной проблеме.
