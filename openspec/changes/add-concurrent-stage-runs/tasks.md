## 1. Конфигурация

- [ ] 1.1 Добавить `ORCH_MAX_CONCURRENT_TASKS` в загрузку конфигурации с кодовым дефолтом `2` и валидацией положительного integer.
- [ ] 1.2 Прокинуть лимит параллелизма из `config.Config` в `coreorch.Config`.
- [ ] 1.3 Обновить `.env.example`, добавив `ORCH_MAX_CONCURRENT_TASKS` без значения по умолчанию.
- [ ] 1.4 Добавить unit-тесты конфигурации для дефолта, валидного значения и невалидных значений лимита.

## 2. Конкурентное выполнение orchestration pass

- [ ] 2.1 Добавить в `internal/coreorch` внутреннюю модель routed task run с route name и функцией запуска существующего `process*Task`.
- [ ] 2.2 Переписать `RunProposalsOnce`, чтобы он сначала собирал runnable proposal/apply/archive задачи и skipped задачи, затем выполнял runnable tasks через worker pool с лимитом параллелизма.
- [ ] 2.3 Сохранить per-task порядок операций внутри `processProposalTask`, `processApplyTask` и `processArchiveTask` без изменения контрактов runner'ов.
- [ ] 2.4 Реализовать сбор ошибок из goroutines и возврат агрегированной ошибки после завершения всех запущенных task runs.
- [ ] 2.5 Добавить логирование start/done/error для concurrent task run с task ID, identifier и route name.

## 3. Тесты orchestration

- [ ] 3.1 Обновить существующие sequential-order тесты так, чтобы они проверяли маршрутизацию и per-task порядок, а не глобальный порядок всех задач.
- [ ] 3.2 Добавить тест, доказывающий, что две ready-задачи могут находиться в runner'ах одновременно при дефолтном лимите.
- [ ] 3.3 Добавить тест, доказывающий, что configured limit не позволяет выполнить больше указанного числа task runs одновременно.
- [ ] 3.4 Добавить тест, что ошибка одной задачи не останавливает независимую задачу в том же pass, а итоговая ошибка содержит контекст отказа.
- [ ] 3.5 Сделать fake `TaskManager` и fake runner'ы потокобезопасными там, где они используются concurrent тестами.

## 4. Документация и проверка

- [ ] 4.1 Обновить `architecture.md`, если итоговая реализация меняет ответственность `coreorch` или wiring runtime-компонентов.
- [ ] 4.2 Запустить `go fmt ./...`.
- [ ] 4.3 Запустить `go test ./...`.
- [ ] 4.4 Запустить `openspec status --change add-concurrent-stage-runs` и убедиться, что change готов к apply.
