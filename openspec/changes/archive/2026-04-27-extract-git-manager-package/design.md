## Context

Архитектура уже выделяет `GitManager` как одного из пяти внутренних акторов, но текущий код держит его ответственность внутри runner-ов. `proposalrunner` сам создает clone workspace, проверяет `git status`, создает branch, делает commit/push, создает PR через `gh` и публикует comment. `applyrunner` и `archiverunner` независимо повторяют clone, branch resolution через `gh pr view`, checkout, status, add/commit/push.

Это изменение затрагивает несколько runner-пакетов и меняет границу ответственности, поэтому нужен отдельный дизайн. Внешнее поведение proposal/apply/archive stages не должно измениться: меняется организация кода и тестовая граница, а не пользовательский workflow.

## Goals / Non-Goals

**Goals:**
- Создать пакет `internal/gitmanager` как единую точку для git/gh lifecycle операций.
- Убрать прямую сборку команд `git clone`, `checkout`, `status`, `add`, `commit`, `push` и `gh pr ...` из `proposalrunner`, `applyrunner` и `archiverunner`.
- Сохранить существующие форматы команд, логи модулей `git`/`github`, оборачивание ошибок и правила no-changes.
- Сохранить testability: unit-тесты должны подменять command runner, temp dir функции и время без реальных git/GitHub вызовов.
- Обновить `architecture.md`, потому что `GitManager` станет реализованным внутренним сервисом.

**Non-Goals:**
- Не менять runtime-конфигурацию, `.env.example`, имена env-переменных или CLI-поведение.
- Не менять контракт `AgentExecutor` и формат команд `codex exec`.
- Не добавлять новый внешний git/GitHub SDK: пакет должен работать через существующий `internal/commandrunner`.
- Не вводить scheduler, retry/backoff или параллельную обработку задач.
- Не объединять `proposalrunner`, `applyrunner` и `archiverunner` в один общий runner.

## Decisions

1. **`GitManager` будет доменным сервисом поверх `commandrunner`, а не заменой `commandrunner`.**

   `internal/commandrunner` остается низкоуровневым адаптером запуска процессов. Новый пакет должен говорить языком операций репозитория: `Clone`, `StatusShort`, `Checkout`, `CheckoutNewBranch`, `CommitAllAndPush`, `CreatePullRequest`, `ResolvePullRequestBranch`, `CommentPullRequest`. Это делает runner-ы короче и сохраняет техническую деталь запуска процессов в одном нижнем слое.

   Альтернатива: вынести только `runLoggedCommand` в общий пакет. Это уменьшило бы копипасту логирования, но не оформило бы ответственность `GitManager` и оставило бы workflow git/gh в runner-ах.

2. **Workspace lifecycle должен жить в `GitManager`, включая temp dir creation/cleanup.**

   Proposal, Apply и Archive одинаково создают временную директорию, clone-dir `repo`, логируют сохранение или удаление temp dir. Новый пакет должен предоставить операцию создания isolated workspace и cleanup-hook/метод, чтобы runner-ы не дублировали temp lifecycle.

   Альтернатива: оставить temp lifecycle в runner-ах, а в `GitManager` вынести только команды внутри clone. Это проще для первого шага, но оставляет значимую часть repository/workspace ответственности вне `GitManager`.

3. **Stage-specific metadata остается в runner-ах.**

   `GitManager` не должен знать, что такое Linear task, OpenSpec Apply/Archive или правила построения proposal title. Runner-ы продолжают строить `branchName`, `prTitle`, `prBody` и commit message, а `GitManager` исполняет операции с этими значениями.

   Альтернатива: перенести построение branch/title/commit message в `GitManager`. Это увеличит связность пакета с proposal/apply/archive semantics и усложнит повторное использование.

4. **PR URL parsing и branch resolution переходят в `gitmanager`.**

   Это часть GitHub CLI взаимодействия и сейчас находится рядом с `gh pr create/view/comment`. После выделения пакета все `gh`-операции и parsing их результата должны быть сосредоточены в одном месте.

   Альтернатива: оставить `parsePRURL` в `proposalrunner`, но тогда runner сохранит знание о формате `gh pr create` output.

5. **Инъекция зависимостей должна сохранить текущую простоту.**

   Runner-ы могут получать `GitManager` через поле интерфейсного типа для тестов. Если поле не задано, runner создает default manager из `config.ProposalRunnerConfig`, общего `commandrunner.Runner`, logger service/stdout/stderr, temp dir функций и времени. Это сохраняет текущий стиль конструктора `New(cfg)`.

   Альтернатива: внедрять manager только через конструктор и запретить zero-value fallback. Это чище с точки зрения DI, но потребует более широкого изменения wiring без практической необходимости.

## Risks / Trade-offs

- **Риск: тесты runner-ов станут слишком мокать `GitManager` и перестанут проверять команды.** → Командные последовательности нужно проверять в unit-тестах `internal/gitmanager`, а runner-тесты оставить на уровне orchestration вокруг агента и вызовов manager.
- **Риск: при переносе логирования изменится JSON Lines поток.** → Тесты должны фиксировать, что git/github stdout/stderr продолжают идти через `steplog` с модулями `git` и `github`.
- **Риск: выделение temp lifecycle в `GitManager` может изменить cleanup behavior.** → Нужно явно покрыть default preserve и `CleanupTemp=true` сценарии.
- **Trade-off: пакет будет зависеть от `config.ProposalRunnerConfig`, хотя имя config пока proposal-oriented.** → Для этого изменения это приемлемо, чтобы не раздувать миграцию конфигурации; переименование config можно сделать отдельным изменением, если появится необходимость.
- **Trade-off: Apply и Archive останутся отдельными runner-ами с похожей структурой.** → Это сохраняет простоту и stage-specific ответственность; дальнейшее объединение стоит делать только при реальном усложнении.

## Migration Plan

1. Добавить `internal/gitmanager` с интерфейсом/типом manager, workspace lifecycle и методами git/gh операций.
2. Перенести shared helpers для logged command, writer fallback, PR URL parsing и branch resolution в новый пакет или заменить их публичными методами manager.
3. Перевести `proposalrunner` на `GitManager`: clone workspace, status, new branch checkout, commit/push, PR creation и final response comment.
4. Перевести `applyrunner` и `archiverunner` на `GitManager`: clone workspace, resolve branch, checkout, status, commit/push.
5. Добавить unit-тесты `internal/gitmanager` и адаптировать существующие runner-тесты.
6. Обновить `architecture.md` в секции маппинга текущего кода.
7. Запустить `go fmt ./...` и `go test ./...`.

Rollback: поскольку изменение внутреннее, откат заключается в возврате runner-ов к прямому использованию `commandrunner`; runtime данные и внешние API не мигрируются.

## Open Questions

- Нет открытых вопросов для proposal-этапа. Решение намеренно сохраняет текущую конфигурацию и внешнее поведение.
