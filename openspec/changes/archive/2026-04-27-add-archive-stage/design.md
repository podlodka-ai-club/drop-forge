## Context

Текущий runtime уже маршрутизирует managed Linear tasks в proposal и Apply. Proposal создает OpenSpec change в отдельной ветке и PR, Apply в отдельном временном клоне переключается на ветку PR, запускает `openspec-apply-change`, коммитит и пушит изменения в ту же ветку. В конфигурации и `TaskManager` уже есть state IDs для archive-route: `Ready to Archive`, `Archiving in Progress`, `Need Archive Review`, но `CoreOrch` пока не вызывает Archive executor.

Archive-стадия должна завершать тот же OpenSpec lifecycle для задачи после code review: взять задачу из `Ready to Archive`, работать в правильной ветке task PR, запустить OpenSpec Archive skill и отправить результат на archive review.

## Goals / Non-Goals

**Goals:**

- Обработать задачи из `LINEAR_STATE_READY_TO_ARCHIVE_ID` в том же orchestration monitor, что proposal и Apply.
- Перед запуском Archive переводить задачу в `LINEAR_STATE_ARCHIVING_IN_PROGRESS_ID`, а после успешного push - в `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`.
- Изолировать git/Codex/OpenSpec Archive lifecycle за тестируемым runner-контрактом.
- Использовать ту же branch source модель, что Apply: concrete branch name из payload задачи или PR URL, по которому runner может определить head branch.
- Работать только во временном клоне, не мутируя локальный checkout оператора.

**Non-Goals:**

- Не создавать новый PR для Archive-этапа.
- Не менять OpenSpec archive semantics и не реализовывать архивирование без `openspec-archive-change`.
- Не закрывать/мержить task PR автоматически.
- Не добавлять отдельный планировщик или новый публичный CLI-режим.
- Не выделять общий `GitManager`, пока повторение Apply lifecycle остается небольшим и понятным.

## Decisions

### D1. Archive runner как отдельный контракт

`CoreOrch` должен зависеть от минимального интерфейса:

```go
type ArchiveRunner interface {
	Run(ctx context.Context, input archiverunner.ArchiveInput) error
}
```

`ArchiveInput` должен содержать task identity, title, agent prompt/context, PR URL и optional branch name. Это повторяет границу Apply: orchestration layer знает о Linear states и task payload, runner знает о git, GitHub CLI и Codex.

Альтернатива: переиспользовать `ApplyRunner` с mode flag. Отброшено, потому что prompt, commit message и no-change diagnostics у Archive отличаются, а общий mode увеличит ветвление в уже понятном runner'е.

### D2. Archive route добавляется в существующий pass

Один проход monitor загружает managed tasks и последовательно маршрутизирует их по текущему state ID:

1. `Ready to Propose` -> proposal runner.
2. `Ready to Code` -> Apply runner.
3. `Ready to Archive` -> Archive runner.
4. Остальные managed states логируются как skipped.

Последовательная обработка сохраняет текущие свойства: простые логи, понятные failure boundaries и отсутствие параллельных git/Codex процессов в первом slice.

Альтернатива: отдельный archive loop. Отброшено, потому что `TaskManager` уже загружает все managed input queues, а отдельный loop усложнит backoff, логи и координацию без явной выгоды.

### D3. Branch source остается частью task payload

Archive должен идти в той же ветке задачи, где proposal и Apply уже создали OpenSpec artifacts и code changes. Поэтому Archive input строится из первого детерминированного branch source в `task.PullRequests`: сначала concrete branch name, затем PR URL. Если оба отсутствуют, task не переводится в `Archiving in Progress`, runner не вызывается, а ошибка явно указывает на отсутствующий branch source.

Альтернатива: вычислять ветку из Linear title/identifier. Отброшено, потому что proposal branch содержит timestamp и slug, а единственным надежным источником остается PR attachment/branch metadata.

### D4. Archive runner повторяет минимальный git lifecycle Apply

Реализация runner'а:

1. Валидирует repository config и archive input.
2. Создает отдельную temp dir с archive-specific pattern.
3. Клонирует configured repository в `repo` внутри temp dir.
4. Определяет branch из `BranchName` или через `gh pr view <url> --json headRefName`.
5. Выполняет `git checkout <branch>`.
6. Запускает Codex CLI с prompt, который требует использовать `openspec-archive-change`.
7. Проверяет `git status --short`; no changes считается ошибкой, чтобы не двигать задачу в review без результата.
8. Выполняет `git add -A`, `git commit -m "Archive: <identifier/title>"`, `git push <remote> <branch>`.
9. Удаляет или сохраняет temp dir по существующей настройке cleanup.

Альтернатива: запускать archive напрямую в рабочем репозитории. Отброшено по той же причине, что и для Apply: temp clone защищает checkout оператора и оставляет диагностический артефакт.

### D5. Codex prompt явно выбирает OpenSpec Archive skill

Archive agent executor должен запускать Codex CLI в clone dir и передавать prompt вида: использовать `openspec-archive-change` для задачи из контекста. Если в ветке несколько активных OpenSpec changes и релевантный нельзя определить из task context, агент должен остановиться с понятной ошибкой, а не архивировать произвольный change.

Альтернатива: передавать точное имя change из Linear custom field. Отброшено для первого slice, потому что такой источник сейчас не существует; branch context и task prompt достаточно согласованы с Apply.

## Risks / Trade-offs

- [Ready-to-archive задача без PR/branch source] -> Ошибка до перевода в in-progress; оператор должен прикрепить PR или вернуть задачу на предыдущий этап.
- [Archive skill не находит нужный active change] -> Codex run должен завершиться ошибкой, задача останется в `Archiving in Progress` только если ошибка случилась после начального transition.
- [No-change archive] -> Runner считает это ошибкой. Это может требовать ручной проверки для уже заархивированного change, но защищает от ложного продвижения задачи.
- [Дублирование Apply runner lifecycle] -> Принимается ради простоты. Общий helper можно выделить позже, если появится четвертая стадия или повторение начнет мешать тестам.
- [Monitor имена остаются proposal-focused в части API] -> Можно сохранить существующие имена временно для совместимости, но новые логи и docs должны описывать runtime как orchestration monitor.

## Migration Plan

1. Добавить Archive input builder, runner interface и route в `internal/coreorch`.
2. Добавить `internal/archiverunner` по паттерну `internal/applyrunner` с fake-friendly command runner и agent executor.
3. Подключить реальный Archive runner в `cmd/orchv3`.
4. Обновить тесты `coreorch`, runner tests и при необходимости `config`/`taskmanager` тесты.
5. Обновить docs/specs, если публичное описание runtime не включает Archive.
6. Запустить `go fmt ./...`, `go test ./...`, `openspec status --change add-archive-stage`.

Rollback: не переводить задачи в `Ready to Archive` или отключить route revert'ом этого изменения. Proposal и Apply остаются отдельными route и не требуют миграции данных.

## Open Questions

- Нужно ли Archive runner добавлять PR comment с финальным сообщением Codex, как proposal runner?
  - Recommended option: не добавлять в первом slice. Archive результат виден в commit diff, а дополнительный comment можно добавить позже как observability enhancement.
