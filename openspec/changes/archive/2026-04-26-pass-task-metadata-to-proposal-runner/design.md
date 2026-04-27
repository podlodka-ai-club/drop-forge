## Context

`coreorch.Orchestrator` сейчас сериализует Linear-задачу в одну многострочную строку (`BuildProposalInput`) и отдаёт её `proposalrunner.Runner` через интерфейс `ProposalRunner.Run(ctx, taskDescription string)`. Раннер использует эту же строку и для агентского prompt, и для вычисления `prTitle`/branch name/commit message — последние получаются как `firstLine(taskDescription)` + slug. Поскольку `BuildProposalInput` начинает блок с литерала `"Linear task:"`, все три метаданных PR содержат этот литерал, а не реальное название задачи. CLI direct-режим (`go run ./cmd/orchv3 "..."`) от этого не страдает только потому, что туда приходит уже одна «человеческая» строка без префикса.

Сейчас в репо есть только два потребителя интерфейса `coreorch.ProposalRunner`/`singleProposalRunner`: реальный `proposalrunner.Runner` и тестовые fake'и (`recordingProposalRunner`, `fakeSingleProposalRunner`). Внешних потребителей нет, поэтому ломать сигнатуру безопасно.

## Goals / Non-Goals

**Goals:**
- Сделать так, чтобы `prTitle`, имя ветки и commit message всегда содержали реальный `Title` (и, если есть, `Identifier`) Linear-задачи.
- Убрать неявную связь «формат текста, который оркестратор кладёт первой строкой ↔ парсер первой строки в раннере». Заменить её на явный типизированный контракт, который проверяется компилятором.
- Сохранить агентский prompt (полный task-контекст с identifier/title/description/comments) без потерь — агент по-прежнему получает тот же блок текста, что и сейчас.
- Покрыть стык `coreorch → proposalrunner` интеграционным тестом, чтобы регрессия с заголовком PR ловилась автоматически.

**Non-Goals:**
- Менять поведение CLI снаружи — `stdout`/`stderr`, exit codes, env-переменные, режим `orchestrate-proposals` остаются прежними.
- Менять формат агентского prompt (содержимое, порядок полей, приветственные строки).
- Менять контракт `TaskManager`, `LinearClient`, GraphQL-запросы, схему `.env`.
- Решать смежный баг с обязательным `LINEAR_PROJECT_ID` или другие вопросы по Linear-фильтрации.

## Decisions

### D1. Новый тип `proposalrunner.ProposalInput`, заменяющий `string` в `Run`

```go
// internal/proposalrunner
type ProposalInput struct {
    Title       string // обязательно, человекочитаемое название задачи
    Identifier  string // опционально, например "ZIM-123"
    AgentPrompt string // полный task-контекст, который уходит агенту
}
```

`Runner.Run(ctx, ProposalInput)` валидирует `Title` и `AgentPrompt` (оба не пустые после `TrimSpace`), `Identifier` опционален. `coreorch.ProposalRunner` принимает тот же тип — без алиасов и адаптеров, чтобы не тащить два независимых определения.

**Альтернатива:** оставить `Run(ctx, string)` и добавить отдельный сеттер `runner.SetTaskMetadata(...)` или поля на `Runner`. Отброшено: метаданные привязаны к конкретному вызову `Run`, а не к жизненному циклу `Runner` (который в `orchestrate-proposals` переиспользуется между задачами); это вернуло бы ту же скрытую связь в другом месте.

**Альтернатива:** оставить `Run(ctx, string)` и просто переписать `BuildProposalInput`, чтобы первой строкой был `"<Identifier>: <Title>"`. Отброшено как корневой фикс — это всё та же неявная сцепка через формат текста (см. proposal). Используется только если понадобится hotfix параллельно с этим изменением.

### D2. Где живёт тип

Определяем `ProposalInput` в пакете `proposalrunner`. `coreorch` импортирует его — этот пакет уже зависит от `taskmanager`/`steplog`, и зависимость на `proposalrunner` концептуально допустима (оркестратор уже знает о существовании раннера через интерфейс). Альтернатива — вынести `ProposalInput` в отдельный shared-пакет (`internal/proposal` или подобный) — отброшена как преждевременная декомпозиция: единственный потребитель сейчас — `coreorch`, единственный исполнитель — `proposalrunner`.

### D3. Источник `prTitle`/branch/commit

Раннер строит метаданные так:

- `displayName = "Identifier: Title"` если `Identifier` непустой, иначе просто `Title`.
- `prTitle = BuildPRTitle(prefix, displayName)` — текущая логика truncation до 72 рун и опционального префикса остаётся.
- `branchName = BuildBranchName(prefix, displayName, now)` — slug строится из того же `displayName`, что обеспечивает консистентность ветка↔PR↔commit.
- commit message = `prTitle` (как сейчас).

`firstLine(value)` удаляется: больше нет источника, где бы потребовалось извлекать «первую строку из многострочного текста». Если в `Title` всё-таки попал перевод строки (бывает редко, но защититься нужно), на входе `Runner.Run` делается `strings.ReplaceAll(Title, "\n", " ")` + `TrimSpace` перед использованием в displayName.

### D4. `coreorch.BuildProposalInput` возвращает `ProposalInput`

Новая сигнатура: `func BuildProposalInput(task taskmanager.Task) proposalrunner.ProposalInput`.
- `Title` ← `strings.TrimSpace(task.Title)`; если пусто — fallback `"Untitled task"` (раннер требует непустой `Title`, а в Linear theoretically может приехать issue без title).
- `Identifier` ← `strings.TrimSpace(task.Identifier)`.
- `AgentPrompt` ← текущий многострочный блок (то, что `BuildProposalInput` возвращает сегодня), без изменений в форматировании. Это гарантирует, что агент получает идентичный prompt и поведение Codex не меняется.

### D5. CLI direct-режим

В `cmd/orchv3/main.go` после `readTaskDescription` строится `ProposalInput{Title: text, Identifier: "", AgentPrompt: text}`. То есть для прямого запуска `Title` совпадает с тем, что человек ввёл в аргументах/stdin — это ровно текущее поведение `prTitle = firstLine(text)`, только теперь явно. `singleProposalRunner` интерфейс в `main.go` обновляется под новую сигнатуру; `fakeSingleProposalRunner` в тестах правится.

### D6. Интеграционный тест на стык

В `internal/coreorch/orchestrator_test.go` добавляется тест, который:
1. Вызывает реальный `BuildProposalInput` для задачи с `Identifier="ZIM-42"`, `Title="Add foo bar"`.
2. Передаёт результат в реальный `proposalrunner.BuildPRTitle(prefix, displayName)` (или эквивалентную функцию вычисления displayName).
3. Проверяет, что итоговый `prTitle` начинается с `Identifier: Title`, а не с `Linear task:`.

Это «contract test» в одном пакете — он не требует git/codex/gh/linear и быстро ловит регрессию формата.

## Risks / Trade-offs

- **[Риск] Изменение интерфейса `ProposalRunner` ломает любой внешний код, имитирующий `Run(ctx, string)`** → Mitigation: внешних потребителей нет, в репо только тестовые fake'и; апдейтим их одним коммитом, не выпускаем как библиотеку.
- **[Риск] `task.Title` приходит из Linear с переводами строк или служебными символами** → Mitigation: нормализация в `Runner.Run` (см. D3): `\n`/`\r` → пробел, `TrimSpace`. Truncation до 72 рун в `BuildPRTitle` остаётся.
- **[Риск] `Title` пустой (Linear теоретически допускает)** → Mitigation: fallback `"Untitled task"` в `BuildProposalInput`; раннер не должен падать на этом, потому что для CLI direct-режима тоже возможен короткий ввод.
- **[Trade-off] `proposalrunner` теперь публикует тип `ProposalInput`, который импортирует `coreorch`** → принимаем направление зависимости coreorch → proposalrunner (оно уже было через интерфейс), но не наоборот.
- **[Trade-off] Тестовых правок много (3 файла + новый интеграционный тест)** → принимаем как стоимость явного контракта.
