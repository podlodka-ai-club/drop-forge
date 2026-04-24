## Why

Для демо на хакатоне оркестратор `orchv3` должен показывать живой дашборд с ходом работы — события по модулям, уровни, последние ошибки. Текущее JSON-логирование в stderr даёт корректный формат, но не имеет системы визуализации и агрегации. Нужно доставить события в ELK-стек, сохранив при этом существующее поведение stderr-логирования (никаких регрессов при отсутствии ELK).

## What Changes

- Добавить опциональный TCP-sink в `internal/steplog`: при задании `LOGSTASH_ADDR` приложение шлёт каждое JSON-событие в Logstash параллельно записи в stderr. Пустой `LOGSTASH_ADDR` = фича выключена, поведение как сегодня.
- Добавить поле `service` в `Event` и конструктор `NewWithService(out, service)` в `steplog`. Старый `New(out)` остаётся и пишет событие без `service` (поле опускается при `omitempty`).
- Добавить в `internal/config` секцию `LogstashConfig` с ключами `LOGSTASH_ADDR`, `LOGSTASH_BUFFER_SIZE` (default 1024), `LOGSTASH_DIAL_TIMEOUT` (default 2s).
- Добавить `deploy/docker-compose.yml` с Elasticsearch + Logstash + Kibana в single-node dev-режиме без xpack-security. Порты Logstash `5000` и Kibana `5601` биндятся только на `127.0.0.1`. Tags зафиксированы конкретной версией (`8.13.4`), без `latest`.
- Добавить pipeline `deploy/logstash/pipeline/orchv3.conf` (input TCP `json_lines` → filter date+rename → output ES индекс `orchv3-YYYY.MM.dd`).
- Добавить автопровижининг Kibana через одноразовый setup-контейнер: data view `orchv3-*` и дашборд «orchv3 live» импортируются через `saved-objects.ndjson`.
- Обновить `.env.example` и добавить `docs/elk-demo.md` (запуск, health, остановка, сброс volume, smoke-test через `nc`).
- Покрыть TCP-sink unit-тестами: delivery, non-blocking Write, overflow drop-counter, reconnect, close-flush, periodic drop-summary warning.

## Capabilities

### New Capabilities

- `elk-log-shipping`: опциональная доставка JSON-логов приложения в локальный ELK-стек через TCP + провижининг самого стека через `docker-compose`.

### Modified Capabilities

- `structured-logging`: схема события дополняется опциональным полем `service`; добавляется конструктор `NewWithService`. Существующий `New` продолжает работать и писать события без `service`.

## Impact

- Код: `internal/steplog` (новый `tcp_sink.go`, расширение `Event`), `internal/config` (новая секция `LogstashConfig`), `cmd/orchv3` (новый `logger_setup.go`, правки `main.go`).
- Инфраструктура: новый каталог `deploy/` с `docker-compose.yml`, Logstash pipeline, Kibana setup-скрипт и saved-objects.
- Документация: `docs/elk-demo.md`, обновлённый `.env.example`.
- Тесты: новый `internal/steplog/tcp_sink_test.go`, расширение `internal/steplog/logger_test.go`, `internal/config/config_test.go`, `cmd/orchv3/logger_setup_test.go`. Существующие тесты не меняются.
- Зависимости: ничего в Go (только stdlib). На docker-хосте — Docker Compose и ≥ 2 GB свободной RAM.
- Отмена: фича аддитивная, rollback описан в design.md; без `LOGSTASH_ADDR` приложение работает как сегодня.
