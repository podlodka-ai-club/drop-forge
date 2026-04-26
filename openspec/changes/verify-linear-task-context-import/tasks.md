## 1. Проверка текущего поведения

- [ ] 1.1 Проверить `internal/coreorch/orchestrator.go`, что `BuildProposalInput` формирует явные секции с ID, Identifier, Title, Description и Comments.
- [ ] 1.2 Проверить, что пустые description/comments остаются валидными и не ломают proposal input.

## 2. Тестовое покрытие DRO-28

- [ ] 2.1 Добавить или уточнить unit-тест в `internal/coreorch/orchestrator_test.go` для Linear payload: ID `f5f622b6-b706-4d83-acec-b4a59876ea30`, identifier `DRO-28`, title `Тестовая задача`, description `Проверка как подтягиваетеся описание`.
- [ ] 2.2 В том же тесте проверить, что комментарий `Проверка как тянутся комменты` попадает во вход proposal runner вместе с автором и timestamp, если они доступны.
- [ ] 2.3 Убедиться, что тест проверяет именно строку, передаваемую `ProposalRunner.Run`, а не только промежуточную модель task manager.

## 3. Документация и верификация

- [ ] 3.1 При необходимости обновить документацию proposal orchestration / proposal runner, если фактический формат input не описывает description/comments.
- [ ] 3.2 Выполнить `go fmt ./...`.
- [ ] 3.3 Выполнить `go test ./...`.
