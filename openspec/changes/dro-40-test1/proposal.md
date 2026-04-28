## Why

Linear-задача DRO-40 (`Test1`) не содержит описания или комментариев, но она прошла в proposal-поток и должна давать полезный, проверяемый результат вместо пустого или неоднозначного change. Это нужно как smoke-проверка того, что OpenSpec-процесс корректно фиксирует минимальный контекст задачи и явно отделяет подтвержденные требования от отсутствующих.

## What Changes

- Добавить OpenSpec capability для smoke-проверки proposal generation по Linear-задачам без описания.
- Зафиксировать, что такой proposal должен сохранять traceability к Linear ID, identifier и title.
- Зафиксировать, что при отсутствии описания и комментариев proposal не должен придумывать runtime-поведение или требовать изменений в Go-коде.
- Добавить минимальные критерии приемки для проверки созданных OpenSpec-артефактов.
- **BREAKING**: нет.

## Capabilities

### New Capabilities

- `proposal-smoke-test`: правила и критерии для проверки OpenSpec proposal-пайплайна на минимальной Linear-задаче без описания.

### Modified Capabilities

- Нет.

## Impact

- `openspec/changes/dro-40-test1`: новый proposal, design, tasks и capability spec.
- Runtime Go-код, CLI, Linear API, конфигурация и `.env.example` не меняются в рамках этой задачи.
- `architecture.md` не требует обновления, так как взаимодействие компонентов и orchestration flow не меняются.
