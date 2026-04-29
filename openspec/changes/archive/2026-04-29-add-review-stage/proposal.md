## Why

Оркестратор автоматизирует proposal/apply/archive-стадии, но артефакты агента сегодня попадают на review человеку без какого-либо автоматического контроля качества — все ошибки и пропуски ловятся читателем. Команда параллельно подключает второго coding-агента (Claude); это удобный момент ввести кросс-ревью артефактов между моделями: после того как один агент произвёл артефакт, другой проверяет его и публикует структурированный PR review в стиле CodeRabbit с готовыми fix-prompt'ами.

## What Changes

- Добавить четвёртую orchestration-стадию `Review`, срабатывающую после каждой из трёх существующих producer-стадий (proposal/apply/archive) перед переводом задачи на human review.
- Ввести три новых Linear-состояния-очереди: `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID`. Producer-runner после успешного push переводит задачу в соответствующий AI-review state, а не сразу в human review.
- Добавить `ReviewRunner` — четвёртый stage-runner рядом с `proposalrunner`, `applyrunner`, `archiverunner`. Он клонирует ветку, читает producer-trailer последнего коммита, запускает «противоположную» модель как reviewer, парсит strict-JSON ответ и публикует одним POST'ом PR review через GitHub Pull Request Reviews API.
- Записывать producer на каждом проходе как git trailer в commit-message (`Produced-By`, `Produced-Model`, `Produced-Stage`). ReviewRunner читает trailer самого последнего HEAD-коммита и выбирает «не того, кто только что произвёл».
- Ввести два конфигурируемых reviewer-слота (`REVIEW_ROLE_PRIMARY`, `REVIEW_ROLE_SECONDARY`) со своими model и executor path. Сегодня оба слота физически указывают на Codex CLI с разными моделями; интеграция Claude в будущем — это смена `REVIEW_ROLE_SECONDARY` без правок в `ReviewRunner`.
- Reviewer возвращает строго JSON по фиксированной схеме: summary с verdict/walkthrough/stats и массив findings с category/severity/file/line/title/message/fix_prompt. ReviewRunner делает один retry с repair-prompt'ом при невалидном JSON; повторный сбой оставляет задачу в AI-review state без частичной публикации.
- Категории findings закрытые и стадия-специфичные (proposal: `requirement_unclear`, `scenario_missing`, …; apply: `spec_mismatch`, `bug`, …; archive: `incomplete_archive`, `spec_drift`, …). Severity (`blocker | major | minor | nit`) влияет только на иконку, сортировку и `verdict` — переход статуса не блокируется автоматически, решение всегда за человеком.
- ReviewRunner публикует одно атомарное PR review через `gh api POST /repos/{...}/pulls/{n}/reviews` с `event: COMMENT`: summary в body + inline-комментарий на каждую находку с раскрывающимся блоком `🤖 Prompt for AI Agent`. Идемпотентность по HTML-маркеру `<!-- drop-forge-review-marker:<reviewer>:<stage>:<HEAD-sha> -->` предотвращает дубликаты.
- Сделать AI-review feature-flag-управляемой: пустые AI-review state ID отключают четвёртую route'у целиком, и producer-runner'ы возвращаются к старым transition target'ам. Частичная конфигурация — ошибка старта.

## Capabilities

### New Capabilities

- `review-orchestration`: четвёртая стадия оркестрации; контракт `ReviewRunner`, формат producer-trailer, выбор reviewer-слота, сбор targets, JSON-схема ответа reviewer'а, публикация PR review через GitHub Reviews API, идемпотентность по HEAD-sha, feature-flag через AI-review state IDs.

### Modified Capabilities

- `proposal-orchestration`: расширение orchestration runtime четвёртой route'ой; producer-runner'ы (proposal/apply/archive) переводят задачу в AI-review state соответствующей стадии и встраивают producer-trailer в commit message при включённой фиче.
- `linear-task-manager`: три новых AI-review state ID становятся managed input states и transition-targets producer-runner'ов; правило all-or-nothing валидации в `LinearTaskManagerConfig.Validate()`.

## Impact

- `internal/coreorch`: новая route'а для трёх AI-review state ID, новый интерфейс `ReviewRunner`, расширенный `Config` с `AIReviewEnabled` и тремя AI-review state ID, расширенный `validate()`, изменение target state в `processProposalTask`/`processApplyTask`/`processArchiveTask` через feature-flag.
- `internal/reviewrunner` (новый пакет): runner, stage profiles, prompt templates (`prompts/*.tmpl`), targets collector, JSON parser в подпакете `reviewparse`, PR commenter с идемпотентностью в подпакете `prcommenter`, два executor'а на основе того же контракта, что в существующих runner'ах.
- `internal/agentmeta` (новый пакет): `AppendTrailer`/`ParseTrailer` для producer-trailer'а в commit message; используется тремя existing runner'ами и `ReviewRunner`.
- `internal/proposalrunner`, `internal/applyrunner`, `internal/archiverunner`: добавление поля `Producer agentmeta.Producer` на структуре Runner и однострочной интеграции `agentmeta.AppendTrailer` в commit message.
- `internal/config`: новая структура `ReviewRunnerConfig`, три AI-review state ID в `LinearTaskManagerConfig`, all-or-nothing валидация, расширение `ManagedStateIDs()`.
- `internal/taskmanager`: без изменений в коде — поведение распространяется через `LinearTaskManagerConfig.ManagedStateIDs()`; добавляются регрессионные тесты, что AI-review state ID попадают в Linear-запрос.
- `cmd/orchv3`: фабрика executor'ов с двумя слотами, конструирование `ReviewRunner`, проброс producer-данных в три producer-runner'а, передача `AIReviewEnabled` и трёх AI-review state ID в `coreorch.Config`.
- `architecture.md`: новый раздел «Целевой Поток Review-Stage» по образу Apply/Archive + обновление маппинга на код для `internal/reviewrunner` и `internal/agentmeta`.
- `docs/proposal-runner.md`: упоминание AI-review этапа между push и human review.
- `.env.example`: три AI-review state ID, два reviewer-слота (role/model/executor path), три runtime-knob'а (max context bytes, parse repair retries, prompt dir).
- Внешние зависимости: GitHub Pull Request Reviews API через `gh api`; никаких новых runtime-зависимостей помимо уже используемых (`git`, `gh`, `codex`).
