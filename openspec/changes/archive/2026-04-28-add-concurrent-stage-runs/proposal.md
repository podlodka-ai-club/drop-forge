## Why

Сейчас orchestration monitor обрабатывает задачи proposal, Apply и Archive последовательно в одном проходе. Из-за этого длинный запуск в одной колонке блокирует задачи из других колонок, хотя стадии работают с разными Linear-задачами и отдельными runner lifecycle.

## What Changes

- Добавить параллельную обработку managed tasks в общем orchestration pass: задачи из `Ready to Propose`, `Ready to Code` и `Ready to Archive` должны запускаться в отдельных горутинах.
- Сохранить маршрутизацию по текущему Linear state ID: proposal, Apply и Archive продолжают получать только задачи своей стадии.
- Сделать ожидание завершения всех запущенных goroutine частью одного pass, чтобы monitor не начинал следующий poll до завершения текущих задач.
- Агрегировать ошибки параллельных задач и логировать контекст каждой неуспешной задачи без остановки уже запущенных задач других стадий.
- Ограничить изменение уровнем orchestration runtime: внутренний git/Codex/OpenSpec workflow proposal, Apply и Archive runner'ов не меняется.
- Добавить тесты на одновременный запуск задач из разных колонок, ожидание всех задач и агрегацию ошибок.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: заменить последовательную обработку задач в одном orchestration pass на конкурентную обработку через goroutine для proposal, Apply и Archive routes.

## Impact

- `internal/coreorch`: конкурентный запуск task processing, синхронизация завершения, агрегация ошибок, тесты на параллельность.
- `cmd/orchv3`: wiring, вероятно, остается прежним; публичный CLI-режим не меняется.
- `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`: runner-контракты не должны меняться, но orchestration tests должны продолжать подменять их fake runner'ами.
- `openspec/specs/proposal-orchestration/spec.md`: обновление требований о порядке обработки, failure boundaries и monitor pass.
