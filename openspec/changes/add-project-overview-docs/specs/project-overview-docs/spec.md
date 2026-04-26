## ADDED Requirements

### Requirement: Overview document explains project purpose and scope
Проект SHALL содержать обзорный документ в `docs/`, который коротко объясняет, что такое `orchv3`, какую проблему решает proposal-stage orchestration и какой scope поддерживается в текущей версии.

#### Scenario: New reader wants a high-level explanation
- **WHEN** читатель открывает обзорный документ без предварительного контекста
- **THEN** документ объясняет назначение проекта, его роль в proposal-stage и текущие ограничения без погружения в код

### Requirement: Overview document describes key actors and task lifecycle
Обзорный документ SHALL описывать ключевые компоненты proposal-stage workflow и путь managed Linear-задачи от отбора до перехода в review state.

#### Scenario: Reader studies the proposal-stage flow
- **WHEN** читатель знакомится с верхнеуровневым workflow проекта
- **THEN** документ перечисляет `TaskManager`, `CoreOrch`, `AgentExecutor` и `GitManager` и объясняет, как задача проходит путь от получения из Linear до публикации proposal PR

### Requirement: Overview document supports navigation to detailed references
Обзорный документ SHALL ссылаться на более детальные документы, когда читателю нужен operational или архитектурный уровень деталей.

#### Scenario: Reader needs deeper details after the overview
- **WHEN** обзорного описания недостаточно для следующего шага
- **THEN** документ направляет читателя к `README.md`, `architecture.md`, `docs/proposal-runner.md` и `docs/linear-task-manager.md`
