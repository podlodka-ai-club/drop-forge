## Why

Текущий proposal workflow публикует дополнительный комментарий в PR только после повторного сканирования markdown-артефактов на секции `Open Questions` или `Открытые вопросы`. Это делает Go-реализацию хрупкой, потому что комментарий зависит от структуры файлов в `openspec/changes`, а не от фактического финального ответа Codex.

## What Changes

- Изменить contract proposal runner: после создания PR он должен публиковать в комментарии последнее содержательное сообщение Codex, а не собранные вопросы из markdown-файлов.
- Обновить документацию и тесты под новый источник PR comment и под отказ от markdown-сканирования секций вопросов.

## Capabilities

### New Capabilities

Пока нет новых capabilities.

### Modified Capabilities

- `codex-proposal-pr-runner`: источник и содержимое PR comment меняются с markdown-секции открытых вопросов на последнее сообщение Codex после генерации proposal.

## Impact

- `internal/proposalrunner`: захват последнего содержательного ответа Codex, отказ от `CollectOpenQuestions` как основного источника PR comment, обновление ошибок и логов.
- `openspec/specs/codex-proposal-pr-runner/spec.md`: изменение требований к PR comment.
- `docs/proposal-runner.md` и unit-тесты proposal runner: обновление описания и проверок под новый workflow.
