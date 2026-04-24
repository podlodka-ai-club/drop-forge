## Context

Приложение `orchv3` уже пишет валидные JSON-события в stderr (capability `structured-logging`, archive `2026-04-22-standardize-json-logging`). Для демо-сценария этого недостаточно: жюри и зрители смотрят на дашборд с графиками по модулям, уровням и списком ошибок. Нужна доставка тех же событий в систему визуализации и агрегации без изменения сценария запуска оркестратора (он остаётся локальным бинарём).

Параллельно в ветке `origin/dropforge/propsal/20260423110714-docker-elastic` существовал proposal `add-elasticsearch-docker-container` (concept-level, кода нет). Его scope — только одинокий Elasticsearch в docker-compose и `ELASTICSEARCH_URL`. Он поглощается этим change: мы расширяем scope до полного ELK-стека (ES + Logstash + Kibana) и не подключаем приложение к ES напрямую — события идут через Logstash.

Технически JSON-логгер уже готов к forwarding-у: `steplog.Logger` пишет в `io.Writer`, который можно завернуть в `io.MultiWriter(stderr, sink)`. Остаётся добавить sink, сделать его опциональным, отказоустойчивым и поднять ELK-стек с автопровижинингом Kibana.

## Goals / Non-Goals

**Goals:**

- Опциональный TCP-sink в `steplog`: включается одной переменной `LOGSTASH_ADDR`, пусто = фича выключена.
- Неблокирующий `Write` — основной поток оркестратора никогда не ждёт сетевого I/O.
- Устойчивость к недоступности Logstash: бинарь не падает, если ELK ещё не поднят или упал; при восстановлении доставка возобновляется.
- Drop-oldest семантика переполнения буфера + счётчик потерь, периодически публикуемый в stderr.
- `docker-compose` с ES + Logstash + Kibana в dev-режиме, без xpack-security, с зафиксированными tag image.
- Автопровижининг Kibana: data view `orchv3-*` и дашборд «orchv3 live» через saved-objects import.
- Сохранить текущее поведение stderr-логирования без регрессов.

**Non-Goals:**

- Не включаем xpack-security, TLS, аутентификацию ES/Kibana — эти настройки не переносятся в production.
- Не строим кластер ES — только single-node dev.
- Не добавляем ILM, index templates, custom mappings.
- Не подключаем приложение к Elasticsearch напрямую — только через Logstash.
- Не добавляем health/metrics endpoints в оркестратор.
- Не добавляем дисковую персистенцию буфера sink — in-memory drop-oldest достаточен для демо.
- Не делаем multi-sink (Kafka, UDP, syslog) — только TCP к Logstash.
- Не рефакторим `commandrunner`/`proposalrunner` — их I/O уже JSON-валидное.

## Decisions

### 1. Транспорт — TCP `json_lines` к Logstash

Альтернативы: HTTP bulk прямо в Elasticsearch (без Logstash) или файл + Filebeat. TCP к Logstash выигрывает по трём осям: (1) классическая «ELK» история, понятная жюри; (2) Logstash берёт на себя парсинг даты и retry на своей стороне; (3) в Go-клиенте — простой `net.Conn`, bufio, `json.Encoder`. HTTP bulk требует больше клиентского кода (буфер, ticker, HTTP-клиент, retry). Файл + Filebeat даёт лишний контейнер и проблемы с bind-mount на Windows-хостах.

### 2. Отказоустойчивость — drop-oldest + reconnect, без дисковой персистенции

Ring-buffer фиксированного размера (default 1024). `Write` неблокирующий: `select`+`default`, при полном канале инкрементируется `dropped atomic.Uint64`. Фоновая горутина делает `net.DialTimeout` с экспоненциальным бэкофом (1s → 30s cap), на живом соединении пишет через `bufio.Writer`. Ошибка записи → `conn.Close()` → back to dial. Для демо этого достаточно; дисковый буфер (режим «ничего не теряем») выходит за scope.

Альтернатива: блокирующий `Write` с ретраями в основном потоке. Отвергнута: оркестратор не должен зависать, если Logstash лёг во время демо.

### 3. `Close` — best-effort flush с таймаутом 2s

При shutdown горутина дренит оставшиеся payload-ы в текущее соединение с общим таймаутом `closeFlushTimeout = 2s`. Достаточно, чтобы не терять последние события штатной остановки; и достаточно мало, чтобы бинарь не подвисал на мёртвой сети.

### 4. `service`-поле через `NewWithService`, а не отдельный логгер

Event получает `Service string `json:"service,omitempty"``. В `steplog.Logger` добавляется приватное поле `service`, конструктор `NewWithService(out, service)`. Старый `New(out)` делегирует в `NewWithService(out, "")` — обратная совместимость сохраняется, все существующие тесты проходят без изменений. На выходе пустой `service` опускается из JSON благодаря `omitempty`.

### 5. `docker-compose` в `deploy/`, а не в корне репо

Их proposal `add-elasticsearch-docker-container` предлагал корень. Мы выбираем `deploy/` — в корне появятся pipeline Logstash и Kibana-артефакты, их группировка под одним каталогом яснее. В `docs/elk-demo.md` фиксируется точный путь к compose-файлу.

### 6. Kibana автопровижининг через saved-objects import

Одноразовый `kibana-setup` контейнер на `curlimages/curl`, `restart: "no"`, ждёт `kibana:5601/api/status` и делает `POST /api/saved_objects/_import?overwrite=true`. Идемпотентно при пересоздании контейнера. На первом шаге коммитится `saved-objects.ndjson` только с data view `orchv3-*`; дашборд создаётся в Kibana UI вручную один раз и экспортируется в тот же файл (процедура описана в `docs/elk-demo.md`).

### 7. Pinned image tags, без `latest`

Все image версии зафиксированы (`docker.elastic.co/.../...:8.13.4`, `curlimages/curl:8.7.1`) — чтобы демо не ломалось при неожиданном обновлении Elastic и чтобы версии ES/Logstash/Kibana были совместимы между собой. Это явное acceptance-требование.

## Risks / Trade-offs

- **Security выключена в dev** -> митигация: всё биндится на `127.0.0.1`, компромисс не переносится в production, production-ELK потребует отдельного compose-манифеста.
- **Memory footprint** (ES на JVM ~512 MB, Logstash ~256 MB, Kibana) -> митигация: `ES_JAVA_OPTS=-Xms512m -Xmx512m`, `LS_JAVA_OPTS=-Xms256m -Xmx256m`, требование ≥ 2 GB свободной RAM задокументировано.
- **Потеря событий при длительной недоступности ELK** -> митигация: `dropped` счётчик, периодический warning в stderr; для демо ELK поднимается до бинаря, сценарий редкий.
- **Грязный volume между запусками** -> митигация: `docker compose down -v` описан в `docs/elk-demo.md` как процедура полного сброса.
- **Драйф версий Elastic** -> митигация: pinned tags, без `latest`.
- **Handshake между `kibana-setup` и `kibana`** (Kibana готова по healthcheck, но saved-objects API ещё возвращает 503) -> митигация: setup.sh циклит `curl -sf /api/status` до зелёного ответа перед импортом.

## Migration Plan

1. Добавить `Event.Service`, `NewWithService` в `steplog` (capability `structured-logging` MODIFIED).
2. Добавить `LogstashConfig` в `internal/config` с дефолтами и парсингом новых env-ключей.
3. Реализовать `steplog.TCPSink` (delivery → non-blocking → reconnect → close-flush → drop-warnings).
4. Вынести сборку логгера в `cmd/orchv3/logger_setup.go` с тестами, подключить в `main.go` через `io.MultiWriter`.
5. Добавить `deploy/docker-compose.yml`, Logstash pipeline, Kibana setup + saved-objects.
6. Обновить `.env.example`, написать `docs/elk-demo.md`.
7. `go fmt ./...`, `go test ./...`, manual smoke: `docker compose up -d` → `nc 127.0.0.1 5000` → событие в Kibana.

**Rollback** (если фичу нужно убрать):

1. Удалить `deploy/` целиком.
2. Убрать из `.env.example` секцию `# ELK integration`.
3. В `internal/config` удалить `LogstashConfig` и `loadLogstashConfig`; в `cmd/orchv3/main.go` вернуть прямой `steplog.New(stderr)`, удалить `logger_setup.go`.
4. `Event.Service` и `NewWithService` можно оставить — они полезны сами по себе и не требуют ELK.

## Детальная спека и план

Полный narrative-дизайн и пошаговый план имплементации (с кодом и TDD-последовательностью) дублируются в:

- `docs/specs/2026-04-24-elk-logging-design.md` — расширенный design для агентов и ревью.
- `docs/plans/2026-04-24-elk-logging.md` — пошаговый план на 14 задач.

Формальные требования capability-ов — в `specs/elk-log-shipping/spec.md` и `specs/structured-logging/spec.md` текущего change.
