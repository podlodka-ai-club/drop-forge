## Why

Apply и Archive запускаются после human review, поэтому комментарии в Linear часто содержат уточнения, правки и решения, которые агент должен учитывать при выполнении. Сейчас это поведение нужно закрепить как обязательный контракт: комментарии задачи должны подтягиваться в prompt для Apply и Archive так же надежно, как это уже ожидается для Proposal.

## What Changes

- Уточнить контракт orchestration input для Apply: `AgentPrompt` должен содержать идентификатор задачи, заголовок, описание и комментарии Linear.
- Уточнить контракт orchestration input для Archive: `AgentPrompt` должен содержать идентификатор задачи, заголовок, описание и комментарии Linear.
- Зафиксировать fallback для задач без комментариев: prompt остается валидным и явно сообщает, что комментариев нет.
- Добавить проверяемые сценарии для нескольких комментариев, пустых комментариев и отсутствующего описания.
- Не менять Linear API contract: `TaskManager` уже обязан возвращать описание и комментарии задачи.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `proposal-orchestration`: уточняется требование к построению prompt для Apply и Archive из payload Linear-задачи, включая комментарии.

## Impact

- `internal/coreorch`: построение `ApplyInput` и `ArchiveInput`, общий формат `AgentPrompt`, unit tests.
- `internal/applyrunner` и `internal/archiverunner`: без изменения публичного executor lifecycle; они продолжают получать уже подготовленный prompt.
- `openspec/specs/proposal-orchestration`: delta-spec для обязательного включения comments в Apply/Archive prompts.
