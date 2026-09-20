# Запуск тренажёра в SIP: VAD → STT → dispatcher → TTS

Архитектура: `Asterisk (SIP/RTP) → Go-оркестратор (:8080) → Python-сервер (:8000)`.
Один звонок = один `Session`, реестр — `dispatcher/service.py`.

## 0. Требования

- Python 3.12 + `.venv` (ядро: `pymorphy3 razdel numpy PyYAML pytest`)
- Для шага 2: `espeak-ng`, `ffmpeg`
- Для шага 4: `go ≥1.22`
- Для прод-развёртки: Docker + NVIDIA driver, Asterisk 20, модели в `models/`

## 1. Проверка ядра без SIP

```bash
.venv/bin/python -m dispatcher.cli chat bilet04_call01
.venv/bin/python -m pytest tests -q   # 146 тестов
```

## 2. Предрендер фраз заявителя (TTS заранее)

```bash
python3 tools/synth_audio.py --limit 20  # проба пайплайна
python3 tools/synth_audio.py              # полный прогон: 3598 wav + data/audio/index.json (~150-200 МБ)
python3 tools/synth_audio.py --check      # все audio_id резолвятся в файлы
```

Без `data/audio/index.json` ответы идут через живой TTS по `Reply.text`
(хук `Call.on_tts_text`); с индексом `Reply.audio_id` указывает на готовый wav.

## 3. Python-сервер (понимание + речь)

```bash
# разработка, CPU, лексический каскад:
.venv/bin/python -m dispatcher.serve --port 8000

# прод, GPU (обязательно именно так):
.venv/bin/python -m dispatcher.serve --port 8000 \
  --model e5-small-tuned --backend torch --device cuda \
  --arbiter laya --thin-rescue --require-cuda

curl localhost:8000/health
```

Роutes: `POST /sessions/open|check|partial|final|cancel|close`, `POST /rtp`
(аудио от Go), `GET /health`. STT-движок подключается одной функцией
в `dispatcher/serve.py` (сейчас заглушка — текстовый путь работает полностью).

## 4. Go-оркестратор (звонки по команде)

```bash
cd orchestrator && go build -o orchestrator . && ./orchestrator
```

Env: `ARI_URL` (default `http://127.0.0.1:8088/ari`), `ARI_USER`, `ARI_PASS`,
`PY_URL` (default `http://127.0.0.1:8000`), `RTP_BASE` (default 10000).
Слушает `:8080`.

## 5. Asterisk (регистрации + originate)

Конфиги: `asterisk/pjsip.conf` (транспорт UDP 5060, шаблон `[registrant]`,
только alaw), `asterisk/extensions.conf` (всё уходит в `Stasis(trainer,…)`).
Положить в `/etc/asterisk/` и перезагрузить. Регистрации — по шаблону
(прод: realtime). Подробно: `asterisk/README.md`.

## 6. Всё разом (compose)

```bash
docker compose up --build   # asterisk + python(GPU) + orchestrator(:8080)
```

## 7. Звонок по команде

```bash
curl -X POST localhost:8080/calls \
  -d '{"to":"op_001","scenario_id":"bilet04_call01"}'
# → 202 {"call_id":"...","scenario_id":"...","rtp_port":10000,...}

curl localhost:8080/calls          # активные звонки
curl -X DELETE localhost:8080/calls/<call_id>  # положить трубку
```

## 8. Нагрузка

Один GPU = один python-воркер, звонок липнет к воркеру до BYE.
SLO: VAD→аудио p95 < 800 мс. Потолки: Asterisk ~200–500 параллельных
на бокс, дальше Kamailio + ферма (переписывать ничего не надо).
Детали: `docs/highload.md`. Контракт слоёв: `docs/session-contract.md`.

## Состав изменений

| что | где |
| --- | --- |
| реестр звонков + сериализация `Reply` | `dispatcher/service.py` |
| предрендер аудио + слияние в `Fact.audio` + `audio_id` в рендере | `tools/synth_audio.py`, `dispatcher/data/loader.py`, `dispatcher/dialog/render.py` |
| сигналинг | `asterisk/` |
| медиа-тракт AudioSocket/VAD/barge-in | `dispatcher/media/asterisk.py` |
| оркестратор | `orchestrator/` |
| HTTP-сервер | `dispatcher/serve.py` |
| compose + доки | `docker-compose.yml`, `docs/` |
| тесты (паритет stream==text, медиа, сервер) | `tests/test_{service,audio,media,serve,replay_parity}.py` |
