## Why

Заголовок PR, имя ветки и сообщение коммита, которые раннер создаёт под proposal-задачу из Linear, всегда получаются вида `"<PRTitlePrefix> Linear task:"` вместо реального названия задачи. Причина — оркестратор передаёт раннеру одну многострочную «proposal input» строку, у которой первая непустая строка — литерал-заголовок `"Linear task:"`, а раннер берёт `Title`/branch slug/commit message из первой строки этого блока. Поле `task.Title` оказывается на 4-й строке и в метаданные PR не попадает никогда.

Это коренной дефект на стыке двух компонентов: каждый по отдельности «работает», а интеграционного контракта на «как из Linear-задачи получается заголовок PR» сейчас нет, и тесты его не покрывают.

## What Changes

- **BREAKING (внутренний контракт):** `coreorch.ProposalRunner.Run(ctx, taskDescription string)` заменяется на `Run(ctx, ProposalInput)`, где `ProposalInput` несёт явные поля `Title`, `Identifier`, `AgentPrompt` (а в будущем — другие task-метаданные при необходимости).
- `proposalrunner.Runner.Run` принимает тот же `ProposalInput` и строит `prTitle`/branch slug/commit message из `Identifier` + `Title`, а агенту в качестве prompt передаёт только `AgentPrompt`.
- `coreorch.BuildProposalInput` перестаёт возвращать `string` и возвращает структурированный `ProposalInput`, где `AgentPrompt` — текущий многострочный блок (Linear task / ID / Identifier / Title / Description / Comments), а `Title`/`Identifier` — соответствующие поля задачи.
- CLI direct-режим (`go run ./cmd/orchv3 "..."`) сохраняет поведение: одна аргументная строка трактуется и как `Title`, и как `AgentPrompt`, `Identifier` остаётся пустым; раннер при пустом `Identifier` собирает PR-title только из `Title`.
- Добавляется интеграционный тест на стык `coreorch → proposalrunner`, проверяющий, что для Linear-задачи с конкретным `Title`/`Identifier` PR title начинается с `Identifier: Title`, а не с `Linear task:`.
- `BuildPRTitle`, `BuildBranchName` и существующие unit-тесты раннера обновляются под новый контракт; «парсер первой строки» (`firstLine` из `taskDescription`) удаляется как источник скрытой связи между форматом текста и метаданными.

## Capabilities

### New Capabilities
<!-- Никаких новых capability — меняем поведение существующих. -->

### Modified Capabilities
- `codex-proposal-pr-runner`: входной контракт раннера расширяется до структуры с явными `Title`/`Identifier`/`AgentPrompt`; правила формирования `prTitle`, branch name и commit message переписываются — они выводятся из `Identifier`/`Title`, а не из первой строки prompt; требования «accepts a task description string» и связанные сценарии обновляются.
- `proposal-orchestration`: «Proposal input is built from Linear task payload» переходит от «строка с identifier/title/description/comments» к «структуре `ProposalInput` с явными `Title`/`Identifier` и `AgentPrompt`, содержащим тот же task-контекст».

## Impact

- **Код:**
  - `internal/coreorch/orchestrator.go` — `ProposalRunner` интерфейс, `BuildProposalInput`, `processTask`, тесты.
  - `internal/proposalrunner/runner.go` — `Runner.Run` сигнатура, `BuildPRTitle`, `BuildBranchName`, удаление `firstLine` (или его перевод во внутренний хелпер «склеить Identifier+Title»), `BuildPRBody`.
  - `cmd/orchv3/main.go` — `singleProposalRunner` интерфейс, конструирование `ProposalInput` из CLI-аргументов / stdin, тесты `main_test.go` и `fakeSingleProposalRunner`.
  - Тесты: `internal/coreorch/orchestrator_test.go`, `internal/proposalrunner/runner_test.go`, `cmd/orchv3/main_test.go` + новый интеграционный тест на стык.
- **Внешние контракты:**
  - CLI поведение сохраняется (input/output совместимы).
  - Структура PR на GitHub меняется: title теперь содержит реальный `Identifier: Title`, ветка — slug от `Identifier-Title`, commit message — то же. Это user-visible улучшение, а не регрессия; никаких автоматизаций, парсящих текущий «Linear task:»-префикс, в репо нет.
- **Зависимости/конфигурация:** не меняются (никаких новых env, бинарей, GraphQL-запросов).
- **Риски:** изменение внутреннего интерфейса `ProposalRunner` затронет любой код, который имитирует `Run(ctx, string)` — на текущий момент это только тестовые fake'и в репо, внешних потребителей нет.
