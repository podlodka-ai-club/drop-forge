## Why

Текущий proposal workflow требует отдельную стадию открытых вопросов: skill просит явно выводить блок `Open Questions`, а runner затем повторно сканирует markdown-артефакты и публикует этот блок отдельным комментарием в PR. Это дублирует финальную коммуникацию Codex и делает поведение хрупким, потому что публикация комментария зависит от структуры markdown, а не от фактического итогового ответа агента.

## What Changes

- Убрать из proposal workflow зависимость от отдельной стадии `Open Questions` в финальном ответе Codex.
- Изменить contract proposal runner: после создания PR он должен публиковать в комментарии последнее содержательное сообщение Codex, а не собранные вопросы из markdown-файлов.
- Обновить локальный skill `openspec-propose`, чтобы финальная сводка не требовала отдельный блок `Open Questions` и оставляла только краткое завершение proposal.
- Обновить документацию и тесты под новый источник PR comment и под отказ от markdown-сканирования секций вопросов.

## Capabilities

### New Capabilities

Пока нет новых capabilities.

### Modified Capabilities

- `codex-proposal-pr-runner`: источник и содержимое PR comment меняются с markdown-секции открытых вопросов на последнее сообщение Codex после генерации proposal.

## Impact

- `internal/proposalrunner`: захват последнего содержательного ответа Codex, отказ от `CollectOpenQuestions` как основного источника PR comment, обновление ошибок и логов.
- `.codex/skills/openspec-propose/SKILL.md`: упрощение финального формата ответа без отдельного блока открытых вопросов.
- `openspec/specs/codex-proposal-pr-runner/spec.md`: изменение требований к PR comment.
- `docs/proposal-runner.md` и unit-тесты proposal runner: обновление описания и проверок под новый workflow.
