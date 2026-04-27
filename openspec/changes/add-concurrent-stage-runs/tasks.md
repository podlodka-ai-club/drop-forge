## 1. CoreOrch concurrency

- [ ] 1.1 Обновить `RunProposalsOnce`, чтобы после `GetTasks` eligible proposal, Apply и Archive tasks запускались в отдельных goroutine.
- [ ] 1.2 Добавить синхронизацию завершения через стандартную библиотеку Go (`sync.WaitGroup`, mutex для error collector) без внешних зависимостей.
- [ ] 1.3 Собирать ошибки всех failed goroutine и возвращать aggregated error после завершения всех запущенных задач.
- [ ] 1.4 Сохранить per-task порядок внутри `processProposalTask`, `processApplyTask` и `processArchiveTask` без изменения runner interfaces.
- [ ] 1.5 Сохранить skip-логи и no-ready логи для задач, которые не запускаются как proposal, Apply или Archive.

## 2. Tests

- [ ] 2.1 Сделать fake task manager и fake runner'ы в `internal/coreorch` tests безопасными для конкурентной записи.
- [ ] 2.2 Заменить тесты, проверяющие глобальный последовательный порядок между tasks, на проверки concurrent start и per-task ordering.
- [ ] 2.3 Добавить тест, что pass ждет завершения всех goroutine, включая медленную задачу после быстрой.
- [ ] 2.4 Добавить тест, что ошибка одной задачи не отменяет соседнюю уже запущенную задачу.
- [ ] 2.5 Добавить тест, что несколько ошибок из разных goroutine возвращаются как aggregated error с контекстом каждой задачи.

## 3. Verification

- [ ] 3.1 Запустить `go fmt ./...`.
- [ ] 3.2 Запустить `go test ./...`.
- [ ] 3.3 Запустить `openspec status --change add-concurrent-stage-runs` и убедиться, что change готов к apply.
