## Why

Linear task `DRO-43` has only the title `ТЕст 3` and no description or comments. Today такой payload формально проходит в proposal workflow, но агент получает слишком мало предметного контекста и может создать бессодержательный или произвольный OpenSpec proposal.

Нужно сделать поведение явным: недостаточно описанные задачи должны останавливаться до запуска proposal runner и получать понятный комментарий в Linear, чтобы человек дополнил требования.

## What Changes

- Добавить preflight-проверку качества контекста для задач в `Ready to Propose`.
- Считать задачу недостаточно описанной, если у нее нет содержательного описания и нет содержательных комментариев, даже если есть title/identifier.
- Для таких задач не запускать proposal runner, не создавать proposal PR и не переводить задачу в proposal review.
- Публиковать в Linear короткий комментарий с просьбой добавить цель, ожидаемое поведение и критерии приемки.
- Логировать skip/failure как структурированное orchestration-событие с идентификатором задачи и причиной.
- Покрыть поведение unit-тестами без реального Linear API, GitHub CLI, Codex CLI и сети.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: добавить requirement на preflight-проверку достаточности Linear-контекста перед запуском proposal runner.
- `linear-task-manager`: уточнить requirement на публикацию комментариев для managed tasks, чтобы orchestration мог оставлять actionable feedback по недостаточно описанным задачам.

## Impact

- Затронутые пакеты: `internal/coreorch` или текущий пакет orchestration flow, `internal/taskmanager`, тесты этих пакетов.
- Внешнее поведение: часть задач из `Ready to Propose` больше не будет автоматически получать proposal PR, если контекст пустой.
- Новые runtime-переменные не требуются.
- `architecture.md` не требуется обновлять, если реализация ограничится локальной preflight-проверкой и существующим `TaskManager.AddComment` контрактом.
