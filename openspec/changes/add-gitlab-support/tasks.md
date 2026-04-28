## 1. Конфигурация provider-а

- [x] 1.1 Добавить в `config.ProposalRunnerConfig` поля выбранного Git provider-а и пути к GitLab CLI.
- [x] 1.2 Загрузить `PROPOSAL_GIT_PROVIDER` с default `github` и `PROPOSAL_GLAB_PATH` с code-defined default `glab`.
- [x] 1.3 Обновить validation так, чтобы `PROPOSAL_GH_PATH` требовался только для GitHub, а `PROPOSAL_GLAB_PATH` только для GitLab.
- [x] 1.4 Добавить table-driven tests для GitHub default, GitLab mode, пустого CLI path выбранного provider-а и неподдержанного provider-а.
- [x] 1.5 Обновить `.env.example` новыми ключами без значений.

## 2. GitManager provider operations

- [x] 2.1 Ввести внутренний provider dispatcher/adapter в `internal/gitmanager`, сохранив текущий внешний контракт runner-ов.
- [x] 2.2 Сохранить текущие GitHub команды `gh pr view/create/comment` и покрыть regression tests после refactor-а.
- [x] 2.3 Добавить GitLab create MR через `glab mr create --source-branch --target-branch --title --description --yes`.
- [x] 2.4 Добавить GitLab resolve source branch через `glab mr view <url> --output json` с парсингом source branch из JSON.
- [x] 2.5 Добавить GitLab final comment через `glab mr note create <url> --message <body>` и сохранить skip для пустого body.
- [x] 2.6 Расширить parser review request URL fixtures для GitLab MR URL и смешанного CLI output.
- [x] 2.7 Обновить ошибки и structured log module/message так, чтобы они явно называли выбранный provider и failed operation.

## 3. Runner integration

- [x] 3.1 Обновить `proposalrunner` tests, чтобы proposal workflow создавал GitHub PR по default и GitLab MR при `PROPOSAL_GIT_PROVIDER=gitlab`.
- [x] 3.2 Обновить `applyrunner` tests для резолва branch source через GitLab MR URL.
- [x] 3.3 Обновить `archiverunner` tests для резолва branch source через GitLab MR URL.
- [x] 3.4 Проверить, что branch-name input по-прежнему не вызывает provider CLI в Apply/Archive.
- [x] 3.5 Проверить, что orchestration продолжает передавать review request URL в Linear без provider-specific обработки.

## 4. Документация

- [x] 4.1 Обновить README с GitHub/GitLab prerequisites, provider selection и auth requirements.
- [x] 4.2 Обновить `docs/proposal-runner.md` с GitLab mode, `glab` prerequisite и provider-specific output behavior.
- [x] 4.3 Обновить `docs/linear-task-manager.md`, если текст про PR branch source жестко привязан к `gh`.
- [x] 4.4 Обновить `architecture.md`, потому что изменение затрагивает границу ответственности `GitManager` и provider-specific операции.

## 5. Проверка

- [x] 5.1 Запустить `go fmt ./...`.
- [x] 5.2 Запустить `go test ./...`.
- [x] 5.3 Запустить OpenSpec status/validation для `add-gitlab-support` и убедиться, что proposal готов к apply.
