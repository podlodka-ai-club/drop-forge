# ELK logging integration

- Дата: 2026-04-24
- Статус: утверждённый дизайн, готов к имплементации
- Автор: дизайн-сессия, бриф от пользователя

## Цель

Добавить в `orchv3` доставку структурных логов в ELK-стек для демо на хакатоне. ELK выступает частью продуктового value proposition — жюри показываем live-дашборд с ходом оркестрации (события по модулям, уровни, последние ошибки).

Решение должно:

- работать без изменений сценария запуска: оркестратор остаётся локальным бинарём, ELK поднимается отдельно через `docker-compose`;
- включаться одной переменной окружения и быть полностью опциональным (пустое `LOGSTASH_ADDR` = фича выключена, поведение как сегодня);
- выживать при падении ELK или его отсутствии без блокировок и паник в приложении;
- требовать минимальной ручной работы в Kibana — data view и дашборд провижинятся автоматически.

## Неявные решения, зафиксированные в брейнсторме

- Сценарий запуска: бинарь локально, ELK в `docker-compose` (вариант 2 из обсуждения).
- Транспорт: TCP `json_lines` на Logstash (вариант A).
- Отказоустойчивость: ring-buffer с drop-oldest при overflow, фоновая горутина с reconnect (режим ii).
- Kibana: автопровижининг data view + одного дашборда через одноразовый setup-контейнер (вариант b).
- Scope: без xpack-security, без ILM, без дисковой персистенции буфера, без метрик Prometheus.

## Топология

```
┌─────────────────────────┐          ┌──────────────────────────────────────┐
│  orchv3 (локальный бин) │  TCP     │  docker-compose (deploy/)            │
│                         │ :5000    │                                      │
│  steplog.Logger         ├─────────▶│  logstash ─▶ elasticsearch ─▶ kibana │
│    ├─ stderr (как сейчас)          │                           ▲          │
│    └─ tcpSink (новый)   │          │              kibana-setup┘          │
└─────────────────────────┘          │              (one-shot provision)    │
                                     └──────────────────────────────────────┘
```

- Приложение продолжает писать JSON-события в `stderr`; параллельно тот же байт-поток уходит в TCP-sink на `127.0.0.1:5000`.
- Logstash слушает порт `5000` с кодеком `json_lines`, парсит `time → @timestamp`, пишет в Elasticsearch в индекс `orchv3-YYYY.MM.dd`.
- Kibana на `5601` с заранее импортированным data view `orchv3-*` и дашбордом «orchv3 live».
- Порт `5000` биндится на `127.0.0.1` — только локальный бинарь пишет, снаружи недоступно.

## Компоненты

### 1. `internal/steplog` — новый `TCPSink`

Новый тип в существующем пакете. Реализует `io.Writer` + `io.Closer`. Существующий `Logger` и `LineWriter` не меняются; `TCPSink` подключается через `io.MultiWriter` в `main.go`.

```go
type TCPSink struct {
    addr        string
    bufferSize  int
    dialTimeout time.Duration
    queue       chan []byte      // заранее сериализованные JSON-строки (с \n)
    dropped     atomic.Uint64    // счётчик потерянных из-за overflow
    done        chan struct{}
    wg          sync.WaitGroup
}

func NewTCPSink(addr string, bufferSize int, dialTimeout time.Duration) *TCPSink
func (s *TCPSink) Write(p []byte) (int, error) // неблокирующий
func (s *TCPSink) Close() error                // best-effort flush с таймаутом 2s
```

Внутренняя семантика:

- `Write` копирует `p`, пытается положить в `queue` через `select` с `default` — если буфер полон, инкрементирует `dropped` и возвращает `len(p), nil` (основной поток приложения не блокируется и не видит ошибку).
- Фоновая горутина:
  1. `net.DialTimeout("tcp", addr, dialTimeout)`, при ошибке — экспоненциальный бэкоф (старт 1s, максимум 30s, сброс при успехе).
  2. На живом соединении читает из `queue`, пишет пачкой (`bufio.Writer`, периодический flush). Ошибка записи → `conn.Close()` и возврат к шагу 1.
  3. Раз в 30s, если `dropped` вырос, пишет summary в stderr (через отдельный `io.Writer`, переданный в конструктор, чтобы не зациклить sink на себя).
- `Close`: закрывает `done`, горутина дренит канал в текущее соединение с общим таймаутом 2s, затем закрывает conn.

### 2. `internal/steplog.Event` — добавить поле `service`

```go
type Event struct {
    Time    string `json:"time"`
    Service string `json:"service,omitempty"` // новое
    Module  string `json:"module"`
    Type    string `json:"type"`
    Message string `json:"message"`
}
```

`Logger` получает опциональное значение `service`:

```go
func New(out io.Writer) Logger                      // по-прежнему работает
func NewWithService(out io.Writer, service string) Logger
```

Обратная совместимость сохраняется для существующих тестов; `main.go` использует `NewWithService(..., cfg.AppName)`.

### 3. `internal/config` — новая секция

```go
type Config struct {
    // ...существующие поля
    Logstash LogstashConfig
}

type LogstashConfig struct {
    Addr        string        // LOGSTASH_ADDR, пусто = sink выключен
    BufferSize  int           // LOGSTASH_BUFFER_SIZE, default 1024
    DialTimeout time.Duration // LOGSTASH_DIAL_TIMEOUT, default "2s"
}
```

Правила:

- Пустой `LOGSTASH_ADDR` = фича выключена, никаких warning, приложение как раньше.
- `LOGSTASH_BUFFER_SIZE` < 1 или не число → ошибка `Load()` (fail fast на misconfiguration).
- `LOGSTASH_DIAL_TIMEOUT` парсится через `time.ParseDuration`.
- `LogstashConfig` в `Validate()` не трогаем — она опциональна.

### 4. `cmd/orchv3/main.go` — один edit

**Wire TCP sink**. После загрузки конфига, если `cfg.Logstash.Addr != ""`, собираем sink, заворачиваем в `io.MultiWriter(stderr, sink)`, `defer sink.Close()`. Логгер создаётся уже поверх этого writer-а. Stderr для internal warning-ов sink-а — прямой `os.Stderr` (не MultiWriter, чтобы не попасть в рекурсию).

Subprocess stdout/stderr уже JSON-валидны на всех путях:

- `proposalrunner.runLoggedCommand` (`internal/proposalrunner/runner.go:341-362`) оборачивает subprocess stdout/stderr через `steplog.New(writer).LineWriter(module)`.
- `commandrunner.ExecRunner` (`internal/commandrunner/runner.go:30-50`) использует `LogWriter` только для одной `steplog.Infof("command", ...)`-записи.

Дополнительных правок для чистоты потока не требуется. В критериях приёмки фиксируем smoke-проверку, что каждая строка на stderr парсится как JSON.

### 5. `deploy/` — новый каталог с инфраструктурой

Структура:

```
deploy/
├── docker-compose.yml
├── logstash/
│   └── pipeline/orchv3.conf
└── kibana/
    ├── saved-objects.ndjson   # data view + 1 дашборд
    └── setup.sh               # ждёт Kibana, импортирует saved-objects.ndjson
```

#### `docker-compose.yml`

Четыре сервиса:

- `elasticsearch`: `docker.elastic.co/elasticsearch/elasticsearch:8.x`, env `discovery.type=single-node`, `xpack.security.enabled=false`, `ES_JAVA_OPTS=-Xms512m -Xmx512m`. Healthcheck на `:9200`. Volume `es-data:/usr/share/elasticsearch/data`.
- `logstash`: `docker.elastic.co/logstash/logstash:8.x`, bind-mount `./logstash/pipeline` → `/usr/share/logstash/pipeline`, `depends_on: elasticsearch (service_healthy)`, порт `127.0.0.1:5000:5000`.
- `kibana`: `docker.elastic.co/kibana/kibana:8.x`, `ELASTICSEARCH_HOSTS=http://elasticsearch:9200`, `depends_on: elasticsearch (service_healthy)`, порт `127.0.0.1:5601:5601`, healthcheck на `/api/status`.
- `kibana-setup`: `curlimages/curl:latest`, `restart: "no"`, `depends_on: kibana (service_healthy)`, entrypoint `sh /setup.sh`, bind-mount `./kibana/setup.sh` и `./kibana/saved-objects.ndjson`.

#### `logstash/pipeline/orchv3.conf`

```
input { tcp { port => 5000 codec => json_lines } }

filter {
  date   { match => ["time", "ISO8601"]  target => "@timestamp"  remove_field => ["time"] }
  mutate { rename => { "type" => "level" } }
}

output { elasticsearch {
  hosts => ["http://elasticsearch:9200"]
  index => "orchv3-%{+YYYY.MM.dd}"
} }
```

`type → level` делается, чтобы в Kibana поле называлось привычно и не конфликтовало с мета-полями ES.

#### `kibana/setup.sh`

Простой идемпотентный скрипт:

```
until curl -sf http://kibana:5601/api/status > /dev/null; do sleep 2; done
curl -sf -X POST "http://kibana:5601/api/saved_objects/_import?overwrite=true" \
  -H "kbn-xsrf: true" \
  --form file=@/saved-objects.ndjson
```

#### `kibana/saved-objects.ndjson`

Экспортируется из Kibana один раз вручную и коммитится. Содержит:

1. **Data view** `orchv3-*`, time field `@timestamp`.
2. **Dashboard «orchv3 live»** с тремя визуализациями:
   - *Events timeline* — line chart, bucket по 1-минуте, split series по `module.keyword`.
   - *Modules × levels* — table, rows `module.keyword`, columns `level.keyword`, metric `count`.
   - *Recent errors* — saved search, filter `level: "error"`, sort `@timestamp desc`, limit 50.

### 6. `.env.example` и документация

Дописываем в `.env.example` новую секцию (пустые значения — шаблон):

```
# ELK integration
LOGSTASH_ADDR=
LOGSTASH_BUFFER_SIZE=
LOGSTASH_DIAL_TIMEOUT=
```

В `docs/` добавляем короткую `elk-demo.md` с чеклистом: как поднять стек, как проверить приём события через `nc`, URL дашборда.

## Поток данных

1. Приложение вызывает `logger.Infof(module, format, args...)`.
2. `Logger.write` сериализует `Event` через `json.Encoder.Encode` в `MultiWriter`.
3. `MultiWriter` фанаутит в `stderr` (синхронно, как сейчас) и в `TCPSink.Write` (неблокирующий put в канал).
4. Фоновая горутина `TCPSink` достаёт из канала, отправляет по TCP.
5. Logstash получает JSON-строку, парсит `time` в `@timestamp`, ренеймит `type → level`, пишет в ES в индекс `orchv3-YYYY.MM.dd`.
6. Kibana читает из `orchv3-*`, на дашборде отрисовываются timeline/table/errors.

Основной поток оркестратора никогда не блокируется на сетевом I/O. Stderr-поведение идентично сегодняшнему (локально всё видно как сейчас).

## Обработка ошибок

| Ситуация | Поведение |
|---|---|
| На старте Logstash недоступен | Одна warning-строка в stderr (`steplog: logstash sink unavailable, will retry`). Фоновая горутина стучится бэкофом. Основной поток не замечает. |
| Logstash упал посреди работы | Write на conn возвращает ошибку → закрываем conn, возвращаемся к dial. События продолжают складываться в канал. |
| Буфер переполнился | `Write` не блокирует: событие дропается, `dropped++`. Раз в 30s в stderr summary `steplog: dropped N events due to sink overflow`. |
| Graceful shutdown (`Close`) | Закрывается `done`, горутина дренит канал в текущее соединение с общим таймаутом 2s, затем закрывает conn. |
| Misconfiguration (`LOGSTASH_BUFFER_SIZE=abc`) | `config.Load()` возвращает ошибку, бинарь падает с понятным сообщением до создания sink. |

## Тестирование

### Новые тесты

`internal/steplog/tcp_sink_test.go`:

1. `TestTCPSink_DeliversEvents` — стартуем `net.Listen("tcp", "127.0.0.1:0")`, создаём sink, пишем 10 событий, со стороны сервера читаем `bufio.Scanner` по строкам и сверяем.
2. `TestTCPSink_ReconnectsAfterServerRestart` — закрываем listener, пишем событие (уходит в канал), поднимаем новый listener на том же порту, пишем ещё одно — оба доходят.
3. `TestTCPSink_DropsOnOverflow` — без сервера заливаем `bufferSize + N` событий, проверяем `sink.Dropped() == N`.
4. `TestTCPSink_NonBlockingWrite` — без сервера делаем 1000 `Write`, замер времени < 50ms (не блокируется на dial).
5. `TestTCPSink_CloseFlushes` — с сервером пишем пачку, сразу `Close()`, на сервере должны прийти все события.

### Существующие тесты

`internal/steplog/logger_test.go` и `internal/config/config_test.go` расширяются минимально: добавить case на новое поле `service` в Event и случаи `LogstashConfig` (default, custom, invalid).

`internal/commandrunner/runner_test.go` и `internal/proposalrunner/runner_test.go` не меняем — их поведение из-за новой фичи не затрагивается. Прогон `go test ./...` подтверждает.

### Интеграционный прогон ELK

Не автоматизируем. В `docs/elk-demo.md` фиксируем ручной smoke-test:

```
docker compose -f deploy/docker-compose.yml up -d
# ждём kibana-setup exit 0
echo '{"time":"2026-04-24T10:00:00Z","module":"smoke","type":"info","message":"hello"}' | nc 127.0.0.1 5000
# открыть http://localhost:5601/app/dashboards → orchv3 live
```

## Переменные окружения (новые)

| Переменная | Default | Обязательна | Описание |
|---|---|---|---|
| `LOGSTASH_ADDR` | *(пусто)* | нет | `host:port` Logstash TCP-входа. Пусто = sink выключен. |
| `LOGSTASH_BUFFER_SIZE` | `1024` | нет | Ёмкость канала между Write и фоновой горутиной. |
| `LOGSTASH_DIAL_TIMEOUT` | `2s` | нет | Таймаут одной попытки `net.Dial`. Парсится `time.ParseDuration`. |

## Что НЕ делаем (scope boundary)

- Нет xpack-security (ни паролей ES, ни TLS к Logstash). Всё живёт на `127.0.0.1` и не переносится в production.
- Нет кластера из нескольких ES-нод.
- Нет ILM и index templates — стандартные маппинги ES.
- Нет метрик/эндпоинтов оркестратора (health, Prometheus).
- Нет дисковой персистенции буфера sink (режим iii из обсуждения).
- Нет рефакторинга `commandrunner`/`proposalrunner` — их I/O уже JSON-валидное.
- Нет multi-sink API (Kafka, UDP, syslog) — только TCP к Logstash.
- Нет прямого подключения приложения к Elasticsearch (`ELASTICSEARCH_URL` и т.п.). Приложение знает только про Logstash.

## Риски и компромиссы

- **Security выключена в dev.** Отключены xpack-security и TLS, чтобы упростить bootstrap (соответствует цели демо). Компромисс: не переносим такие настройки в production, отдельный ELK для прода потребует отдельного compose/манифеста.
- **Memory footprint Elasticsearch.** ES на JVM требует заметно больше памяти, чем оркестратор. Фиксируем умеренные лимиты `ES_JAVA_OPTS=-Xms512m -Xmx512m` и `LS_JAVA_OPTS=-Xms256m -Xmx256m` для Logstash. В документации указываем минимальные требования (≥2 GB свободной RAM на docker-хосте).
- **Зафиксированный tag image.** Используем явный tag (`docker.elastic.co/elasticsearch/elasticsearch:8.13.4` и аналоги), а не `latest` — чтобы демо не ломалось при неожиданном обновлении Elastic и чтобы версии ES/Logstash/Kibana между собой были совместимы.
- **Грязный volume между запусками.** Данные ES живут в именованном volume `es-data` и сохраняются между перезапусками. Иногда на демо нужен «чистый лист» — процедура полного сброса описана в `docs/elk-demo.md`.
- **Потеря событий при долгом падении ELK.** Буфер in-memory, fixed size; при длительной недоступности Logstash самые старые события дропаются. Для хакатона приемлемо; счётчик потерь публикуется в stderr раз в 30s.

## Миграция и rollback

Фича аддитивная: без `LOGSTASH_ADDR` приложение ведёт себя как сегодня. Rollback сводится к трём шагам:

1. Удалить `deploy/` (compose + pipelines + kibana provisioning).
2. В `.env.example` убрать секцию `# ELK integration` с тремя ключами.
3. В `internal/config` убрать `LogstashConfig` и секцию `loadLogstashConfig`; в `cmd/orchv3/main.go` убрать вызов `buildLogger`, вернуть прямой `steplog.New(stderr)`.

Поле `Event.Service` и конструктор `NewWithService` можно оставить — они полезны сами по себе и не требуют ELK.

## Критерии приёмки

1. `go fmt ./...` и `go test ./...` проходят зелёными.
2. Новые тесты `TCPSink` покрывают delivery, reconnect, overflow, non-blocking, close-flush.
3. `docker compose -f deploy/docker-compose.yml up -d` поднимает ES/Logstash/Kibana и `kibana-setup` выходит с кодом 0.
4. После импорта saved-objects на `http://localhost:5601/app/dashboards` виден дашборд «orchv3 live» и data view `orchv3-*`.
5. Запуск бинаря с `LOGSTASH_ADDR=127.0.0.1:5000` приводит к появлению событий в Kibana в течение <5s.
6. Запуск бинаря с `LOGSTASH_ADDR=` (пусто) работает как сегодня: stderr-JSON, ни одной warning-строки от sink.
7. Если поднять бинарь до ELK — бинарь не падает, при последующем старте ELK события начинают появляться в Kibana.
8. Если остановить Logstash во время работы — бинарь не падает и не блокируется, при рестарте Logstash доставка возобновляется.
9. `.env.example` содержит три новых ключа с пустыми значениями.
10. В `docs/elk-demo.md` описаны: старт/остановка стека, nc-smoke, URL дашборда, **процедура полного сброса (`docker compose down -v`)** и минимальные требования к памяти.
11. На типичном прогоне оркестратора каждая строка stderr парсится как валидный JSON (smoke-проверка: `jq -c . < run.log` не выдаёт ошибок).
12. Все image tags в `deploy/docker-compose.yml` зафиксированы конкретной версией (`8.13.4` или новее), ни одного `latest`.
