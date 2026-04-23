## 1. Docker Infrastructure

- [ ] 1.1 Добавить в корень репозитория compose-манифест с одним сервисом Elasticsearch для локальной разработки.
- [ ] 1.2 Настроить контейнер Elasticsearch в single-node режиме с опубликованным HTTP-портом, именованным volume и healthcheck.
- [ ] 1.3 Зафиксировать dev-параметры контейнера, включая упрощенную локальную security-конфигурацию и умеренные JVM-лимиты.

## 2. Application Configuration

- [ ] 2.1 Добавить в `.env.example` ключи подключения к Elasticsearch без значений по умолчанию.
- [ ] 2.2 Расширить `internal/config` чтением и валидацией Elasticsearch-настроек из `.env` и process environment.
- [ ] 2.3 Обновить тесты конфигурации для новых Elasticsearch-переменных и поведения загрузки env.

## 3. Documentation

- [ ] 3.1 Добавить документацию по запуску и остановке локального Elasticsearch через Docker Compose.
- [ ] 3.2 Описать проверку health/status Elasticsearch по HTTP и ожидаемую точку подключения приложения.
- [ ] 3.3 Описать сценарий полного сброса локальных данных через удаление Docker volume.

## 4. Verification

- [ ] 4.1 Проверить локальный сценарий `docker compose up` до состояния healthy для Elasticsearch.
- [ ] 4.2 Проверить, что приложение читает Elasticsearch endpoint из централизованной конфигурации без hardcode.
- [ ] 4.3 Запустить `go fmt ./...` и `go test ./...`.
