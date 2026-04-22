## Why

После создания OpenSpec proposal нужен такой же автоматизированный шаг реализации: оператор передает ветку с proposal, а приложение клонирует целевой репозиторий именно с этой ветки, запускает Codex CLI со skill `openspec-apply` и публикует результат обратно без создания новой ветки. Это убирает ручной переход от proposal PR к implementation-коммитам и позволяет переиспользовать уже существующий orchestration-код.

## What Changes

- Добавить apply workflow, который принимает имя ветки с OpenSpec proposal как основной вход.
- Клонировать целевой GitHub-репозиторий сразу из переданной proposal-ветки, а не из base branch.
- Запускать Codex CLI в клоне со skill `openspec-apply`, чтобы реализовать задачи из OpenSpec change.
- Не создавать новую ветку для apply workflow: изменения коммитятся и пушатся обратно в переданную proposal-ветку.
- Реиспользовать общие части proposal runner: загрузку конфигурации, временные директории, запуск внешних команд, потоковое логирование, проверку git status, commit/push и тестовые fake command runner.
- При необходимости выполнить небольшой рефакторинг текущего `internal/proposalrunner`, чтобы отделить общую инфраструктуру workflow от различий proposal/apply.
- Расширить CLI так, чтобы оператор мог явно выбрать proposal или apply сценарий без неоднозначного определения режима по одному позиционному аргументу.
- Обновить `.env.example`, документацию и тесты для нового apply workflow.

## Capabilities

### New Capabilities

- `codex-apply-runner`: Сценарий запуска OpenSpec apply через Codex CLI в клоне proposal-ветки с коммитом и push результата в эту же ветку.

### Modified Capabilities

Пока нет существующих capabilities для изменения: требования proposal runner остаются прежними, а общий рефакторинг не должен менять его публичное поведение.

## Impact

- Новый apply runner или общий orchestration-пакет рядом с `internal/proposalrunner`.
- Возможный рефакторинг общего кода для clone, Codex command, git status, commit/push, logging и temp cleanup.
- Расширение `cmd/orchv3` явными командами или режимами для `proposal` и `apply`.
- Расширение `internal/config` apply-настройками либо общей конфигурацией runner с отдельными prefix/title/prompt параметрами.
- Обновление `.env.example` без секретов и без значений по умолчанию.
- Обновление `docs/proposal-runner.md` или добавление отдельной документации для apply workflow.
- Новые Go-тесты для apply input validation, clone из proposal-ветки, prompt/argv Codex CLI, отсутствия `git checkout -b`, commit/push в переданную ветку и сохранения поведения proposal runner.
