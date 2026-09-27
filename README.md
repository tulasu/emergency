# Тренажёр вызовов службы **112**

![Nuxt](https://img.shields.io/badge/Nuxt-4-00DC82?style=flat-square&logo=nuxt&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat-square&logo=typescript&logoColor=white)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)
![Python](https://img.shields.io/badge/Python-3.11+-3776AB?style=flat-square&logo=python&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Asterisk](https://img.shields.io/badge/Asterisk-20-FBB040?style=flat-square&logoColor=black)
![Caddy](https://img.shields.io/badge/Caddy-2-22D3EE?style=flat-square&logo=caddy&logoColor=black)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white)
![CUE](https://img.shields.io/badge/CUE-config-E85D04?style=flat-square)
![llama.cpp](https://img.shields.io/badge/llama.cpp-Qwen-1B1B1B?style=flat-square)

## Где что лежит

- [`web/`](web/) — клиент на Nuxt: вход, кабинет, пользователи и группы
- [`traineebox/`](traineebox/) — backend на Go: API, сессии, билеты, попытки, старт звонка через ARI
- [`dialog/`](dialog/) — голосовой движок на Python: STT/TTS, NLU, сценарий заявителя по AudioSocket
- [`ticketgen/`](ticketgen/) — AI-пайплайн на Python: черновики учебных сценариев из каталога и LLM
- [`audio/`](audio/) — сервис синтеза и кэширования аудиофрагментов; [`audio-synth/`](audio-synth/) — его внутренний GPU-синтезатор.

В Go-модуле `audio/` точка сборки — `cmd/audio`; `internal/handlers/http` отвечает за HTTP, `internal/services` — за worker и синтез, `internal/repositories` — за PostgreSQL, `internal/domain` — за проверку сценария и фрагменты. Конфигурация остаётся в `internal/config`. Независимые утилиты и клиент RustFS вынесены в `pkg/hash`, `pkg/wav`, `pkg/s3`.

## Запуск и обновление

Для нового развёртывания используйте `just up`. Команда генерирует `docker-compose.yml` из `config/*.cue`, собирает образы и запускает аудиосервисы в порядке `audio-postgres` → `audio-migrate` → `audio`: одноразовый `audio-migrate` ждёт готовности БД и выполняет `audio migrate up`; `audio` не стартует, пока миграция не завершится успешно.

Для обновления уже развёрнутой системы (включая новую миграцию) также используйте `just up`, а не ручной запуск `audio`. Обычный `docker compose restart audio` допустим только для перезапуска уже совместимого сервиса: он не выполняет миграции.

HTTP API аудиосервиса доступен на порту `8002`. `audio-synth` намеренно не публикует порт на хост и доступен только другим сервисам Compose по `http://audio-synth:8003`.

При ошибке Silero `audio-synth` переключается на локальную русскую нейросетевую модель `ru_RU-denis-medium` из поддерживаемого [OHF-Voice/piper1-gpl](https://github.com/OHF-Voice/piper1-gpl). Модель загружается при сборке образа, а не во время запроса; её датасет указан как CC0 в [карточке модели](https://huggingface.co/rhasspy/piper-voices/blob/main/ru/ru_RU/denis/medium/MODEL_CARD). Движок Piper распространяется под GPLv3. Голос fallback — Denis независимо от выбранного голоса Silero: это аварийное воспроизведение речи, не совпадающее по тембру с основным голосом.

## Лицензия

Разработано в рамках хакатона © 2026 Команда «Токеноежки»
