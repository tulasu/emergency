# Шаг 7. Нагрузка и SLO

Топология: `Asterisk ×K → Go ×N (stateless) → Python GPU ×M (stateful)`.
Звонок липнет к одному Python-воркеру (`session_id = call_id`, sticky до BYE):
`NluState`/`CallState`/спекуляция `_draft` переезд не переживут.

## Правила воркера

- Старт только с `--require-cuda` (fail-fast, без молчаливого CPU-фолбэка)
  и прод-флагами: `--model e5-small-tuned --backend torch --device cuda
  --arbiter laya --thin-rescue`. VRAM ~2 ГБ (448 МБ e5 + 1.6 ГБ laya).
- `backend=auto` в проде запрещён (тихо выбирает onnx на CPU).
- Readiness — после прогрева (`/health` 200; старт до 2 мин).
- Смешанный пресет запрещён: tuned-веса + lexical-пороги = отказ от отказов.

## SLO и метрики

- Бюджет VAD→аудио p95 < 800 мс. Главный инструмент — спекуляция
  `on_partial` (ответ готов к `on_final`), не оптимизация энкодера.
- Готовое: `Understanding.latency_ms` (NLU), счётчики `LayaArbiter`
  (calls/refusals/failures). Доли `dont_know` (пробел онтологии → `lint`) и
  `mishear` (`topical==0` → чинить аудио, не NLU) — алерты раздельно.
- Prometheus-экспорт — когда упрёмся в логи (ponytail: пока хватит
  `/health` + latency из журнала `Turn`).

## Проверки перед продом

1. `bench/thread_probe.py`: `encode`/`router.predict` из 8 потоков —
   CUDA-ошибки/дрейф → по `Lock` на каждую секцию.
2. Нагрузочный `Final` 1/2/4/8 конкурентных → p95 `latency_ms`;
   лимит inflight = последний без роста. Переполнение — деградация
   в лексику, не обрыв звонка.
3. Упал Python → Go переоткрывает звонок на живом (контекст с нуля,
   метрика `session_migrated_total`); упал Asterisk → его звонки рвутся,
   новые идут на живой (`asterisk_id` в `session_id`, коллизий нет).
