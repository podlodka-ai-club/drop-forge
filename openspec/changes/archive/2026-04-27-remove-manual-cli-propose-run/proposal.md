## Why

Сейчас proposal workflow можно запустить вручную через CLI первой тестовой командой, что оставляет обходной путь мимо Linear-очереди и ручной режим управления задачами. Для DRO-32 нужно перевести запуск Propose в постоянный оркестратор, который сам мониторит столбец `Ready to propose` и запускает подготовку proposal без ручной передачи задачи в CLI.

## What Changes

- **BREAKING**: удалить ручной CLI-режим, который принимает описание задачи через аргументы или stdin и напрямую запускает proposal runner.
- Изменить основной CLI-запуск так, чтобы он стартовал бесконечный цикл мониторинга Linear-задач в состоянии `Ready to propose`.
- В каждом цикле запускать существующий proposal orchestration pass для найденных задач и затем ждать следующего интервала polling.
- Добавить runtime-настройку интервала polling через `.env` и синхронизировать `.env.example`.
- Сохранять текущий контракт proposal runner как внутренний executor, вызываемый только оркестратором.

## Capabilities

### New Capabilities

- Нет.

### Modified Capabilities

- `proposal-orchestration`: меняется режим запуска orchestration с одноразового CLI-pass на бесконечный мониторинг `Ready to propose`.
- `codex-proposal-pr-runner`: удаляется публичное CLI-поведение прямого запуска proposal runner по описанию задачи.

## Impact

- CLI entrypoint и wiring конфигурации запуска.
- Proposal orchestration loop, логирование polling-циклов и обработка ошибок между итерациями.
- Конфигурация `.env` / `.env.example` для polling interval.
- Тесты CLI/orchestration, которые сейчас ожидают прямой ручной запуск или одноразовый pass.
