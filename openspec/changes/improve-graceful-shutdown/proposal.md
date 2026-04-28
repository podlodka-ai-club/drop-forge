## Why

Задача DRO-47 требует доработать graceful shutdown для долгоживущего orchestration monitor. Сейчас `CoreOrch` уже умеет завершать loop по отмене контекста, но CLI запускает monitor от `context.Background()`, поэтому системные сигналы не дают предсказуемого управляемого завершения процесса.

## What Changes

- CLI будет создавать signal-aware root context и инициировать остановку monitor при `SIGINT` и `SIGTERM`.
- Orchestration loop сохранит текущую семантику: после отмены контекста не стартовать новую итерацию и корректно выйти из ожидания между polling-проходами.
- Один уже начатый orchestration pass должен завершаться управляемо: in-flight task runners получают отмененный context, `RunProposalsOnce` дожидается всех запущенных goroutine и возвращает агрегированную ошибку, если зависимости вернули ошибку отмены.
- CLI будет логировать запрос на shutdown и результат завершения в существующем JSON Lines формате.
- Тесты покроют signal-driven shutdown на уровне CLI и context cancellation на уровне `CoreOrch`.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: CLI runtime должен инициировать graceful shutdown по OS signal и не запускать новые polling-итерации после отмены.

## Impact

- `cmd/orchv3/main.go`: root context, signal handling, shutdown logging, тестируемая точка внедрения context/signal behavior.
- `internal/coreorch`: уточнение и тесты поведения loop/pass при отмене context.
- `openspec/specs/proposal-orchestration`: delta requirements для signal-driven graceful shutdown.
- Новых внешних runtime-зависимостей не требуется; достаточно стандартной библиотеки Go (`os/signal`, `syscall`, `context`).
