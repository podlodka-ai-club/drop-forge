## Context

`TaskManager` уже возвращает описание и комментарии Linear-задачи, а orchestration layer строит input для Proposal, Apply и Archive перед передачей в runner. Proposal уже имеет явный контракт на `AgentPrompt` с комментариями; для Apply и Archive в спецификациях описан общий `task context`, но не закреплено, что comments обязательно входят в фактический prompt агента.

Задача DRO-27 требует подтягивать комментарии из Linear как prompt к выполнению `archive` и `apply`. Это локальное изменение на границе `coreorch -> applyrunner/archiverunner`: runners не должны самостоятельно ходить в Linear, а должны получать уже подготовленный контекст.

## Goals / Non-Goals

**Goals:**

- Гарантировать, что Apply и Archive получают `AgentPrompt` с ID, identifier, title, description и comments из Linear-задачи.
- Сохранить единый формат task prompt для Proposal, Apply и Archive, чтобы human feedback выглядел одинаково на всех стадиях.
- Покрыть поведение unit tests на уровне `BuildApplyInput`, `BuildArchiveInput` и orchestration route.
- Явно обрабатывать отсутствие комментариев через текстовый marker в prompt.

**Non-Goals:**

- Не менять GraphQL-запросы Linear, если текущий `TaskManager` уже возвращает comments.
- Не добавлять новые переменные `.env`.
- Не менять lifecycle Apply/Archive runner: clone, checkout, Codex skill, commit и push остаются прежними.
- Не вводить отдельный prompt-builder пакет без необходимости.

## Decisions

### D1. Использовать общий builder task context в `coreorch`

Apply и Archive input должны строиться из того же Linear task payload, что и Proposal. Общий builder устраняет расхождение форматов между стадиями и не требует runners знать о структуре `taskmanager.Task`.

Альтернатива: собирать comments отдельно внутри `applyrunner` и `archiverunner`. Отброшено, потому что эти пакеты должны исполнять уже подготовленный agent prompt и не зависеть от Linear-модели.

### D2. Комментарии включаются в `AgentPrompt`, а не в отдельное поле input

Codex executor в Apply и Archive уже передает в CLI один task description/prompt. Добавление отдельного `Comments` поля потребовало бы менять контракт runner и prompt assembly в нескольких местах без реальной пользы.

Альтернатива: расширить `ApplyInput` и `ArchiveInput` полем `Comments`. Отброшено как лишнее усложнение: стадиям нужен не структурный доступ к комментариям, а полный человекочитаемый prompt.

### D3. Missing comments остаются явным marker в prompt

Если comments отсутствуют, prompt должен содержать `No comments available.` или эквивалентный marker. Это делает поведение наблюдаемым в тестах и снижает риск, что агент примет отсутствие блока comments за ошибку сборки input.

Альтернатива: пропускать секцию comments. Отброшено, потому что downstream prompt становится неоднородным и хуже диагностируется.

## Risks / Trade-offs

- [Длинные Linear-комментарии увеличивают prompt] -> Mitigation: на этом этапе передавать все доступные комментарии; лимиты, усечение и сортировка требуют отдельного продуктового решения.
- [Комментарии могут содержать устаревшие указания] -> Mitigation: сохранять порядок и metadata комментариев, чтобы агент видел контекст review; разрешение конфликтов остается частью agent reasoning.
- [Текущая реализация может уже частично делать нужное] -> Mitigation: закрепить контракт спеками и тестами, а реализацию оставить минимальной, если поведение уже соответствует требованию.

## Migration Plan

Изменение не требует миграции данных и новых runtime-настроек. После реализации достаточно прогнать `go fmt ./...`, `go test ./...` и `openspec status --change pull-linear-comments-into-stage-prompts`.

Rollback: откатить изменения в `coreorch` prompt tests/specs; runtime state и Linear payload не меняются.

## Open Questions

Нет открытых вопросов для proposal stage. Возможное будущее решение о лимитировании длинных комментариев не блокирует DRO-27.
