## Context

Текущий proposal runner уже содержит общий workflow: проверка задачи, загрузка конфигурации, создание temp clone, запуск agent CLI, проверка diff, commit/push, создание PR через `gh` и публикация финального ответа в PR comment. Codex сейчас встроен напрямую через `CodexPath`, `BuildCodexPrompt`, `CodexArgs`, файл `codex-last-message.txt` и log module `codex`.

Claude Code CLI поддерживает non-interactive print mode через `claude -p`, форматы вывода `text`, `json`, `stream-json` и режим пропуска permission prompts. Это достаточно близко к текущему запуску Codex, чтобы расширить runner через небольшой слой backend-ов, а не создавать отдельный workflow.

## Goals / Non-Goals

**Goals:**

- Сохранить Codex CLI как default backend и не ломать текущие ENV-настройки.
- Добавить runtime-переключатель backend-а: `codex` или `claude`.
- Добавить конфигурацию пути к Claude CLI.
- Обобщить выполнение агентного шага: prompt builder, argv builder, имя log module и способ извлечь финальный ответ.
- Для Claude запускать CLI в директории клона, передавать prompt через stdin и разрешать редактирование workspace в non-interactive режиме.
- Покрыть выбор backend-а и Claude happy path unit-тестами без реального запуска Claude.

**Non-Goals:**

- Не добавлять произвольный ENV-шаблон argv для любых CLI.
- Не поддерживать несколько agent CLI в одном запуске.
- Не менять git/gh часть workflow.
- Не добавлять очередь задач, retry-policy или параллельные agent runs.
- Не хранить токены Anthropic или Claude-настройки в коде или `.env.example`.

## Decisions

### Ввести явный agent backend вместо отдельного Claude workflow

Добавить внутри `internal/proposalrunner` небольшой слой, например `agentBackend`, который возвращает:

- стабильное имя backend-а (`codex`, `claude`);
- log module;
- путь к executable;
- argv для запуска в clone dir;
- prompt;
- путь к optional final-message file или post-processing stdout.

Альтернатива - добавить `if cfg.AgentCLI == "claude"` прямо в `Run`. Это быстрее, но увеличит связность основного workflow и оставит Codex-специфичные имена в местах, которые уже становятся общими. Backend-слой ограничит изменение агентного шага и оставит git/PR flow прежним.

### ENV-контракт

Добавить:

- `PROPOSAL_AGENT_CLI`: `codex` по умолчанию, допустимые значения `codex`, `claude`.
- `PROPOSAL_CLAUDE_PATH`: путь к Claude CLI, по умолчанию `claude`.

`PROPOSAL_CODEX_PATH` остается поддержанным и обязательным только для backend-а `codex`. `PROPOSAL_CLAUDE_PATH` валидируется только для backend-а `claude`. Это сохраняет совместимость: существующий `.env` без новых значений продолжит запускать Codex.

Альтернатива - заменить `PROPOSAL_CODEX_PATH` на общий `PROPOSAL_AGENT_PATH`. Это уменьшило бы количество ключей, но создало бы миграцию для уже настроенных окружений и хуже документировало бы разные prerequisites.

### Формат запуска Claude CLI

Начальный формат:

```bash
claude -p --output-format json --dangerously-skip-permissions
```

Runner запускает команду с `Dir=<clone-dir>` и передает prompt через stdin. `--dangerously-skip-permissions` нужен для headless workflow, потому что агент должен создавать OpenSpec-файлы без интерактивных подтверждений. Для тестируемости argv собирается отдельной функцией, как сейчас `CodexArgs`.

Финальный ответ для PR comment берется из Claude stdout:

- если stdout является JSON и содержит текстовый result/response/content field, использовать его;
- иначе использовать последний непустой stdout chunk как fallback;
- если итог пустой, пропустить PR comment так же, как сейчас.

Альтернатива - сначала использовать `--output-format text`. Это проще, но хуже для надежного извлечения финального ответа и будущей диагностики. JSON-формат лучше подходит для автоматизации, а fallback сохраняет устойчивость к отличиям версий CLI.

### Prompt contract

Сделать общий смысл prompt одинаковым для Codex и Claude: "создай complete OpenSpec proposal по задаче ниже". Для Claude явно указать, что артефакты надо писать в текущем workspace, следовать `AGENTS.md`, использовать русскую коммуникацию проекта и не начинать реализацию.

Codex prompt можно оставить функционально прежним, но переименовать helper-ы так, чтобы основная логика не зависела от слова Codex. Поведение Codex prompt не должно измениться без необходимости.

### Логирование

Логировать выбранный backend и его prompt перед запуском. Поток stdout/stderr выбранного CLI продолжать писать JSON Lines событиями через module `codex` или `claude`, чтобы диагностика оставалась читаемой и не ломала stdout с PR URL.

### Документация и tests

Обновить `docs/proposal-runner.md` и `.env.example`:

- описать `PROPOSAL_AGENT_CLI`;
- перечислить prerequisites для Codex и Claude;
- указать, что Codex остается default.

Тесты должны проверять:

- default config выбирает Codex;
- Claude config выбирает `claude` executable и ожидаемый argv;
- prompt передается через stdin;
- неизвестный backend дает validation error до filesystem side effects;
- пустой путь активного backend-а дает validation error;
- PR comment создается из финального Claude ответа или пропускается при пустом ответе.

## Risks / Trade-offs

- [Claude CLI flags may change] -> изолировать argv builder, покрыть текущий формат тестом и документировать поддерживаемую версию/режим через docs.
- [Headless Claude needs broad permissions] -> использовать `--dangerously-skip-permissions` только для выбранного Claude backend-а и явно описать prerequisite; запуск все равно ограничен temp clone целевого репозитория.
- [JSON output shape can vary between CLI versions] -> реализовать tolerant parser с fallback на последний непустой stdout.
- [Переименование Codex-специфичных helper-ов может раздуть diff] -> переименовывать только публично полезные места и не трогать git/PR flow.
- [Новые ENV-ключи могут запутать существующих пользователей] -> оставить Codex default, старый `PROPOSAL_CODEX_PATH` и обновить `.env.example` без значений.
