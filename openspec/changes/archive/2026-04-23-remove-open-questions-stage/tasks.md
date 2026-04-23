## 1. Codex Output Capture

- [x] 1.1 Расширить сборку argv для `codex exec`, чтобы runner сохранял последнее сообщение агента через `--output-last-message` во временный файл.
- [x] 1.2 Добавить helper для чтения и нормализации сохраненного последнего сообщения Codex после завершения команды.

## 2. Proposal Runner Comment Flow

- [x] 2.1 Заменить публикацию комментария из `CollectOpenQuestions` на публикацию непустого последнего сообщения Codex.
- [x] 2.2 Удалить код, который сканирует `openspec/changes/**/*.md` ради секций `Open Questions`.
- [x] 2.3 Обновить ошибки и step-логи proposal runner под новый сценарий создания комментария.

## 3. Documentation And Verification

- [x] 3.1 Обновить `docs/proposal-runner.md`, описав, что PR comment теперь строится из последнего сообщения Codex.
- [x] 3.2 Переписать unit-тесты `internal/proposalrunner` на happy path, пустое последнее сообщение и ошибку публикации комментария из последнего сообщения Codex.
- [x] 3.3 Прогнать `go fmt ./...`.
- [x] 3.4 Прогнать `go test ./...`.
