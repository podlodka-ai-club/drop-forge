## Why

Проект уже умеет выполнять один из сценариев работы агента: запускать proposal workflow по текстовому описанию задачи и публиковать результат в GitHub. Следующий шаг в целевой архитектуре - добавить `TaskManager` как отдельный слой интеграции с Linear, чтобы будущий `CoreOrch` мог получать задачи из конкретного проекта Linear, читать описание и историю human feedback, а затем возвращать результат работы обратно в Linear без ручного копирования данных между системами.

## What Changes

- Добавить `TaskManager`, который инкапсулирует взаимодействие с Linear и не берет на себя обязанности `CoreOrch`.
- Поддержать выбор задач только из одного настроенного проекта Linear и только из настраиваемых state'ов `ready to propose`, `ready to code` и `ready to archive`.
- Поддержать получение данных задачи в форме, пригодной для `CoreOrch`: идентификатор, ключ, описание и комментарии, включая human feedback после HITL reject, чтобы `CoreOrch` мог передать обновленный контекст в нужный executor.
- Поддержать операции обратной синхронизации в Linear: перевод задачи в другой state, добавление комментария и привязка PR к задаче.
- Вынести настройки Linear, project filter и state mapping в централизованный runtime-конфиг и `.env.example`.
- Зафиксировать наблюдаемость и тестируемость Linear-интеграции на том же уровне, что и текущий proposal workflow.
- Заложить unit-тесты на project-scoped выборку задач, возврат комментариев после HITL reject, write-операции `move/comment/add PR` и обработку ошибок/пустых данных от Linear.

## Capabilities

### New Capabilities
- `linear-task-manager`: Получение и обновление задач Linear в рамках одного проекта, включая описание, комментарии, state transitions и привязку PR.

### Modified Capabilities

## Impact

- Новый пакет `TaskManager` как Linear-facing слоя для будущего `CoreOrch`.
- Расширение `internal/config` и `.env.example` новыми runtime-параметрами Linear и state mapping.
- Будущий `CoreOrch` сможет использовать `TaskManager` для получения задачи вместе с human comments и для обратной записи результата.
- Новые спецификации и тесты для project-scoped доступа к задачам Linear, повторного чтения задачи после HITL reject и операций обновления задачи.
