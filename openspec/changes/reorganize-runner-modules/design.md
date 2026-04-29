## Context

Сейчас runner-логика разложена по трем соседним пакетам: `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`. Каждый пакет содержит свой `Runner`, свой `AgentExecutor` contract, свой `CodexCLIExecutor`, свой `logged_command.go`, свой `writerOrDiscard` и свой wiring `GitManager`. При этом фактические отличия stage-ов меньше, чем объем копирования:

- proposal запускает Codex с `openspec-propose`, создает новую ветку, PR и комментарий с финальным ответом агента;
- apply запускает Codex с `openspec-apply-change`, работает в существующей branch/PR branch и пушит commit без нового PR;
- archive запускает Codex с `openspec-archive-change`, работает аналогично apply, но с archive prompt и commit prefix.

Архитектурная граница уже описана через `AgentExecutor` и `GitManager`, поэтому изменение должно укрепить эту границу, а не заменить orchestration flow новым уровнем планировщика.

## Goals / Non-Goals

**Goals:**

- Сгруппировать runner-пакеты под единой директорией, чтобы runner-layer был виден как отдельная область кода.
- Убрать дублирование общих файлов между proposal/apply/archive runner-ами.
- Сохранить stage-specific поведение в малых пакетах или файлах: prompt, commit/PR metadata, branch policy, error context.
- Устранить зависимость apply/archive от `proposalrunner` ради metadata helper-ов.
- Сохранить тестируемость без реального Codex CLI, GitHub CLI, network и Linear API.
- Обновить `architecture.md` после переноса границ.

**Non-Goals:**

- Не менять публичное CLI-поведение и state routing в `CoreOrch`.
- Не добавлять новые `.env`-переменные.
- Не менять протокол `codex exec` и формат логов.
- Не вводить внешний workflow engine, scheduler, DI framework или новый abstraction слой поверх `GitManager`.
- Не менять бизнес-семантику proposal/apply/archive переходов.

## Decisions

### Единая runner-директория

Runner-пакеты переносим под общую область, например `internal/runners/proposal`, `internal/runners/apply`, `internal/runners/archive`, а общие части размещаем рядом в `internal/runners/agent`, `internal/runners/codex`, `internal/runners/runlog` или одном компактном `internal/runners/shared`, если код остается небольшим.

Альтернативы:

- Оставить пакеты на текущем уровне `internal/*runner` и добавить только `internal/runnercommon`. Это меньше diff, но хуже отвечает задаче "перенести раннеры в отдельную папку" и оставляет runner-layer размазанным по `internal`.
- Сделать один пакет `internal/runners` со всеми stage-ами. Это уберет импорты между runner-пакетами, но быстро смешает proposal/apply/archive ответственность.

### Общий Codex executor с stage profile

Codex CLI запуск выделяем в общий компонент, который принимает stage profile: service/module, prompt builder, last-message filename, error label и флаг "читать финальный ответ". Stage runner передает только stage-specific настройки и получает общий `AgentExecutionResult`.

Альтернативы:

- Оставить три `CodexCLIExecutor` и вынести только `codexArgs`. Это сохраняет большую часть копирования.
- Сделать интерфейс prompt template engine. Сейчас prompts статичны и малы, поэтому отдельный template layer будет преждевременной абстракцией.

### Общий контракт agent execution

`AgentExecutionInput`, `AgentExecutionResult` и `AgentExecutor` должны быть едиными для всех stage-ов. `FinalMessage` остается опциональным: proposal использует его для PR comment, apply/archive игнорируют.

Альтернативы:

- Оставить разные result-типы. Это формально точнее для apply/archive, но не дает практической пользы и поддерживает дублирование.

### Общая metadata-логика без зависимости на proposal runner

Функции display name/title/slug/commit message выносим из proposal package в общий runner metadata helper. Proposal, apply и archive используют его с разными prefix/policy, не импортируя друг друга.

Альтернативы:

- Продолжать импортировать `proposalrunner` из apply/archive. Это дешевле, но связывает stage-и через случайную helper-функцию.

### Общий workflow для existing-branch stage-ов

Apply и archive имеют одинаковую форму workflow: validate input, clone, resolve branch from explicit branch or PR URL, checkout, run agent, require non-empty status, commit and push to same branch. Эту последовательность можно вынести в общий helper для existing-branch runner-ов, оставив stage profile для validation labels, prompt и commit prefix.

Альтернативы:

- Не выносить workflow, ограничившись Codex/logging dedup. Это снижает риск большого refactor-а, но оставляет наиболее опасное копирование apply/archive.
- Сделать общий workflow и для proposal. Proposal отличается созданием ветки, PR и comment, поэтому лучше не притягивать его к apply/archive helper-у.

## Risks / Trade-offs

- [Risk] Большой перенос пакетов может сломать импорты в `coreorch`, `cmd/orchv3` и тестах. → Миграция должна идти механически: сначала перенос пакетов и импорты, затем выделение общих частей, затем тесты.
- [Risk] Общий Codex executor может скрыть stage-specific ошибки за слишком универсальными настройками. → Stage profile должен быть маленьким и явным: prompt builder, last-message file, error label, final-message behavior.
- [Risk] Вынос apply/archive workflow может сделать тесты менее точечными. → Сохранить stage-level unit tests на публичный `Run`, а common workflow покрыть отдельными table-driven tests.
- [Risk] Новая структура увеличит число пакетов. → Не дробить на большее число подпакетов, чем нужно для реального переиспользования; если общий код мал, держать его в одном `shared` пакете.

## Migration Plan

1. Перенести stage packages в новую runner-директорию и обновить импорты без изменения логики.
2. Вынести общий `AgentExecutor` contract и адаптировать stage fakes/tests.
3. Вынести `runLoggedCommand`, `writerOrDiscard`, Codex args и last-message reading в общий компонент.
4. Вынести metadata helper-ы из proposal package и заменить apply/archive imports.
5. Вынести apply/archive existing-branch workflow, если после шагов 1-4 дублирование остается очевидным и тесты показывают одинаковый сценарий.
6. Обновить `architecture.md`.
7. Запустить `go fmt ./...` и `go test ./...`.

Rollback: refactor не меняет runtime data или external state; при проблеме можно откатить перенос пакетов и вернуть stage-local implementations без миграции данных.

## Open Questions

- Насколько дробить common runner area: один `shared` пакет или несколько маленьких пакетов (`agent`, `codex`, `metadata`, `workflow`). Рекомендуемый старт: один `shared` пакет, затем разделять только если package API станет размытым.
