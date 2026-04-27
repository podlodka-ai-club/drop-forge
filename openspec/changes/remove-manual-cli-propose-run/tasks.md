## 1. Конфигурация polling

- [ ] 1.1 Добавить `PROPOSAL_POLL_INTERVAL` в структуру конфигурации, загрузку из `.env` и валидацию положительного `time.Duration`.
- [ ] 1.2 Обновить `.env.example`, добавив ключ `PROPOSAL_POLL_INTERVAL` без значения по умолчанию.
- [ ] 1.3 Добавить/обновить тесты `internal/config` для default interval, env override и invalid duration.

## 2. Continuous proposal monitor

- [ ] 2.1 Добавить тестируемый loop поверх `RunProposalsOnce`, который повторяет проходы до отмены context.
- [ ] 2.2 Покрыть loop тестами на повтор после успеха, продолжение после ошибки, ожидание configured interval и остановку по context cancellation.
- [ ] 2.3 Добавить structured logs для старта итерации, ошибки итерации и остановки monitor.

## 3. CLI runtime

- [ ] 3.1 Перевести default CLI-запуск на continuous proposal monitor с wiring существующих `TaskManager`, proposal runner и logger.
- [ ] 3.2 Удалить прямой manual proposal path, который читал task description из args/stdin и вызывал `proposalrunner.Run`.
- [ ] 3.3 Удалить или сделать unsupported публичную команду `orchestrate-proposals` для one-pass запуска.
- [ ] 3.4 Обновить CLI-тесты: default path стартует monitor, args/stdin возвращают usage error и не вызывают proposal runner напрямую.

## 4. Документация и архитектура

- [ ] 4.1 Обновить `architecture.md`: CLI больше не имеет direct single-run path, runtime запускает долгоживущий proposal monitor поверх `CoreOrch`.
- [ ] 4.2 Проверить, что публичное поведение CLI/API описано без устаревшей первой тестовой команды.

## 5. Проверки

- [ ] 5.1 Запустить `go fmt ./...`.
- [ ] 5.2 Запустить `go test ./...`.
- [ ] 5.3 Запустить OpenSpec validation/status для изменения и исправить найденные проблемы.
