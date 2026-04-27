## Why

Оркестратор уже покрывает proposal и Apply, но последний шаг OpenSpec lifecycle остается ручным: после code review оператор должен сам запускать archive и двигать Linear-задачу дальше. Archive-стадия нужна сейчас, чтобы замкнуть основной поток `propose -> apply -> archive` в одном monitor без изменения уже работающих proposal/apply сценариев.

## What Changes

- Добавить Archive-стадию оркестрации для задач в `LINEAR_STATE_READY_TO_ARCHIVE_ID`.
- Archive-стадия должна переводить задачу в `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID` перед запуском архивирования.
- Добавить executor для Archive, который в отдельной временной директории клонирует репозиторий, переключается на правильную ветку задачи, запускает Codex с OpenSpec Archive skill, затем коммитит и пушит изменения в ту же ветку.
- После успешного Archive оркестратор должен переводить Linear-задачу в `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`.
- Сохранить существующую модель изоляции: работа происходит во временном клоне, а не в рабочем checkout оператора.
- Не менять публичный ручной CLI-режим: default runtime остается долгоживущим monitor, который маршрутизирует задачи по Linear state.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: добавить Archive route в общий orchestration runtime рядом с proposal и Apply.
- `linear-task-manager`: уточнить использование `Ready to Archive`, `Archiving in Progress` и `Need Archive Review` как input, in-progress и review transitions для Archive-стадии.

## Impact

- `internal/coreorch`: новый Archive runner interface, input builder, route по state, state transitions и failure-path тесты.
- Новый пакет или модуль рядом с `internal/applyrunner`: Archive executor с temp clone, checkout ветки, запуском OpenSpec Archive skill, git status, commit и push.
- `cmd/orchv3`: wiring реального Archive runner в default runtime.
- `internal/config` и `.env.example`: использовать существующие Linear state переменные для archive-route; при необходимости добавить только runtime-настройки Archive executor'а без секретов и без значений по умолчанию в `.env.example`.
- `openspec/specs/proposal-orchestration/spec.md` и `openspec/specs/linear-task-manager/spec.md`: обновление требований.
- `README.md`, `architecture.md` и профильные docs: обновить только если реализация меняет описанный orchestration flow или публичное поведение.
