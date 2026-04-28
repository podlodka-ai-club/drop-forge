## Context

`cmd/orchv3/main.go` запускает orchestration monitor как default runtime и передает в него `context.Background()`. В `internal/coreorch` уже есть loop, который останавливается при отмене context и не начинает следующую итерацию после cancellation, но root context процесса сейчас не связан с `SIGINT` или `SIGTERM`.

Изменение затрагивает CLI и orchestration loop, но не требует нового доменного актора: graceful shutdown остается частью runtime wiring и контракта `CoreOrch`.

## Goals / Non-Goals

**Goals:**

- Останавливать orchv3 по `SIGINT` и `SIGTERM` через отмену root context.
- Не запускать новые polling-итерации после shutdown signal.
- Передавать отмену context во все in-flight runner зависимости и дожидаться завершения уже запущенных goroutine в текущем pass.
- Логировать начало и результат shutdown в существующем JSON Lines формате.
- Покрыть поведение unit-тестами без отправки реальных OS signals процессу.

**Non-Goals:**

- Не добавлять новый scheduler, supervisor или отдельный shutdown service.
- Не реализовывать forced kill timeout для зависших внешних процессов в рамках этой задачи.
- Не менять контракты runner'ов, GitManager или TaskManager, кроме использования уже передаваемого `context.Context`.
- Не менять формат логов и не добавлять новую logging dependency.

## Decisions

1. CLI создает root context через внедряемую функцию, построенную поверх `signal.NotifyContext`.

   Рationale: это стандартный механизм Go для graceful shutdown без внешних зависимостей. Тесты смогут подменить фабрику context-а и вызвать cancel напрямую, не посылая реальные сигналы текущему процессу.

   Альтернатива: ловить сигналы в отдельной goroutine с каналом `os.Signal`. Это дает больше ручного кода и хуже тестируется, а стандартная библиотека уже закрывает сценарий.

2. `RunProposalsLoop` остается владельцем loop-семантики, а `RunProposalsOnce` сохраняет ожидание всех запущенных task goroutine.

   Рationale: текущий код уже агрегирует ошибки и дожидается `sync.WaitGroup`. После отмены context зависимости получают cancellation через существующий параметр `ctx`; loop после pass не должен начинать новую итерацию.

   Альтернатива: прерывать ожидание `RunProposalsOnce` сразу после cancellation. Это оставило бы запущенные goroutine без понятного владельца и усложнило бы состояние задач.

3. CLI интерпретирует штатное завершение monitor-а после cancellation как успешный exit code.

   Рationale: операторский `SIGINT`/`SIGTERM` не является ошибкой бизнес-логики. Ошибкой остается только failure, который вернет monitor вне штатной отмены.

   Альтернатива: возвращать ненулевой код при любом signal shutdown. Это усложнит запуск под supervisor-ами и будет выглядеть как аварийное завершение.

## Risks / Trade-offs

- In-flight runner может долго завершаться, если внешний процесс игнорирует context → в рамках задачи cancellation пробрасывается вниз, но forced timeout не добавляется; при необходимости его стоит оформить отдельной задачей.
- Если signal приходит во время долгого orchestration pass, процесс дождется завершения всех уже запущенных route goroutine → состояние задач остается последовательным, но shutdown может быть не мгновенным.
- Дополнительная точка внедрения context factory в CLI немного расширит test deps → это локальная тестируемость без новой архитектурной абстракции.
