## Why

`OPENAI_API_KEY` сейчас объявлен как runtime-параметр приложения, но проект запускает Codex CLI как внешний агент и не использует этот ключ напрямую. Наличие неиспользуемого секрета в конфигурации вводит в заблуждение, создает лишнее требование к окружению и увеличивает риск случайного хранения секретов.

## What Changes

- Удалить поддержку `OPENAI_API_KEY` из централизованной конфигурации приложения.
- Убрать `OPENAI_API_KEY` из `.env.example` и документации поддерживаемых runtime-параметров.
- Обновить тесты конфигурации так, чтобы они больше не ожидали чтения или очистки `OPENAI_API_KEY`.
- Сохранить существующий способ запуска Codex CLI без передачи ключа из приложения.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `codex-proposal-pr-runner`: runtime-конфигурация и `.env.example` больше не должны включать неиспользуемый `OPENAI_API_KEY`; агентский runtime полагается на собственное окружение Codex CLI.

## Impact

- `internal/config`: структура `Config`, загрузка окружения и тесты конфигурации.
- `.env.example`: удаление ключа `OPENAI_API_KEY`.
- `README.md`: актуализация списка runtime-параметров.
- Внешнее поведение orchestration flow, Linear-интеграции, git/gh workflow и OpenSpec artifact generation не меняется.
