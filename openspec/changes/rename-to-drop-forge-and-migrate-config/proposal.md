## Why

Проект начинался как автоматизация стадии proposal, но теперь управляет несколькими стадиями Linear workflow и будет расширяться дальше. Старые имена, ENV-ключи и stage-specific формулировки создают путаницу: конфигурация выглядит привязанной к proposal-only сценарию, а код и пользовательские сообщения не отражают роль приложения как общего оркестратора.

## What Changes

- Переименовать пользовательское и runtime-название приложения в `Drop Forge`.
- Ввести единый нейтральный слой конфигурации для оркестратора и стадий вместо proposal-only названий там, где параметр больше не относится только к proposal.
- Сохранить понятные stage-specific ключи только для действительно stage-specific настроек Linear и runner-ов.
- Обновить `.env.example`, загрузку конфигурации, валидацию и тесты так, чтобы поддерживаемые runtime-параметры были явно описаны и не содержали значений по умолчанию в шаблоне.
- Обновить CLI/startup/logging/PR metadata/docs так, чтобы внешнее имя `Drop Forge` не конфликтовало с внутренними пакетами и существующими stage names.
- **BREAKING**: устаревшие ENV-ключи с proposal-only смыслом, которые теперь описывают общий runtime оркестратора, будут заменены на новые нейтральные ключи. Обратная совместимость не требуется, если она усложняет конфигурацию.

## Capabilities

### New Capabilities

- `drop-forge-configuration`: единая runtime-конфигурация и внешняя идентичность приложения Drop Forge.

### Modified Capabilities

- `proposal-orchestration`: polling/runtime-настройки и CLI-поведение больше не должны быть описаны как proposal-only runtime.
- `codex-proposal-pr-runner`: runner сохраняет proposal-specific контракт, но использует общую конфигурацию приложения и имя Drop Forge во внешних сообщениях без переименования stage-specific сущностей.
- `linear-task-manager`: Linear-конфигурация должна быть разделена на общие параметры Drop Forge и stage-specific state IDs.

## Impact

- Код загрузки конфигурации, `.env.example`, тесты конфигурации и места wiring в CLI.
- Пользовательские сообщения CLI, startup/fatal logs, PR/comment metadata и документация, где фигурирует прежнее proposal-only название.
- Спеки OpenSpec для конфигурации, orchestration runtime, proposal runner и Linear task manager.
- Локальная конфигурация разработчиков и окружения запуска: потребуется обновить имена переменных окружения согласно новой схеме.
