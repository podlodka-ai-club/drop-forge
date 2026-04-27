## Why

Сейчас proposal-stage запускается либо вручную через CLI по тексту задачи, либо отдельной командой на один orchestration pass. Для целевого сценария оркестратора нужен долгоживущий процесс, который сам мониторит Linear-столбец `Ready to propose` и запускает `Propose`, а первая тестовая команда ручного запуска больше не должна оставаться публичным рабочим путем.

## What Changes

- **BREAKING** Убрать прямой ручной запуск proposal runner из CLI по аргументам или `stdin`.
- Заменить явный одноразовый CLI-режим proposal orchestration на основной режим запуска процесса, который в бесконечном цикле опрашивает `Ready to propose`.
- Добавить настраиваемый интервал polling через runtime-конфигурацию и `.env.example`.
- Сохранить существующий поток обработки найденной задачи: подготовка input из Linear, запуск `Propose`, прикрепление PR и перевод задачи в `Need Proposal Review`.
- Обеспечить корректное завершение по отмене контекста/сигналу процесса и структурные логи для итераций цикла.
- Обновить документацию и архитектурное описание под долгоживущий orchestration loop.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `proposal-orchestration`: Proposal orchestration должен работать как долгоживущий polling loop по `Ready to propose`, а CLI больше не должен поддерживать ручной запуск proposal по тексту задачи или тестовую one-shot команду.
- `project-readme`: README должен описывать основной запуск оркестратора как мониторинг Linear, а не ручную передачу описания задачи в CLI.

## Impact

- Код: `cmd/orchv3`, `internal/coreorch`, `internal/config`, wiring логгера и обработка завершения процесса.
- Конфигурация: новая переменная polling interval в `.env.example` и загрузчике конфигурации без секретов и без значения в шаблоне.
- Документация: `README.md`, возможно `docs/proposal-runner.md` или отдельный документ про proposal orchestration.
- Архитектура: `architecture.md`, так как меняется interaction flow между CLI entrypoint и `CoreOrch`.
- Тесты: unit-тесты `internal/coreorch`, CLI wiring tests, config tests, регрессия на отсутствие ручного запуска через args/stdin.
