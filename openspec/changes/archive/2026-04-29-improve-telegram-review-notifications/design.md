## Context

Сейчас `TaskManager.MoveTask` после успешного Linear transition публикует `task.status_changed` только с `TaskID` и `TargetStateID`. Telegram subscriber подписан на все такие события и отправляет сообщение для каждого перехода, включая in-progress состояния. Формат уже умеет использовать identifier/title/target state name, но publisher их фактически не передает.

В orchestration flow нужный бизнес-контекст уже есть в `CoreOrch`: исходная `taskmanager.Task` содержит identifier, title и attached PRs, а proposal-stage получает новый PR URL прямо перед переводом в `Need Proposal Review`. Поэтому изменение можно сделать без дополнительных Linear/GitHub запросов.

## Goals / Non-Goals

**Goals:**

- Уведомлять Telegram только когда задача перешла в состояние, где нужен человек: proposal/code/archive review.
- Включать в сообщение readable task reference: identifier и title, а не только UUID.
- Включать PR URL, когда он известен orchestration flow или есть в task payload.
- Сохранить best-effort уведомления: Telegram failure логируется и не ломает успешный Linear transition.
- Сохранить обратную совместимость базового `MoveTask(ctx, taskID, stateID)` для локальных вызовов и тестов.

**Non-Goals:**

- Не менять Telegram Bot API интеграцию, способ конфигурации или транспорт.
- Не добавлять новый event broker или асинхронную доставку.
- Не выполнять дополнительные запросы в Linear/GitHub только ради уведомления.
- Не менять lifecycle proposal/apply/archive runner-ов.

## Decisions

1. **Фильтрация будет в Telegram subscriber.**

   Subscriber получает список review state IDs из конфигурации и пропускает `task.status_changed`, если `TargetStateID` не совпадает с одним из `LINEAR_STATE_NEED_PROPOSAL_REVIEW_ID`, `LINEAR_STATE_NEED_CODE_REVIEW_ID`, `LINEAR_STATE_NEED_ARCHIVE_REVIEW_ID`.

   Альтернатива: публиковать события только для review-состояний. Она хуже для расширяемости: внутренний event stream перестанет отражать все смены статусов, а будущие подписчики не смогут использовать in-progress переходы.

2. **Расширить payload события optional-полями `PullRequestURL` и `PullRequestBranch`.**

   Это сохраняет стабильный event type и не ломает существующих подписчиков. Telegram формат использует `PullRequestURL` как основную ссылку; branch можно показывать только если URL отсутствует.

   Альтернатива: передавать весь `taskmanager.Task` в событие. Это сильнее связывает `events` с task manager model и раздувает payload данными, которые уведомлениям не нужны.

3. **Добавить расширенный путь публикации статуса в `TaskManager`.**

   Базовый `MoveTask` остается как есть. Для orchestration flow нужен вариант, который вместе с transition принимает snapshot metadata: task identifier, title, current/target state names и PR URL/branch. Это может быть новый метод на manager, например `MoveTaskWithContext`, или небольшой options struct вокруг существующего метода. Внешний интерфейс `coreorch.TaskManager` обновляется под потребности оркестратора.

   Альтернатива: после `MoveTask` отдельно публиковать событие из `CoreOrch`. Тогда появится дублирование ответственности: task manager уже является владельцем Linear transition и event publication.

4. **PR URL берется из ближайшего доступного источника.**

   Proposal-stage передает `prURL`, который вернул `ProposalRunner`, при финальном переходе в `Need Proposal Review`. Apply/Archive используют первый deterministic PR URL из `task.PullRequests`; если URL отсутствует, но есть branch, сообщение может показать branch как fallback, но не должно выдумывать ссылку.

## Risks / Trade-offs

- [Risk] Если caller использует старый `MoveTask`, Telegram может отфильтровать событие по state ID, но сообщение останется без title/PR. → Mitigation: orchestration flow должен использовать расширенный метод для review-переходов; formatter сохраняет fallback на ID.
- [Risk] Конфигурация review state IDs уже обязательна для Linear task manager, но Telegram subscriber находится в другом пакете. → Mitigation: wiring передает review state IDs из существующей Linear config без новых env keys.
- [Risk] Code/Archive задачи могут иметь только branch без PR URL. → Mitigation: сообщение показывает PR URL только при наличии; отсутствие URL не блокирует статусный переход и доставку уведомления.
- [Risk] Фильтрация в subscriber скрывает in-progress переходы из Telegram. → Это ожидаемая trade-off ради снижения шума; полный поток остается доступен через internal events/logs.
