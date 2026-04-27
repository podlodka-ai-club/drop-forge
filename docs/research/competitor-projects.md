# Research: проекты podlodka-ai-club

Дата сбора данных: 2026-04-27.

Источник списка репозиториев: <https://github.com/orgs/podlodka-ai-club/repositories?type=all>.
Для воспроизводимости список перечитан через GitHub REST API:
`https://api.github.com/orgs/podlodka-ai-club/repos?type=all&per_page=100&sort=full_name`.

## Ограничения

- Исследование использует только публичные материалы: README, дерево файлов GitHub, языки репозитория, `.env.example`, CI-конфиги, тестовые каталоги, документацию и доступные публичные счетчики.
- GitHub repo API возвращает `open_issues_count`, который включает issues и pull requests вместе. GitHub Search API начал возвращать `403` после первых запросов, поэтому PR/issue workflow оценивался в основном по README, шаблонам GitHub, labels/board-подходам в документации и структуре.
- Код конкурентов не копировался. В отчет перенесены только паттерны, продуктовые идеи и архитектурные наблюдения.
- Глубина анализа разная: репозитории с минимальным README или без кода отмечены как такие, где сигналы не найдены или неясны.

## Список репозиториев

| Репозиторий | URL | Основной язык | Stars | Forks | Open issues count | Последний push | Глубина |
|---|---|---:|---:|---:|---:|---|---|
| `blast-furnace` | <https://github.com/podlodka-ai-club/blast-furnace> | TypeScript | 0 | 0 | 0 | 2026-04-27 | Полная по README и дереву |
| `boiler-room` | <https://github.com/podlodka-ai-club/boiler-room> | Python | 0 | 0 | 3 | 2026-04-25 | Полная по README и дереву |
| `drop-forge` | <https://github.com/podlodka-ai-club/drop-forge> | Go | 0 | 0 | 5 | 2026-04-27 | Полная, это текущий проект |
| `gear-grinders` | <https://github.com/podlodka-ai-club/gear-grinders> | Python | 0 | 0 | 1 | 2026-04-27 | Частичная: README короткий |
| `heavy-lifting` | <https://github.com/podlodka-ai-club/heavy-lifting> | Python | 1 | 1 | 0 | 2026-04-27 | Полная по README, docs и структуре |
| `iron-press` | <https://github.com/podlodka-ai-club/iron-press> | TypeScript | 0 | 0 | 2 | 2026-04-27 | Полная по README и структуре |
| `night-shift` | <https://github.com/podlodka-ai-club/night-shift> | TypeScript | 0 | 0 | 2 | 2026-04-27 | Полная по README, OpenSpec и CI |
| `rivet-gang` | <https://github.com/podlodka-ai-club/rivet-gang> | Не определен | 0 | 0 | 1 | 2026-04-27 | Частичная: planning/docs без runtime-кода |
| `spark-gap` | <https://github.com/podlodka-ai-club/spark-gap> | Python | 1 | 0 | 2 | 2026-04-26 | Полная по README и docs |
| `steam-hammer` | <https://github.com/podlodka-ai-club/steam-hammer> | Python | 2 | 0 | 12 | 2026-04-27 | Полная по README, Go wrapper и тестам |
| `the-anvils` | <https://github.com/podlodka-ai-club/the-anvils> | Python | 0 | 0 | 0 | 2026-04-24 | Полная по README/docs/CI |
| `the-foundry` | <https://github.com/podlodka-ai-club/the-foundry> | Python | 0 | 0 | 7 | 2026-04-27 | Полная по README и структуре |
| `the-furnace` | <https://github.com/podlodka-ai-club/the-furnace> | TypeScript | 0 | 0 | 0 | 2026-04-27 | Полная по README/OpenSpec/Temporal |
| `the-smelters` | <https://github.com/podlodka-ai-club/the-smelters> | Python | 1 | 0 | 0 | 2026-04-27 | Полная по README, sample projects и CI |
| `X15` | <https://github.com/podlodka-ai-club/X15> | Не определен | 0 | 0 | 0 | 2026-04-25 | Поверхностная: README-заглушка |

Репозитории из актуального списка не пропускались. Для `X15`, `rivet-gang` и частично `gear-grinders` глубокий runtime-анализ ограничен, потому что публичные материалы почти не раскрывают рабочий поток или код.

## Сравнительная матрица

| Проект | Назначение | Стек | Структура | User workflow | Task/issue handling | Agent/automation approach | Git/PR workflow | Observability | Configuration | Tests |
|---|---|---|---|---|---|---|---|---|---|---|
| `blast-furnace` | Сервер-оркестратор для GitHub issues с очередью стадий | Node.js, TypeScript, BullMQ, Redis | `src/jobs`, `src/github`, `dist`, agent skills | Polling issues с label `ready`, затем stages `intake -> prepare -> assess -> plan -> develop -> quality -> review -> make-pr` | GitHub issue labels, run handoff JSONL | Codex CLI как worker внутри подготовленного workspace | Branch, push, PR, sync tracker state | Run JSON, handoff JSONL, queue retries | `.env.local.example`, env-only config | Неясно из дерева верхнего уровня; тесты не попали в первые сигналы |
| `boiler-room` | CLI берет задачи из GitHub Project и делегирует локальному агенту | Python, gh CLI, Claude/Copilot CLI | `boiler_room/*`, `tests`, `docs/superpowers` | `boiler-room --agent ... --project ...` обрабатывает Todo items | GitHub Project columns, issue labels, draft item tags | Claude или Copilot CLI, output contract в `.agent-runs/.../output.json` | Создает branch, commit, push, PR; failure branch остается для inspection | Output JSON по run; e2e проверяет board/PR | CLI flags, gh auth | Unit tests и e2e на реальном GitHub Project |
| `drop-forge` | Go CLI для Linear -> OpenSpec proposal/apply/archive | Go | `cmd/orchv3`, `internal/coreorch`, `internal/taskmanager`, runners, docs, openspec | Долгоживущий monitor по Linear states | Linear states и attached PR URL/branch | Codex через `AgentExecutor` для proposal/apply/archive | Clone, branch, PR для proposal; push в существующую ветку для apply/archive | Structured JSONL logs, ELK docs | `.env.example`, централизованный config | Широкие Go unit/integration tests |
| `gear-grinders` | Agent orchestrator, детали почти не раскрыты | Python | `src/gg/agents`, `analyzers`, `generators`, `knowledge`, `platforms` | Неясно | Не найдено | Есть слои agents/analyzers/generators | Неясно | Неясно | `pyproject.toml`; env-сигналы не найдены | `tests/test_analyzers.py`, `test_knowledge.py`, `test_search.py`, `test_system.py` |
| `heavy-lifting` | Backend-оркестратор с API intake и worker pipeline | Python, Flask, PostgreSQL, Docker | `src/backend`, `docs/contracts`, `docs/process`, `instration`, workers | `POST /tasks/intake`, затем worker1/2/3 | TrackerProtocol, DB tasks, PR feedback tasks | `CliAgentRunner` через `opencode run` или local runner | SCM protocol, deliver worker публикует результат/PR links | `/health`, `/stats`, token_usage, worklog docs | `.env.example`, Docker Compose, Makefile | `tests`, runtime scenarios, pre-commit hooks |
| `iron-press` | Детерминированный graph workflow для Claude Agent SDK | TypeScript, Graphology, Claude Agent SDK, Zod | `src/sdk/workflow`, `node`, `session`, `workflows`, `runs`, UI | `pnpm do <workflow> <issueId>`, resumable `--run-id` | Linear issue input, GitHub/Linear clients | Каждый node запускает headless SDK session со skill/permissions | GitHub client есть; PR flow зависит от workflow | `.runs/<runId>`, run log, Studio UI | `.env.example`, commander flags | Есть tests, часть legacy broken явно задокументирована |
| `night-shift` | GitHub Project -> OpenSpec -> implement -> review -> PR | TypeScript, Node, Codex SDK, Claude SDK | `src/stages`, `providers`, `store`, `workspace`, `openspec`, reports | Один запуск claims Ready item и проходит stages | GitHub Project, task states, reports | Provider adapters для Codex/Claude, role-specific planner/implementer/reviewer | PR template, ReportPublisher, GitHub adapter | Pretty/JSON run summary, reports, RunStore/Inspector | `.env.example`, `feature-factory.config.json` в target repo | `scripts/check.sh`, tests, dependency reports, GitHub workflow |
| `rivet-gang` | Planning repository для AI orchestration approach | Markdown/docs | `PRD.md`, `ARCHITECTURE.md`, `docs/exec-plans`, `_bmad-output` | Exec-plan workflow, runtime не найден | Неясно | Не найдено | Неясно | Документированные execution plans | Не найдено | Не найдено |
| `spark-gap` | Stateless GitHub Issues orchestrator | Python, PyGithub, uv | `orchestrator/*`, `docs/workflow.md`, `plans` | Polling issues, label state machine, pinned JSON comment | Workflow labels + pinned JSON comment in issue | Local `codex` implementer, later `claude` validator | Fresh worktree, branch, push, PR, manual merge | GitHub labels/comments как observable state | `.env.example`; PAT вне repo через token file | `tests/test_config.py`, `test_workflow.py` |
| `steam-hammer` | Issue/PR review runner с Python script и Go CLI wrapper | Python, Go, gh CLI | `scripts`, `cmd/orchestrator`, `internal/cli`, `docs`, `retro`, `tests` | Run issue mode, PR review mode, doctor diagnostics | GitHub issues, PR review comments, Jira option | Claude default, opencode optional | Branch modes, existing PR reuse, review-comment mode | Doctor `[PASS]/[WARN]/[FAIL]`, retrospectives | local/project config examples, Jira env | Много pytest по modes, staging, recovery, diagnostics + Go tests |
| `the-anvils` | Continuous agent loop с dashboard, PRD/TRIZ/readiness направлениями | Python, Claude CLI, Rich/TUI, tmux/worktrees | `docs`, `scripts`, `.github`, `config`, `examples`, `tests` | Task plan loop, parallel agents, budget/deadlock guards | JSON task plans, GitHub Projects sync docs | Claude CLI loop, self-healing/repair ideas | vNext Forge: issue -> branch -> PR; GitHub integration scripts | TUI dashboard, state store, logs, notifications | `.env.example`, `config/integrations.example.json` | CI workflows, integration/demo scripts, tests |
| `the-foundry` | Sandbox runner для issue -> PR эксперимента | Python, uv, gh | `src/foundry`, `docs/architecture`, `tests` | `uv run foundry run` по source/target repo | GitHub sandbox issues with label `agent-task` | Pipeline stages, shell/worktree helpers | Ожидаемый результат: PR в sandbox repo | Architecture docs, meeting notes | `.env.example` with source/target repo | `tests/test_implement.py`, `test_pipeline.py`, `test_state.py` |
| `the-furnace` | Temporal-based autonomous coding system | TypeScript, Temporal, PGLite/Postgres, Express | `server`, `openspec`, `build`, `.agent/.claude/.opencode` | Linear tickets -> tests -> coder -> reviewer -> PR | Linear poller schedule and state sync | Spec/coder/review agents as Temporal workflow activities | GitHub integration planned in workflow; devcontainer image manifests | Temporal UI, health endpoint, schedule/activity retries | `server/.env.example`, Docker Compose | Vitest/Supertest, Temporal smoke, devcontainer e2e |
| `the-smelters` | Multi-agent dev assistant with custom and Agno pipelines | Python, Agno, Claude/Gemini, Kotlin sample | `agents`, `agno_agents`, `projects`, `tasks`, `src`, `shared` | Seed tasks -> DB -> orchestrator, or direct Agno task run | Markdown task files, SQLite tracker, JSONL events | Custom coder/reviewer loop and Agno TDD workflow | Task sandbox worktrees with temporary git repo | Events JSONL, printer/TUI, per-project DB | `agent_config.yml`, task/project folders | GitHub workflow, Python fixture, Android sample tests |
| `X15` | Placeholder repository | Не найдено | README only | Не найдено | Не найдено | Не найдено | Не найдено | Не найдено | Не найдено | Не найдено |

## Наблюдения по проектам

- `blast-furnace` ближе всего к queue-backed service orchestration: полезны durable handoff records, фиксированная stage chain и Redis locks против повторной обработки issues.
- `boiler-room` сильный минималистичный CLI: task claiming через GitHub Project, явный JSON-контракт результата агента и e2e, который создает реальные задачи и проверяет PR/board state.
- `drop-forge` уже хорошо совпадает с нашим целевым стилем: Go, явные границы `TaskManager`, `CoreOrch`, runner-пакеты, OpenSpec-first workflow и обязательные Go tests.
- `gear-grinders` показывает потенциально полезное разделение на `agents`, `analyzers`, `generators`, `knowledge`, но публичного workflow недостаточно для надежных выводов.
- `heavy-lifting` хорошо декомпозирует backend MVP: API intake, worker pipeline, tracker/scm protocols, token accounting и mock adapters для локальной разработки.
- `iron-press` выделяется детерминированным graph engine: LLM не выбирает следующий шаг, а только возвращает typed status, который мапится на ребра графа.
- `night-shift` полезен как пример полного feature-factory цикла с provider adapters, role-specific model config, validation config в target repo и machine-readable run summary.
- `rivet-gang` полезен скорее как процессный reference: exec plans и planning artifacts, но не как runtime-конкурент.
- `spark-gap` дает простой stateless подход: состояние workflow хранится прямо в GitHub issue через labels и pinned JSON comment.
- `steam-hammer` силен diagnostics-first подходом: doctor mode, dry-run/branch modes, PR review comments mode, ретроспективы после прогонов.
- `the-anvils` показывает широкий набор UX/операционных идей: dashboard, budget/deadlock guards, state store, self-healing, readiness gate и PR composition roadmap.
- `the-foundry` полезен как небольшой скелет pipeline/worktree/state, пригодный для проверки идей без тяжелой инфраструктуры.
- `the-furnace` показывает зрелую сторону orchestration runtime: Temporal schedules, activity retries, devcontainer image manifests и Linear polling как workflow.
- `the-smelters` показывает два параллельных pipeline-подхода в одном репозитории: простой custom loop и более богатый Agno TDD flow с retry-гейтами.
- `X15` сейчас не дает технических сигналов, кроме принадлежности к инициативе.

## Повторяющиеся паттерны

- Task intake почти всегда идет из существующего tracker surface: GitHub Issues/Projects, Linear, Jira или локальные markdown task files.
- Сильные проекты отделяют orchestration state от agent output: JSONL handoff, run store, SQLite/Postgres, pinned issue comment или Temporal history.
- Agent execution чаще всего спрятан за небольшим adapter/runner contract, а конкретный CLI или SDK выбирается конфигурацией.
- Git/PR workflow обычно явно моделируется: отдельный worktree/branch, push, PR body, review feedback loop, cleanup или сохранение failed branch.
- Хорошо проверяемые проекты имеют e2e или smoke-тесты на полный happy path, а не только unit tests.
- Конфигурация почти везде вынесена в env/example files; лучшие варианты дополнительно имеют doctor/preflight checks.
- Observability полезна не только как logs: dashboards, JSON summaries, run artifacts, issue comments и retrospectives помогают разбирать неудачные agent runs.

## Shortlist идей для текущего оркестратора

| Идея | Источник | Граница проекта | Польза | Сложность | Риск | Приоритет | Follow-up OpenSpec change |
|---|---|---|---|---|---|---|---|
| Doctor/preflight command для проверки `git`, `gh`, `codex`, Linear env, repo access и writable temp dirs | `steam-hammer`, `spark-gap`, `night-shift` | CLI, configuration, `GitManager`, `TaskManager`, `AgentExecutor` | Быстрее диагностировать misconfiguration до запуска monitor | Средняя | Можно раздуть CLI проверками внешней инфраструктуры | P1 | `add-doctor-preflight-command` |
| Durable run artifact на каждый orchestration pass: input, stage transitions, outputs, PR URL, errors | `blast-furnace`, `iron-press`, `night-shift`, `the-smelters` | `CoreOrch`, `Logger`, documentation | Упрощает review, retry и postmortem без чтения только stderr | Средняя | Дублирование structured logs и вопросы хранения секретов | P1 | `add-run-artifacts` |
| Typed output contract для agent runners вместо свободного текста | `boiler-room`, `iron-press`, `night-shift` | `AgentExecutor`, runners, tests | Надежнее определять success/no-change/failure и формировать PR/comment summary | Средняя | Потребует миграции prompts и обратной совместимости | P1 | `add-agent-output-contract` |
| Validation commands per target repo before moving Linear task to review | `night-shift`, `the-smelters`, `the-furnace` | Apply runner, configuration, tests | Снижает число review PR без локальной проверки | Средняя | Нужен безопасный timeout и понятный fallback | P2 | `add-target-repo-validation` |
| Retry/rework loop для apply по review feedback или validation failure | `the-smelters`, `the-anvils`, `steam-hammer` | `CoreOrch`, `applyrunner`, `AgentExecutor` | Позволяет чинить часть failures автоматически | Высокая | Может увеличить стоимость и зациклить задачу без лимитов | P2 | `add-apply-rework-loop` |
| Stateless external progress marker в Linear comments или GitHub PR body | `spark-gap`, `boiler-room`, `night-shift` | `TaskManager`, `GitManager`, documentation | Состояние видно без доступа к логам процесса | Низкая-Средняя | Нужно избежать шумных комментариев | P2 | `publish-run-progress-to-tracker` |
| Explicit provider adapter interface, если появятся Claude/OpenCode/other runners | `night-shift`, `the-smelters`, `heavy-lifting` | `AgentExecutor` | Снижает coupling к Codex CLI при расширении | Средняя | Сейчас может быть преждевременной абстракцией | P3 | `introduce-agent-provider-adapters` |
| Temporal/BullMQ-like queue runtime для долгих и retryable workflows | `the-furnace`, `blast-furnace` | `CoreOrch`, deployment architecture | Надежные retries, visibility и concurrency control | Высокая | Сильно усложнит текущий простой Go monitor | P3 | `evaluate-durable-workflow-runtime` |
| E2E happy-path test with fake tracker/scm/agent adapters | `boiler-room`, `heavy-lifting`, `the-foundry` | tests, `TaskManager`, `GitManager`, runners | Покрывает весь route Linear -> runner -> PR/state без реальных сервисов | Средняя | Нужно аккуратно замокать внешние CLIs | P1 | `add-orchestrator-e2e-fakes` |
| Документированный failure taxonomy и ретроспективы runs | `steam-hammer`, `the-anvils`, `heavy-lifting` | documentation, `Logger` | Ускоряет улучшение prompts и runner behavior после неудачных прогонов | Низкая | Может стать ручной бюрократией без шаблона | P3 | Не требует runtime change; можно сделать docs-only |

## Проверка требований `competitor-research`

- Source scope зафиксирован: указана GitHub organization page, дата сбора и точный список 15 репозиториев.
- Для каждого репозитория указаны URL, язык/стек и публичные метрики, когда они доступны через repo API.
- Единая матрица покрывает назначение, стек, структуру, workflow, task/issue handling, automation/agent approach, git/PR workflow, observability, configuration и tests.
- Недоступные сигналы явно помечены как `Не найдено`, `Неясно` или описаны в ограничениях.
- Идеи отделены от наблюдений, приоритизированы и смэплены на текущие границы `TaskManager`, `CoreOrch`, `AgentExecutor`, `GitManager`, `Logger`, CLI, configuration, tests и documentation.
- Runtime-поведение текущего оркестратора в рамках этого research-change не изменялось; идеи, требующие поведения, вынесены в candidate follow-up OpenSpec changes.
