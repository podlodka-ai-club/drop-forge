## 1. Подготовка обзорной документации

- [ ] 1.1 Собрать и сверить актуальные формулировки о назначении проекта, proposal-stage workflow и ролях компонентов по `README.md`, `architecture.md`, `docs/proposal-runner.md` и `docs/linear-task-manager.md`
- [ ] 1.2 Создать `docs/project-overview.md` с кратким описанием проекта, текущего scope, proposal-stage flow, основных внутренних ролей и ограничений текущей итерации

## 2. Обновление навигации в README

- [ ] 2.1 Обновить `README.md`, чтобы он явно ссылался на `docs/project-overview.md` как на следующий уровень документации после базового entrypoint
- [ ] 2.2 Проверить, что README сохраняет краткий формат и не дублирует весь обзорный документ или `architecture.md`

## 3. Согласованность и верификация

- [ ] 3.1 Проверить, что `docs/project-overview.md` ссылается на `README.md`, `architecture.md`, `docs/proposal-runner.md` и `docs/linear-task-manager.md` и не описывает неподдерживаемые сценарии
- [ ] 3.2 Запустить `go fmt ./...` и `go test ./...` и зафиксировать, что изменение документации не нарушило базовые проверки проекта
