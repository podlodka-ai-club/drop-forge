## Context

В проекте уже есть `internal/proposalrunner.Runner`, который закрывает общий цикл: валидирует вход, загружает конфигурацию, создает temp workspace, клонирует репозиторий, запускает Codex CLI, проверяет diff, коммитит изменения, пушит ветку и создает PR через `gh`. CLI `cmd/orchv3` сейчас трактует переданный текст как описание задачи для `openspec-propose`.

Apply workflow должен использовать тот же внешний контур, но другой вход и другое состояние git перед запуском Codex: оператор передает имя ветки с готовым OpenSpec proposal, clone должен перейти на эту ветку, а Codex должен выполнить `openspec-apply` уже из нее. Новая логика должна сохранить простоту текущего Go-кода и не вводить заранее отдельный job scheduler или полноценный GitHub API-клиент.

## Goals / Non-Goals

**Goals:**

- Добавить apply-сценарий, принимающий proposal branch name и возвращающий URL PR с реализацией.
- Сохранить текущее proposal-поведение и обратную совместимость CLI, если оператор запускает существующий сценарий.
- Переиспользовать общие шаги proposal/apply: temp workspace, clone, command runner, Codex execution, status check, commit, push, PR create, logging, cleanup.
- Явно разделить различия workflow: входная валидация, git checkout proposal branch перед apply, Codex prompt, PR title/body и branch prefix.
- Оставить тесты без реального GitHub, Codex CLI и network через существующий fake command runner.

**Non-Goals:**

- Не реализовывать архивирование OpenSpec change после apply.
- Не выполнять merge proposal PR или implementation PR.
- Не добавлять очередь задач, web API или параллельный worker pool.
- Не заменять `git`, `codex` и `gh` собственными клиентами.
- Не хранить токены, URL окружений или значения runtime-настроек в репозитории.

## Decisions

### Обобщить runner вокруг workflow options

Текущий `Runner.Run(ctx, taskDescription)` оставить как proposal entrypoint для совместимости. Добавить apply entrypoint, например `RunApply(ctx, proposalBranch string)`, который вызывает общий приватный workflow с настройками:

- `kind`: `proposal` или `apply`;
- `input`: описание задачи или имя proposal branch;
- `beforeCodex`: дополнительные git-команды перед Codex;
- `promptBuilder`: builder prompt для нужного skill;
- `branchBuilder`, `titleBuilder`, `bodyBuilder`: metadata для PR.

Альтернатива - скопировать `Run` в отдельный apply runner. Это быстрее, но почти весь код совпадет и тесты будут поддерживать две версии одного workflow. Общий приватный workflow лучше соответствует задаче про reuse и снижает риск расхождения proposal/apply.

### Apply стартует из переданной proposal-ветки

После `git clone` apply workflow должен выполнить checkout переданной ветки в clone workspace и только затем запускать Codex. Ветка для результатов реализации не создается до Codex apply step. Это важно, потому что `openspec-apply` должен видеть proposal artifacts в том виде, в котором они находятся на proposal branch.

Для первой версии использовать простой и наблюдаемый путь:

- `git clone <repo> <clone-dir>`;
- `git checkout <proposal-branch>`;
- `codex exec --sandbox danger-full-access --cd <clone-dir> -` со stdin prompt для `openspec-apply`;
- после успешного Codex и непустого diff создать implementation branch из текущего состояния, commit/push и PR.

Альтернатива - `git clone --branch <proposal-branch>`. Отдельный `git checkout` проще тестировать как явный шаг и дает более понятную ошибку, если ветка отсутствует.

### Prompt для Codex apply минимален и детерминирован

Добавить `BuildApplyCodexPrompt(proposalBranch string)`, который явно просит использовать skill `openspec-apply` и сообщает, что входной контекст уже находится в клоне proposal branch. Prompt не должен передавать shell-интерполируемые фрагменты; branch name остается обычным stdin-текстом.

Формат запуска Codex CLI остается тем же, что у proposal: `codex exec --sandbox danger-full-access --cd <clone-dir> -`. Это сохраняет текущую локальную совместимость и не добавляет новый runtime-параметр для argv-шаблона.

### Конфигурация: общие CLI пути, отдельные apply defaults

Пути к `git`, `codex`, `gh`, repository URL, remote name, cleanup-temp и base branch остаются общими. Для PR metadata apply нужен отдельный branch prefix и title prefix, чтобы результаты не смешивались с proposal PR:

- `APPLY_BRANCH_PREFIX`, например default в коде `codex/apply`;
- `APPLY_PR_TITLE_PREFIX`, например default в коде `OpenSpec apply:`.

Если реализация выберет более общий `RunnerConfig`, `.env.example` все равно должен перечислять все поддерживаемые ключи без значений. Процесс environment должен сохранять приоритет над `.env`.

Альтернатива - переиспользовать `PROPOSAL_BRANCH_PREFIX` и `PROPOSAL_PR_TITLE_PREFIX`. Это меньше ключей, но PR по реализации будут выглядеть как proposal PR, что ухудшит диагностику.

### CLI mode выбрать явно

Текущий CLI принимает свободный текст как proposal task. Чтобы не ломать этот сценарий, apply лучше включать явным флагом или subcommand:

- существующее: `orchv3 "описание задачи"` запускает proposal;
- новое: `orchv3 apply <proposal-branch>` запускает apply.

Subcommand проще для оператора и не конфликтует с task description, где может быть слово `apply`. Если позже появятся новые workflow, CLI можно расширять тем же способом.

### PR base для apply

Implementation PR должен создаваться относительно proposal branch, потому что код реализации основан на этой ветке и должен включать принятые OpenSpec artifacts. Поэтому для apply `gh pr create --base <proposal-branch> --head <implementation-branch>` является целевым поведением.

Если проекту нужен PR реализации в `main`, это можно добавить отдельной настройкой позже. Сейчас входом apply является proposal branch, и base PR от нее делает dependency явной.

## Risks / Trade-offs

- [Proposal branch отсутствует или недоступна локальному clone] -> вернуть ошибку с контекстом `git checkout proposal branch`, логировать stdout/stderr checkout.
- [Apply может не создать изменений] -> переиспользовать текущую проверку `git status --short` и возвращать ошибку без пустого PR.
- [CLI parsing может сломать старый proposal запуск] -> покрыть тестом legacy path `orchv3 "<task>"` и новый path `orchv3 apply <branch>`.
- [Общий workflow станет слишком параметризованным] -> держать options приватными и минимальными; публично оставить понятные `Run` и `RunApply`.
- [PR base от proposal branch может отличаться от ожиданий отдельного проекта] -> задокументировать поведение; не добавлять настройку до реальной необходимости.
- [Новые ENV-ключи могут рассинхронизироваться с `.env.example`] -> обновить config tests и шаблон переменных в том же изменении.
