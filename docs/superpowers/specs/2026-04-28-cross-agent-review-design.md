# Cross-Agent Review Stage — Design

**Дата:** 2026-04-28
**Статус:** Дизайн утверждён через `superpowers:brainstorming`. OpenSpec-артефакты (proposal/design/tasks/specs) пользователь генерирует отдельно через `openspec-propose` на основе этого документа.

## Why

Оркестратор сегодня автоматизирует proposal, apply и archive-стадии: каждая берёт Linear-задачи из своего ready-state, запускает агента в изолированном temp-клоне и переводит задачу в человеческий review-статус. Артефакты агента (`proposal.md`/`design.md`/`specs`, реализация, архив) попадают на review человеку без какого-либо предварительного контроля качества — все ошибки и пропуски ловятся читателем.

Следующий шаг — добавить кросс-ревью артефактов между моделями: после того как один агент произвёл артефакт, другой агент его проверяет, а результат публикуется как PR review с inline-комментариями и готовыми fix-prompt'ами в стиле CodeRabbit. Сегодня в коде один реальный `AgentExecutor` (Codex CLI), но команда параллельно интегрирует Claude — поэтому review-стадия должна быть готова к двум независимым executor-слотам с правилом «producer и reviewer никогда не совпадают в одном проходе».

## Goals / Non-Goals

**Goals:**

- Между «producer-runner закончил push» и «человек получает задачу на review» вставить автоматический контроль качества артефакта, выполняемый «противоположной» моделью.
- Использовать тот же architectural pattern, что три существующие стадии: stage-runner + Linear state как очередь + изолированный temp clone + structured logging + fakeable dependencies в тестах.
- Однозначный machine-readable producer marker, чтобы reviewer-логика не зависела от Linear-комментариев или внешних метаданных.
- Дать пользователю-читателю PR review в формате, привычном по CodeRabbit: summary + inline + готовый fix-prompt в каждой находке.
- Сохранить роль человека как единственного, кто принимает решение о следующем переходе задачи; severity информативна, не блокирует автоматически.
- AI-review feature-flag-управляемая: пустые AI-review state ID отключают четвёртую route'у целиком, и producer-runner'ы возвращаются к старым transition target'ам.

**Non-Goals:**

- Не интегрировать Claude как реальный second-executor в этой спеке — только подготовить контракт.
- Не делать автоматическое применение fix-prompt'ов; человек копирует prompt сам.
- Не вводить дискуссию с reviewer'ом, webhook'и для откликов на review, статистику качества по моделям.
- Не публиковать дублирующие комментарии в Linear — комментарии ходят только в PR.
- Не делать параллельное выполнение review (внутри monitor-pass'а — последовательно).
- Не выделять отдельный `GitManager` пакет; review продолжает вызывать git/gh через `commandrunner` напрямую.
- Не вводить ручной CLI-режим запуска review для конкретной задачи.

## Архитектура и роли

В терминологии акторов из `architecture.md`:

- `CoreOrch` получает четвёртую route в monitor-loop'е: задачи в `Need * AI Review` маршрутизируются в `ReviewRunner`.
- `ReviewRunner` — четвёртый stage-runner рядом с `proposalrunner`, `applyrunner`, `archiverunner`. Изолирует review-workflow: клон ветки, чтение артефактов и diff'а, выбор reviewer-модели по producer-trailer'у, запуск executor'а с review-prompt'ом, парсинг JSON-ответа, публикация PR-комментариев, перевод задачи в человеческий review-статус.
- `AgentExecutor` остаётся тем же контрактом — `ReviewRunner` использует его без изменений; разница только в prompt'е и в том, какой executor выбран как reviewer для данной стадии.
- `GitManager` (текущая фактическая ответственность) переиспользуется для `clone` и `git log` (вычитать producer-trailer).
- `TaskManager` переиспользуется для статусов; comment-API НЕ используется — комментарии идут только в PR.
- `Logger` — те же step-events, новый логический модуль `review`.

Границы: `ReviewRunner` не содержит деталей `gh`/`codex`/JSON-схем выше необходимого. Публикация PR-комментариев инкапсулируется в подпакете `prcommenter`, парсинг ответа — в подпакете `reviewparse`. Логика публикации не должна протекать в `CoreOrch`.

## State machine и Linear-статусы

К каждой существующей стадии добавляется промежуточный «AI review» статус. Producer-runner после успешного push переводит задачу не в человеческий review-статус, а в AI-review статус соответствующей стадии.

Новые Linear-стейты (3 штуки):

| ID env-переменной | Назначение | Переход «in» (от кого) | Переход «out» |
|---|---|---|---|
| `LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID` | очередь review для proposal | `ProposalRunner` после push PR | `ReviewRunner` → `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID` |
| `LINEAR_STATE_NEED_CODE_AI_REVIEW_ID` | очередь review для apply | `ApplyRunner` после push в ветку | `ReviewRunner` → `LINEAR_STATE_NEED_CODE_REVIEW_ID` |
| `LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID` | очередь review для archive | `ArchiveRunner` после push в ветку | `ReviewRunner` → `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID` |

Изменения в существующих runner'ах — каждый меняет ровно одну строку: целевой `MoveTask` после успешного push с `Need * Review` на `Need * AI Review`. Логика clone/exec/push/PR не трогается.

`CoreOrch` route table расширяется тремя записями:

```
ReadyToProposeStateID         -> ProposalRunner
ReadyToCodeStateID            -> ApplyRunner
ReadyToArchiveStateID         -> ArchiveRunner
NeedProposalAIReviewStateID   -> ReviewRunner(stage=Proposal)
NeedCodeAIReviewStateID       -> ReviewRunner(stage=Apply)
NeedArchiveAIReviewStateID    -> ReviewRunner(stage=Archive)
```

`ReviewRunner` принимает `stage` как явный параметр конструкции — это влияет на prompt-template, набор категорий и target-файлы. Сам runner stage-агностичен; stage-specific логика инкапсулируется в strategy-объекте `StageProfile`.

**Поведение при сбое review.** Если `ReviewRunner` упал (executor, невалидный JSON после retry, gh) — задача **не двигается** из `Need * AI Review`. Monitor подхватит её следующим тиком. Дубликаты PR-комментариев предотвращаются маркером (см. ниже). Никакого автоматического fallback в человеческий review-статус не делаем — отказ ревью должен быть видимым.

**Поведение при отклонении человеком.** Если человек откатил задачу обратно в `Ready to *`, цикл повторяется естественно: producer пушит новый коммит → новый trailer → review снова видит «противоположного» автора.

## Producer marker и выбор reviewer'а

### Запись producer'а

Каждый producer-runner добавляет в тело коммита (последний commit перед push) git trailer:

```
Produced-By: codex
Produced-Model: gpt-5-codex
Produced-Stage: proposal
```

Trailer формируется единым хелпером `agentmeta.AppendTrailer(message, producer)` в новом маленьком пакете `internal/agentmeta`, который переиспользуется всеми тремя runner'ами. Конкретные строки (`Produced-By`, `Produced-Model`, `Produced-Stage`) — стабильный контракт. `ReviewRunner` парсит их через `git log -1 --format=%B` + `git interpret-trailers --parse`. Парсер нечувствителен к регистру ключа.

Каждый из трёх существующих runner'ов получает однострочную правку: вызов `agentmeta.AppendTrailer` при формировании commit-message.

### Выбор reviewer'а

`ReviewRunner` после клонирования читает trailer **последнего коммита HEAD ветки задачи** (не всей истории) и строит:

```
producer  = trailer.Produced-By
reviewer  = config.OppositeOf(producer)
executor  = executors[reviewer.Slot]
```

Маппинг `OppositeOf` живёт в новой секции конфига:

```env
REVIEW_ROLE_PRIMARY=codex
REVIEW_ROLE_SECONDARY=codex          # сегодня тоже codex; в будущем => claude
REVIEW_PRIMARY_MODEL=gpt-5-codex
REVIEW_SECONDARY_MODEL=gpt-5
REVIEW_PRIMARY_EXECUTOR_PATH=...
REVIEW_SECONDARY_EXECUTOR_PATH=...
```

`OppositeOf("codex"==primary) → secondary`, и наоборот. Сегодня physically оба слота — это `CodexCLIExecutor`, но с разными моделями и разной маркировкой в trailer/комментариях. День, когда подключают Claude, — это перенастройка `REVIEW_ROLE_SECONDARY=claude` плюс регистрация `ClaudeCLIExecutor`, без правок в `ReviewRunner`.

### Граничные случаи

- **Trailer отсутствует** (старый коммит, ручной push) → ReviewRunner логирует warning, дефолтит producer = `unknown`, reviewer = `REVIEW_ROLE_SECONDARY`. Review проходит и публикуется с пометкой `producer unknown` в summary.
- **Trailer есть, но указывает несуществующий слот** → ошибка config-mismatch, задача остаётся в AI-review state, лог говорит «настройте `REVIEW_ROLE_*`».
- **Несколько коммитов producer'а подряд** → берём trailer самого последнего, остальные игнорируем (правило «не тот, кто только что произвёл»).

## ReviewRunner: контракт, prompt, JSON-схема

### Контракт

```go
type ReviewInput struct {
    Stage       Stage   // Proposal | Apply | Archive
    Identifier  string  // "ZIM-42"
    Title       string
    BranchName  string
    PRNumber    int
    RepoURL     string
}

type Runner interface {
    Run(ctx context.Context, in ReviewInput) (Result, error)
}
```

`coreorch.BuildReviewInput` собирает структуру из Linear-задачи и PR-аттача (как сейчас делает `BuildProposalInput`).

### Шаги Run()

1. **Clone** — temp-директория, `git clone --branch <BranchName>`, переиспользуется тот же пакет, что использует `applyrunner`.
2. **Read producer trailer** — `git log -1 --format=%B` → `agentmeta.ParseTrailer`. Определяем producer и reviewer.
3. **Collect targets** — что именно ревьюим, зависит от стадии:

   | Stage | Targets |
   |---|---|
   | `Proposal` | Полный текст всех файлов нового OpenSpec change: `openspec/changes/<change>/{proposal.md, design.md, tasks.md}`, все файлы под `specs/`. Diff не нужен. |
   | `Apply` | `git diff <merge-base>..HEAD`; полные тексты затронутых `*.go` файлов; соответствующий OpenSpec change как контекст «что должно было быть сделано». |
   | `Archive` | `git diff <merge-base>..HEAD` по архивным изменениям + полный текст архивированных spec-файлов. |

4. **Build review prompt** — stage-специфичный template + JSON-schema-инструкция.
5. **Run executor** — тот же `AgentExecutor.Execute(ctx, prompt)` контракт. Reviewer-executor выбирается по `reviewer` slot.
6. **Parse JSON** — strict, через `encoding/json` + go-схему. При невалидном JSON один retry с repair-prompt'ом «твой предыдущий ответ невалиден, верни строго JSON по схеме». Если и repair упал — ошибка, задача остаётся в AI-review state.
7. **Publish PR comments** — summary + inline (см. секцию публикации).
8. **Move task** — через `TaskManager.MoveTask` в человеческий review-state.

Размер контекста ограничивается `REVIEW_MAX_CONTEXT_BYTES` — если targets превышают порог, ReviewRunner усекает по приоритетам (auto-generated/lock-файлы → тестовые stub-данные) и помечает в summary, что часть контекста урезана.

### JSON-схема ответа reviewer'а

```json
{
  "summary": {
    "verdict": "ship-ready | needs-work | blocked",
    "walkthrough": "markdown-абзац: что произошло, ключевые наблюдения",
    "stats": { "findings": 7, "by_severity": { "blocker": 0, "major": 2, "minor": 4, "nit": 1 } }
  },
  "findings": [
    {
      "id": "F1",
      "category": "<stage-specific closed enum>",
      "severity": "blocker | major | minor | nit",
      "file": "openspec/changes/foo/proposal.md",
      "line_start": 14,
      "line_end": 18,
      "title": "Короткий заголовок проблемы",
      "message": "Markdown-описание проблемы и почему это важно",
      "fix_prompt": "Готовый prompt для агента-исполнителя"
    }
  ]
}
```

`line_start`/`line_end` могут быть `null` для general-findings — такие пойдут только в summary, не в inline.

## Категории по стадиям и severity

### Категории (закрытые enum'ы)

**Stage = Proposal:**

- `requirement_unclear` — формулировка требования допускает несколько прочтений.
- `requirement_contradicts_existing` — конфликт с уже существующей спекой в `openspec/specs/`.
- `scenario_missing` — отсутствует ключевой сценарий приёмки или граничный случай.
- `acceptance_criteria_weak` — критерии не проверяемы или общие.
- `scope_creep` — proposal расширяется за рамки, описанные в `Why`.
- `tasks_misaligned` — `tasks.md` не покрывает заявленный `What Changes`, или содержит лишнее.
- `architecture_violation` — design нарушает границы акторов из `architecture.md`.
- `nit` — стилистика, типографика, формулировки без влияния на смысл.

**Stage = Apply:**

- `spec_mismatch` — реализация не соответствует принятой спеке.
- `bug` — потенциальная ошибка поведения / падение / гонка.
- `error_handling` — ошибки проглочены, не обёрнуты с контекстом, panic/log.Fatal в библиотечном коде.
- `concurrency` — небезопасный доступ, утечка goroutine, неправильное использование контекста.
- `test_gap` — изменение поведения без теста, или тест проверяет не то, что заявлено.
- `architecture_violation` — нарушение границ из `architecture.md`.
- `idiom` — неидиоматичный Go.
- `config_drift` — новая runtime-настройка не отражена в `.env.example` или захардкожена.
- `nit` — naming, форматирование, мелочи.

**Stage = Archive:**

- `incomplete_archive` — change перенесён, но активные части (specs, design) остались в `openspec/changes/`.
- `spec_drift` — то, что архивируется как «принято», расходится с фактическим состоянием `openspec/specs/`.
- `dangling_reference` — оставленные ссылки на архивированный change в активных файлах.
- `metadata_missing` — отсутствует дата/идентификатор архива в имени директории согласно текущему паттерну (`YYYY-MM-DD-...`).
- `nit` — мелкие формальные вопросы.

### Severity

Закрытое enum `blocker | major | minor | nit`. Reviewer-prompt инструктирует:

- `blocker` — нельзя мёржить «как есть»; запуск feature/release сломается, или спека внутренне противоречива.
- `major` — нужно исправить до мёржа, но не блокирует понимание.
- `minor` — стоит починить, но допустимо отложить.
- `nit` — косметика; на усмотрение автора.

Severity **не влияет** на статус задачи (решение всегда за человеком). Влияет только на:

- сортировку в summary;
- иконку в начале inline-комментария: `🛑` blocker, `⚠️` major, `💡` minor, `🪶` nit;
- значение `verdict` в summary (`blocked` если есть blocker, `needs-work` если есть major, иначе `ship-ready`).

### Prompt-шаблоны

Три файла в `internal/reviewrunner/prompts/`:

- `proposal_review.tmpl`
- `apply_review.tmpl`
- `archive_review.tmpl`

Каждый шаблон состоит из секций (одинаковая рамка, разные тела):

1. **Роль и задача** — «ты рецензент стадии X; продьюсер — модель Y; вернёшь строго JSON по схеме Z».
2. **Стадия-специфичный контекст** — что именно ревьюим, как читать diff/файлы, какие закрытые категории доступны.
3. **Правила формирования fix_prompt** (см. ниже).
4. **JSON-схема** — буквальный текст из секции выше.
5. **Targets** — собранный пакет файлов/diff'ов.

### Правила fix_prompt

Prompt обязан быть **самодостаточным** и **исполняемым** — человек копирует его в любого агента-исполнителя без дополнительного контекста. Шаблон fix_prompt:

```
Контекст: [файл и строки].
Проблема: [одно предложение, что не так].
Задача: [одно предложение, что должно стать].
Ограничения: [правила из architecture.md / openspec / category-specific].
Acceptance: [как проверить, что починено].
```

Reviewer обязан включать в `fix_prompt` достаточно текста файла, чтобы изменение было однозначным; ссылки в стиле «см. строку 42» без процитированного фрагмента запрещены.

## Публикация PR-комментариев

### API

Inline review-комментарии нельзя сделать через `gh pr comment` (он создаёт issue-comment в PR, не привязанный к строкам). Используем GitHub Reviews API через `gh api`:

```
POST /repos/{owner}/{repo}/pulls/{number}/reviews
{
  "commit_id": "<HEAD-sha>",
  "event": "COMMENT",
  "body": "<summary markdown>",
  "comments": [
    { "path": "...", "line": 18, "side": "RIGHT", "body": "<inline markdown>" },
    ...
  ]
}
```

`event: COMMENT` важен — не аппрувим и не реквестим changes; review информативное. Один POST = одно атомарное review с summary + всеми inline'ами.

Публикация инкапсулируется в подпакете `internal/reviewrunner/prcommenter`:

```go
type PRCommenter interface {
    PostReview(ctx context.Context, in PostReviewInput) error
}
```

Реализация `GHPostReviewCommenter` использует `commandrunner` для `gh api -X POST ... --input -` со stdin'ом-payload'ом. `CoreOrch` про `gh` ничего не знает; `ReviewRunner` про конкретные флаги `gh` тоже не знает — только зовёт интерфейс.

### Формат inline-комментария

```
🛑 **[review by codex · severity: blocker · category: spec_mismatch]**

Реализация `ApplyRunner.run` перестала переводить задачу в `Code in Progress`
до запуска executor'а — спека требует это как обязательное состояние входа.

<details>
<summary>🤖 Prompt for AI Agent</summary>

Контекст: internal/applyrunner/runner.go, функция Run, строки 80–110.
Проблема: статус задачи переводится в `Code in Progress` после запуска executor'а,
а спека требует, чтобы перевод произошёл до запуска.
Задача: вынести вызов `taskManager.MoveTask(..., CodeInProgress)` перед `executor.Execute`.
Ограничения: не делать перевод дважды; обработать ошибку MoveTask отдельным шагом
с обёрткой `fmt.Errorf("move task to code in progress: %w", err)`.
Acceptance: тест `TestApplyRunnerMovesTaskBeforeExecutor` фиксирует порядок вызовов.
</details>
```

Префикс модели и категория — машиночитаемы для будущей фильтрации/тестов.

### Формат summary-комментария

```
## 🤖 Review by codex (producer: claude · stage: proposal)

**Verdict:** needs-work · **Findings:** 7 (🛑 0 · ⚠️ 2 · 💡 4 · 🪶 1)

### Walkthrough
<один абзац: что сделано, что выглядит хорошо, что вызывает вопросы>

### Findings
1. ⚠️ **F1** [requirement_unclear] proposal.md:14–18 — Заголовок проблемы
2. ⚠️ **F2** [scenario_missing] design.md:42 — ...
3. 💡 **F3** ...
...

### Tripwires
- Контекст усечён: пропущены файлы X, Y (см. REVIEW_MAX_CONTEXT_BYTES).
- Producer trailer отсутствует — reviewer выбран по REVIEW_ROLE_SECONDARY.
```

В первой версии **ссылки на inline-комментарии в summary опускаются** — это сохраняет атомарность POST'а; идентификаторы `F1, F2…` достаточны для навигации глазами.

### Идемпотентность

ReviewRunner перед POST'ом делает `GET /pulls/{n}/reviews` и ищет существующее review с маркером в первой строке body:

```
<!-- drop-forge-review-marker:codex:proposal:<HEAD-sha> -->
```

Маркер вшивается как HTML-комментарий, не виден в рендере. Если review с тем же `(reviewer-id, stage, HEAD-sha)` уже опубликовано — ReviewRunner пропускает публикацию и сразу двигает статус задачи. Если HEAD ветки сместился (новый push producer'а) — sha другой, новое review публикуется отдельно. Старые review никогда не правим/не удаляем.

## Конфигурация, тестирование, наблюдаемость

### Новые env-переменные

Все добавляются в `.env.example` без значений по умолчанию:

```
# Linear states (новые AI-review очереди)
LINEAR_STATE_NEED_PROPOSAL_AI_REVIEW_ID
LINEAR_STATE_NEED_CODE_AI_REVIEW_ID
LINEAR_STATE_NEED_ARCHIVE_AI_REVIEW_ID

# Reviewer slots
REVIEW_ROLE_PRIMARY               # текущий producer-slot, напр. codex
REVIEW_ROLE_SECONDARY             # сейчас тоже codex; в будущем => claude
REVIEW_PRIMARY_MODEL              # gpt-5-codex
REVIEW_SECONDARY_MODEL            # gpt-5
REVIEW_PRIMARY_EXECUTOR_PATH
REVIEW_SECONDARY_EXECUTOR_PATH

# Review runtime
REVIEW_MAX_CONTEXT_BYTES
REVIEW_PARSE_REPAIR_RETRIES       # 0 или 1; default 1
REVIEW_PROMPT_DIR                 # опционально
```

### Feature-flag

Если хотя бы один из трёх AI-review state ID пуст, ReviewRunner **не активируется**, и producer-runner'ы возвращаются к старым transition target'ам (`Need * Review` напрямую). Это безопасный rollout. Конфигурация валидируется при старте: либо все три AI-review state ID и оба reviewer-слота заполнены, либо все три AI-review state ID пусты. Частичная конфигурация — ошибка старта.

### Тестирование

- `internal/reviewrunner/runner_test.go` — table-driven по стадиям с фейковыми `AgentExecutor`, `PRCommenter`, `TaskManager`. Покрывает: happy path, невалидный JSON → repair → success, невалидный JSON дважды → ошибка без перевода статуса, отсутствующий trailer → fallback reviewer + warning.
- `internal/agentmeta/trailer_test.go` — table-driven парсинг trailer'ов: валидный, со смешанным регистром, отсутствующий, с лишними trailer'ами.
- `internal/reviewrunner/reviewparse/parse_test.go` — JSON-схема: валидный ответ, отсутствующее поле, неизвестная category, severity вне enum.
- `internal/reviewrunner/prcommenter/gh_test.go` — формирование payload'а, идемпотентность по маркеру, поведение при существующем review.
- `internal/coreorch/orchestrator_test.go` — расширяется три новых route'а: задачи в каждом из AI-review state'ов попадают в `ReviewRunner` с правильным `stage`.
- Тесты на producer trailer в commit message каждого из трёх существующих runner'ов.

### Наблюдаемость

В `Logger` появляется новый модуль `review` с шагами:

- `review.start` — stage, branch, PR number, producer/reviewer slots.
- `review.clone`, `review.read_targets`, `review.execute`, `review.parse`, `review.publish`, `review.move_task` — каждый со своим status/duration.
- `review.findings` — итоговый счёт по severity и category.
- `review.skipped_idempotent` — когда маркер уже есть на PR.
- `review.parse_failed` — с raw-ответом executor'а в diagnostic-поле.

### Архитектурные апдейты

- `architecture.md` получает новый раздел «Целевой Поток Review-Stage» (по образу Apply/Archive), и в «Маппинг На Текущий Код» добавляется запись про `internal/reviewrunner` и `internal/agentmeta`.
- `docs/proposal-runner.md` дополняется упоминанием AI-review этапа между push и human review.

## Risks / Trade-offs

- **Reviewer возвращает невалидный JSON regularly** → строгая JSON-схема в prompt'е, один retry с repair-prompt'ом, явный отказ ревью при повторной невалидности. Если паттерн станет регулярным, добавим стадия-специфичные few-shot примеры.
- **Inline-комментарии указывают на несуществующие строки после force-push** → review публикуется с привязкой к конкретному `commit_id` (HEAD-sha на момент review). После force-push GitHub привяжет inline к старому коммиту; маркер по sha обеспечит, что новое review запустится снова.
- **Reviewer и producer оба = Codex с одной моделью** → контракт настройки требует разные значения `REVIEW_PRIMARY_MODEL` и `REVIEW_SECONDARY_MODEL`; при их совпадении логируется warning «producer и reviewer используют одну модель — кросс-ревью теряет смысл».
- **Контекст превышает лимит модели** → `REVIEW_MAX_CONTEXT_BYTES` плюс детерминированное усечение, факт усечения видим в summary.
- **GH Reviews API rate-limit** → один POST на review, idempotency-проверка не плодит лишних запросов. При rate-limit сбое review остаётся не опубликованным, задача в AI-review state, monitor подхватит снова.
- **Дубликаты review при сбое после POST'а до MoveTask** → idempotency-маркер по `(reviewer, stage, HEAD-sha)`. Повторный run найдёт существующее review и сразу перейдёт к MoveTask.
- **Размытие границ архитектуры** → gh-логика только в `prcommenter`, JSON-парсер в `reviewparse`, prompts — отдельные `.tmpl` файлы.

## Migration Plan

1. Добавить `internal/agentmeta` с `AppendTrailer`/`ParseTrailer` и table-driven тестами.
2. Точечно встроить `agentmeta.AppendTrailer` в commit-формирование трёх существующих runner'ов; добавить тесты на presence trailer'а.
3. Расширить `internal/config` тремя новыми state ID и reviewer-slot переменными; обновить `.env.example`.
4. Расширить `taskmanager.ManagedStateIDs` тремя новыми AI-review state ID.
5. Реализовать `internal/reviewrunner` (runner + reviewparse + prcommenter + prompts) с полным набором table-driven тестов на фейковых executor/PRCommenter/TaskManager.
6. Добавить `ReviewRunner` route'у в `CoreOrch` с тестами на маршрутизацию по трём AI-review state ID и feature-flag поведением.
7. Поменять transition target в трёх existing runner'ах с `Need * Review` на `Need * AI Review`.
8. Wiring в `cmd/orchv3` — фабрика executor'ов с двумя слотами, конструирование `ReviewRunner` со stage-инстансами.
9. Обновить `architecture.md`, `docs/proposal-runner.md`.
10. `go fmt ./...` + `go test ./...`.

**Rollback:** очистить три AI-review state ID в env. Это автоматически возвращает существующие runner'ы к прямому переходу в human review state без правок в коде. Кодовый rollback — revert этой спеки.

## Open Questions

- **Локализация PR review-комментариев.** Сегодня вся внешняя коммуникация в проекте — на русском (Linear-комментарии, OpenSpec тексты). Recommended: PR review публикуем на русском по умолчанию (соответствие AGENTS.md) с английскими keyword'ами в machine-readable метках (`severity: blocker`, `category: bug`). Если в команде появится non-RU читатель, добавим переключатель в env.
- **Ручной запуск review.** Out of Scope в первой версии. Добавим, если появится регулярная потребность диагностики.

## Что осознанно НЕ делаем (YAGNI)

- Re-review при ручном правке reviewer'ом — оставляем только идемпотентность по HEAD-sha.
- Comment-on-comment (дискуссия с reviewer'ом).
- Автоматический фикс по fix_prompt — выведено из scope (решение всегда за человеком).
- Веб-UI / дашборд для агрегации ревью.
- Механизм «приглушения» категорий (whitelist/blocklist) — добавим, если появится реальная потребность.
- Cross-PR сравнение или статистика качества по моделям.

## Затронутые OpenSpec capabilities (для последующего `openspec-propose`)

- **New:** `review-orchestration` — четвёртая стадия оркестрации, контракт ReviewRunner, producer trailer, выбор reviewer'а, публикация PR review, идемпотентность.
- **Modified:** `proposal-orchestration` — расширение orchestration runtime четвёртой route'ой; producer-runner'ы переводят задачу в AI-review state и встраивают producer trailer.
- **Modified:** `linear-task-manager` — три новых AI-review state ID используются как input-очереди review-стадии и transition-targets producer-runner'ов.
