# ELK demo — как поднять и что показать

## Требования

- Docker / Docker Desktop.
- ≥ 2 GB свободной RAM на docker-хосте (ES под JVM + Logstash + Kibana).
- Порты `5000`, `5601`, `9200` свободны на `127.0.0.1`.

## Запуск стека

```
docker compose -f deploy/docker-compose.yml up -d
```

Ждём, пока `kibana-setup` выйдет с кодом 0:

```
docker compose -f deploy/docker-compose.yml logs -f kibana-setup
```

После этого в Kibana (`http://localhost:5601`) создан data view `orchv3-*`.

## Проверка health

- Elasticsearch: `curl -sf http://127.0.0.1:9200/_cluster/health?pretty`
- Kibana: `curl -sf http://127.0.0.1:5601/api/status | jq '.status.overall'`

## Остановка

Мягкая остановка (данные ES сохраняются в volume):

```
docker compose -f deploy/docker-compose.yml down
```

## Полный сброс данных (чистый лист перед демо)

```
docker compose -f deploy/docker-compose.yml down -v
```

Флаг `-v` удаляет именованный volume `es-data`. Следующий `up -d` начнёт с пустого индекса.

## Smoke-test (без оркестратора)

```
echo '{"time":"2026-04-24T10:00:00Z","service":"orchv3","module":"smoke","type":"info","message":"hello"}' \
  | nc 127.0.0.1 5000
```

Через пару секунд событие видно в Kibana Discover, индекс `orchv3-*`.

## Запуск оркестратора с доставкой в ELK

В `.env`:

```
LOGSTASH_ADDR=127.0.0.1:5000
```

Запуск бинаря — как обычно. Поле `service` берётся из `APP_NAME`.

Если `LOGSTASH_ADDR` пусто — sink выключен, бинарь работает как раньше (только stderr).

## Экспорт дашборда после ручной настройки

1. В Kibana создать дашборд «orchv3 live» (timeline, modules × levels, recent errors).
2. Management → Stack Management → Saved Objects → Export (Data views + Dashboards).
3. Полученный `.ndjson` сохранить как `deploy/kibana/saved-objects.ndjson` (заменить).
4. `docker compose -f deploy/docker-compose.yml up -d --force-recreate kibana-setup` — идемпотентный reimport.

## Smoke-проверка чистоты stderr

На типичном прогоне каждая строка stderr должна быть валидным JSON:

```
orchv3 <args> 2> /tmp/run.log
jq -c . < /tmp/run.log
```

`jq` не должен выдавать ошибок.
