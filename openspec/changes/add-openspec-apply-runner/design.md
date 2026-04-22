## Context

Текущий `proposalrunner` уже реализует большую часть инфраструктуры, нужной для apply: загрузку runtime-конфигурации, temp workspace, `git clone`, запуск Codex CLI через stdin, проверку `git status`, commit, push, создание PR через `gh` и step logging. Отличие apply workflow в источнике задачи и базовой ветке: вместо текстового описания задачи runner получает имя ветки с готовым OpenSpec proposal, стартует рабочую копию от этой ветки и просит Codex применить proposal через skill `openspec-apply`.

## Goals / Non-Goals

**Goals:**
- Добавить apply workflow с входом `proposalBranch`.
- Запускать Codex CLI с prompt, который явно требует использовать `openspec-apply`.
- Клонировать или переключать рабочую копию на переданную proposal-ветку до запуска Codex.
- Не создавать новую ветку до завершения Codex apply и появления изменений.
- Создавать implementation branch после проверки изменений и открывать PR с base, равным proposal-ветке.
- Переиспользовать существующую инфраструктуру proposal workflow без изменения публичного поведения proposal runner.
- Покрыть новый поток unit-тестами с fake command runner.

**Non-Goals:**
- Не реализовывать автоматический поиск proposal-ветки по PR URL, change name или OpenSpec change id.
- Не выполнять merge proposal PR или implementation PR.
- Не запускать apply напрямую внутри текущего репозитория без temp clone.
- Не добавлять новые внешние зависимости, если достаточно текущих `git`, `codex`, `gh` и стандартной библиотеки Go.

## Decisions

1. Apply runner принимает именно имя ветки, а не описание задачи.

   Это соответствует текущему требованию и оставляет discovery proposal по PR URL/change name на будущее. Валидация должна отбрасывать пустое значение до создания temp directory и запуска внешних команд.

   Альтернатива: принимать PR URL и вычислять branch через `gh pr view`. Это удобнее для пользователя, но добавляет новый внешний шаг и неоднозначность при разных remote.

2. Рабочая копия стартует от proposal-ветки, implementation PR базируется на proposal-ветке.

   Runner должен гарантировать, что Codex видит proposal artifacts из переданной ветки. После успешного Codex apply и обнаружения изменений создается новая implementation branch, которая пушится в remote; `gh pr create` вызывается с `--base <proposalBranch>` и `--head <implementationBranch>`.

   Альтернатива: создавать PR в `main`. Тогда PR будет включать и proposal artifacts, и реализацию, что ломает разделение proposal/apply и усложняет review.

3. Общие шаги выделяются вокруг реального совпадения workflow.

   Существующий proposal runner не нужно переписывать полностью. Достаточно выделить небольшие helper-функции или внутренний workflow-тип для temp lifecycle, clone, Codex invocation, git status, commit/push, PR creation и writer defaults. Proposal-specific и apply-specific части остаются отдельными: input validation, prompt builder, branch/title/body builders и PR base.

   Альтернатива: сделать один универсальный runner с mode enum. Это уменьшает дублирование, но может быстро смешать разные инварианты proposal и apply. На текущем этапе понятнее иметь отдельные entrypoints поверх общих примитивов.

4. Конфигурация apply получает отдельные prefix/title settings и переиспользует общие command paths.

   Минимальный набор новых ключей: `APPLY_BRANCH_PREFIX` и `APPLY_PR_TITLE_PREFIX`. `PROPOSAL_REPOSITORY_URL`, `PROPOSAL_REMOTE_NAME`, `PROPOSAL_CLEANUP_TEMP`, `PROPOSAL_GIT_PATH`, `PROPOSAL_CODEX_PATH` и `PROPOSAL_GH_PATH` можно переиспользовать как общие настройки текущей интеграции, чтобы не дублировать URL и пути.

   Альтернатива: ввести полностью отдельный namespace `APPLY_*` для всех полей. Это гибче, но сейчас создает лишнюю конфигурацию без отдельного runtime-сценария.

5. CLI должен явно различать proposal и apply, сохраняя старый запуск proposal.

   Рекомендуемый интерфейс: `orchv3 proposal <task description>` и `orchv3 apply <proposal-branch>`. Для обратной совместимости `orchv3 <task description>` продолжает запускать proposal workflow. Stdin можно оставить для legacy proposal; apply через stdin не обязателен на первом шаге, если CLI требует явный subcommand `apply`.

   Альтернатива: определять режим эвристикой по форме строки. Это ненадежно, потому что описание задачи и имя ветки оба являются строками.

## Risks / Trade-offs

- Риск: proposal-ветка не существует в remote. -> Mitigation: ошибка `git clone` или `git checkout` должна содержать контекст proposal branch.
- Риск: Codex apply меняет proposal artifacts и код одновременно. -> Mitigation: runner проверяет только факт изменений, а review остается в PR; scope изменений должен быть виден через `git status` и diff.
- Риск: общий workflow helper станет слишком абстрактным. -> Mitigation: выделять только повторяющиеся операции с одинаковыми инвариантами, а mode-specific builder-логику оставить в отдельных runner.
- Риск: старый CLI `orchv3 "task"` будет сломан при добавлении subcommands. -> Mitigation: добавить тест обратной совместимости legacy proposal invocation.
- Риск: implementation PR к proposal-ветке может быть непривычным для части пользователей. -> Mitigation: задокументировать stacked PR model и явно логировать `base`/`head` при создании PR.

## Migration Plan

1. Добавить apply runner и тесты, не меняя поведение существующего proposal runner.
2. Ввести subcommands `proposal` и `apply`, оставив legacy positional proposal запуск.
3. Обновить `.env.example` и документацию.
4. Прогнать `go fmt ./...` и `go test ./...`.
5. Rollback: удалить apply entrypoint и новые config keys; legacy proposal workflow остается отдельным и продолжает работать.

## Open Questions

Нет.
