## Context

Сейчас `cmd/orchv3` при наличии аргументов или stdin запускает только proposal workflow. Основная логика живет в `internal/proposalrunner`: загрузка config, создание temp-директории, `git clone`, запуск Codex CLI через `codex exec --sandbox danger-full-access --cd <clone-dir> -`, проверка `git status`, commit/push и создание PR через `gh`.

Apply workflow должен быть похожим по инфраструктуре, но отличаться по Git-потоку:

- вход - имя ветки, где уже лежит OpenSpec proposal;
- clone должен выполняться из этой ветки;
- новая ветка не создается;
- после `openspec-apply` изменения коммитятся и пушатся обратно в эту же ветку;
- `gh pr create` не нужен, потому что proposal branch уже является веткой существующего PR или рабочей веткой оператора.

Ограничения проекта сохраняются: Go, простые пакеты, runtime-настройки через `.env`, `.env.example` без значений, тестируемость без реального GitHub/Codex.

## Goals / Non-Goals

**Goals:**

- Добавить apply runner с методом, принимающим `context.Context` и имя proposal-ветки.
- Запускать Codex CLI со skill `openspec-apply` в клоне переданной ветки.
- Не выполнять `git checkout -b` и не создавать новую ветку в apply workflow.
- Коммитить результат `openspec-apply` и пушить его в переданную ветку.
- Переиспользовать существующие command runner, logger, temp cleanup и test fakes.
- Сохранить текущее поведение proposal runner.
- Сделать CLI-выбор режима явным для apply и удобным для proposal.

**Non-Goals:**

- Не реализовывать полный lifecycle OpenSpec change после apply, включая archive.
- Не создавать новый PR для apply workflow.
- Не определять proposal branch автоматически по GitHub PR.
- Не добавлять очередь задач, параллельное выполнение или API-сервер.
- Не заменять Git CLI собственным Git-клиентом.
- Не менять semantics существующего `openspec-propose` workflow.

## Decisions

### Apply runner как отдельный пакет с общими helpers

Добавить `internal/applyrunner` с типом `Runner`, похожим на `internal/proposalrunner.Runner`. Общую инфраструктуру вынести только там, где есть реальное повторение:

- writer fallback;
- temp-dir lifecycle;
- `git status --short`;
- `git add -A`, `git commit`, `git push`;
- Codex argv builder для `codex exec --sandbox danger-full-access --cd <clone-dir> -`;
- логирование команд и потоков.

Минимальный рефакторинг предпочтительнее крупного "универсального runner" с множеством hooks. Различия proposal/apply остаются в отдельных пакетах, чтобы сценарии читались явно.

Альтернатива - сразу заменить `proposalrunner` универсальным state machine. Это преждевременно для двух сценариев и увеличит риск регрессии в уже готовом proposal flow.

### Apply config отделить от proposal config

Добавить `ApplyRunnerConfig` в `internal/config` с apply-ключами:

- `APPLY_REPOSITORY_URL`;
- `APPLY_REMOTE_NAME`;
- `APPLY_COMMIT_TITLE_PREFIX`;
- `APPLY_CLEANUP_TEMP`;
- `APPLY_GIT_PATH`;
- `APPLY_CODEX_PATH`.

`APPLY_REPOSITORY_URL` обязателен для apply workflow. Остальные значения получают defaults в коде, но `.env.example` содержит только ключи без значений. Процесс environment должен иметь приоритет над `.env`, как и сейчас.

Альтернатива - использовать существующие `PROPOSAL_*` ключи. Это уменьшает число переменных, но делает конфигурацию apply неочевидной и связывает два workflow сильнее, чем нужно.

### Clone из переданной ветки

Apply workflow должен клонировать ветку командой вида:

```bash
git clone --branch <proposal-branch> --single-branch <repo-url> <clone-dir>
```

Это точнее, чем clone default branch с последующим checkout: ошибка отсутствующей ветки возникает на clone-step, логи проще, а рабочая директория сразу соответствует входному proposal branch.

Перед запуском внешних команд runner отбрасывает пустые branch names и имена с пробельными символами или ведущим `-`. Команды запускаются без shell interpolation, через `os/exec` args.

### Codex prompt для openspec apply

Сохранить тот же non-interactive формат Codex CLI:

```bash
codex exec --sandbox danger-full-access --cd <clone-dir> -
```

Prompt строится отдельной функцией и передается через stdin. В prompt явно указать skill `openspec-apply`, имя proposal branch и задачу: реализовать OpenSpec change из текущей ветки.

Не добавлять ENV-шаблон argv в первой версии: текущий proposal runner уже зафиксировал локальный формат Codex CLI, и общий builder можно покрыть тестами.

### Apply commit и push без PR

После успешного Codex CLI:

1. выполнить `git status --short`;
2. если изменений нет, вернуть contextual error;
3. выполнить `git add -A`;
4. выполнить `git commit -m <title>`;
5. выполнить `git push <remote> HEAD:<proposal-branch>`.

`git checkout -b` и `gh pr create` в apply workflow не выполняются. Возвращаемым значением runner и stdout CLI после успешного apply будет имя ветки, в которую выполнен push. Это дает скриптам стабильный, простой результат и не требует угадывать URL PR.

### CLI modes

Добавить явный режим:

```bash
orchv3 apply <proposal-branch>
orchv3 proposal <task description>
```

Для совместимости текущий запуск `orchv3 <task description>` и stdin без subcommand продолжает работать как proposal workflow. Ошибки apply должны идти в stderr с контекстом, а stdout после успеха должен содержать только имя обновленной ветки.

## Risks / Trade-offs

- [Apply может пушить в неправильную ветку при ошибочном вводе] -> валидировать branch input, логировать branch до clone и использовать явный `HEAD:<proposal-branch>` при push.
- [Proposal branch может не существовать в remote] -> ошибка должна возникнуть на `git clone --branch` и вернуться как `git clone proposal branch`.
- [Codex может не найти нужный OpenSpec change в ветке] -> prompt должен требовать `openspec-apply`; ошибка Codex возвращается как apply-step error с полным stdout/stderr в логах.
- [Рефакторинг proposal runner может сломать готовый flow] -> выносить только маленькие общие helpers и сохранить существующие proposal tests без изменения ожиданий.
- [Отдельные APPLY_* переменные дублируют PROPOSAL_*] -> явность конфигурации важнее, а common parser helpers уменьшат дублирование кода.
- [Push в существующую ветку может конфликтовать с remote updates] -> первая версия не делает rebase/force-push; обычный `git push` должен упасть, чтобы оператор увидел конфликт.
