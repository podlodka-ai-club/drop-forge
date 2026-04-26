## Why

Linear-задача `DRO-28` создана как контрольный пример для проверки, что оркестратор действительно подтягивает описание задачи и комментарии, а не только заголовок и идентификатор. Сейчас это критично зафиксировать в contract-level поведении, потому что proposal agent должен получать весь человеческий контекст из Linear для корректной подготовки OpenSpec proposal.

## What Changes

- Уточнить поведение proposal orchestration для формирования входа proposal runner из Linear-задачи: `ID`, `Identifier`, `Title`, `Description` и `Comments` должны попадать в task description в явном виде.
- Добавить тестовое покрытие для задачи с описанием и комментариями на примере структуры данных `DRO-28`.
- Сохранить текущую устойчивость к пустому описанию и отсутствующим комментариям.
- Не менять Linear API contract, статусы workflow, PR-flow и внутренний git/Codex workflow proposal runner.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: уточняется требование к содержимому входа proposal runner: описание и комментарии Linear-задачи должны быть явно представлены и проверяться тестами на примере задачи с человеческим feedback.

## Impact

- Код формирования input для proposal runner в proposal orchestration.
- Unit-тесты proposal orchestration с fake task manager / fake proposal runner.
- Возможное точечное обновление документации, если текущий формат task input в docs не отражает description/comments.
- Внешние зависимости и runtime-конфигурация не меняются.
