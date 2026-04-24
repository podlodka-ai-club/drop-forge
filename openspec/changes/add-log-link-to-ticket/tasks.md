## 1. Конфигурация ссылки на логи

- [ ] 1.1 Добавить в `internal/config.ProposalRunnerConfig` поле для опциональной ссылки `PROPOSAL_LOGS_URL` и загрузку этого значения из окружения.
- [ ] 1.2 Обновить валидацию и тесты `internal/config/config_test.go`, чтобы новая переменная считалась опциональной и корректно читалась из `.env` и process environment.
- [ ] 1.3 Обновить `.env.example`, добавив ключ `PROPOSAL_LOGS_URL` без значения по умолчанию.

## 2. Публикация ссылки в PR body

- [ ] 2.1 Изменить построение PR body в `internal/proposalrunner`, чтобы при непустом `PROPOSAL_LOGS_URL` в тело PR добавлялся отдельный блок со ссылкой на логи.
- [ ] 2.2 Сохранить текущий текст PR body без дополнительного блока, когда `PROPOSAL_LOGS_URL` пустой или состоит только из whitespace.
- [ ] 2.3 Дополнить `internal/proposalrunner/runner_test.go` сценариями, которые проверяют содержимое `gh pr create --body` с настроенной ссылкой и без неё.

## 3. Документация и проверка

- [ ] 3.1 Обновить `README.md` и `docs/proposal-runner.md`, описав `PROPOSAL_LOGS_URL` как опциональную ссылку, которая попадает в тело создаваемого PR.
- [ ] 3.2 Прогнать `go fmt ./...`.
- [ ] 3.3 Прогнать `go test ./...`.
