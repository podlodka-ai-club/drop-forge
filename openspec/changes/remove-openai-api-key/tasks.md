## 1. Конфигурация

- [ ] 1.1 Удалить поле `OpenAIAPIKey` из `internal/config.Config` и убрать чтение `OPENAI_API_KEY` в `Load()`.
- [ ] 1.2 Обновить тесты `internal/config`, удалив очистку, установку и проверки `OPENAI_API_KEY`.
- [ ] 1.3 Проверить через `rg "OPENAI_API_KEY|OpenAIAPIKey"` отсутствие оставшихся ссылок в Go-коде.

## 2. Шаблон окружения и документация

- [ ] 2.1 Удалить `OPENAI_API_KEY` из `.env.example`.
- [ ] 2.2 Обновить README со списком поддерживаемых runtime-параметров приложения без `OPENAI_API_KEY`.
- [ ] 2.3 Убедиться, что документация не описывает `OPENAI_API_KEY` как настройку orchestrator-а.

## 3. Проверка

- [ ] 3.1 Запустить `go fmt ./...`.
- [ ] 3.2 Запустить `go test ./...`.
- [ ] 3.3 Запустить `openspec status --change remove-openai-api-key` и убедиться, что proposal готов к apply.
