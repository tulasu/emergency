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

## Интеграция аудио

TraineeBox отправляет сценарии в `config.audio.traineebox_audio_url` (`http://audio:8002` по умолчанию), а dialog читает манифесты и WAV через `config.audio.dialog_audio_url` (`http://127.0.0.1:8002`, доступный из host network). Audio-сервис вызывает callback по `config.audio.traineebox_url`; все три адреса и общий `INTERNAL_SERVICE_TOKEN` задаются в CUE. Пустой `traineebox_audio_url` пропускает ensure; пустой `dialog_audio_url` или отказ audio оставляет звонок на живом TTS. Лимит `AUDIO_CACHE_MB` (500 МБ по умолчанию) ограничивает как память, так и WAV-кеш dialog в `/tmp/dialog-audio`; старые файлы вытесняются при загрузке новых.

После обновления ID фрагментов существующие манифесты нужно перестроить: `just up` выполняет `import-scenarios` с предрендером всех билетов; при отдельном развёртывании запустите `just import-scenarios`. Повторный `ensure` обновляет манифест и при неизменном digest, если изменились ID или текст фрагментов. Пока билет не обработан, dialog использует живой TTS вместо несовпавших ID.

## Лицензия

Разработано в рамках хакатона © 2026 Команда «Токеноежки»
