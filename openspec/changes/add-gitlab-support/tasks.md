## 1. Конфигурация provider-а

- [ ] 1.1 Добавить в `config.ProposalRunnerConfig` поля выбранного Git provider-а и пути к GitLab CLI.
- [ ] 1.2 Загрузить `PROPOSAL_GIT_PROVIDER` с default `github` и `PROPOSAL_GLAB_PATH` с code-defined default `glab`.
- [ ] 1.3 Обновить validation так, чтобы `PROPOSAL_GH_PATH` требовался только для GitHub, а `PROPOSAL_GLAB_PATH` только для GitLab.
- [ ] 1.4 Добавить table-driven tests для GitHub default, GitLab mode, пустого CLI path выбранного provider-а и неподдержанного provider-а.
- [ ] 1.5 Обновить `.env.example` новыми ключами без значений.

## 2. GitManager provider operations

- [ ] 2.1 Ввести внутренний provider dispatcher/adapter в `internal/gitmanager`, сохранив текущий внешний контракт runner-ов.
- [ ] 2.2 Сохранить текущие GitHub команды `gh pr view/create/comment` и покрыть regression tests после refactor-а.
- [ ] 2.3 Добавить GitLab create MR через `glab mr create --source-branch --target-branch --title --description --yes`.
- [ ] 2.4 Добавить GitLab resolve source branch через `glab mr view <url> --output json` с парсингом source branch из JSON.
- [ ] 2.5 Добавить GitLab final comment через `glab mr note create <url> --message <body>` и сохранить skip для пустого body.
- [ ] 2.6 Расширить parser review request URL fixtures для GitLab MR URL и смешанного CLI output.
- [ ] 2.7 Обновить ошибки и structured log module/message так, чтобы они явно называли выбранный provider и failed operation.

## 3. Runner integration

- [ ] 3.1 Обновить `proposalrunner` tests, чтобы proposal workflow создавал GitHub PR по default и GitLab MR при `PROPOSAL_GIT_PROVIDER=gitlab`.
- [ ] 3.2 Обновить `applyrunner` tests для резолва branch source через GitLab MR URL.
- [ ] 3.3 Обновить `archiverunner` tests для резолва branch source через GitLab MR URL.
- [ ] 3.4 Проверить, что branch-name input по-прежнему не вызывает provider CLI в Apply/Archive.
- [ ] 3.5 Проверить, что orchestration продолжает передавать review request URL в Linear без provider-specific обработки.

## 4. Документация

- [ ] 4.1 Обновить README с GitHub/GitLab prerequisites, provider selection и auth requirements.
- [ ] 4.2 Обновить `docs/proposal-runner.md` с GitLab mode, `glab` prerequisite и provider-specific output behavior.
- [ ] 4.3 Обновить `docs/linear-task-manager.md`, если текст про PR branch source жестко привязан к `gh`.
- [ ] 4.4 Обновить `architecture.md`, потому что изменение затрагивает границу ответственности `GitManager` и provider-specific операции.

## 5. Проверка

- [ ] 5.1 Запустить `go fmt ./...`.
- [ ] 5.2 Запустить `go test ./...`.
- [ ] 5.3 Запустить OpenSpec status/validation для `add-gitlab-support` и убедиться, что proposal готов к apply.
