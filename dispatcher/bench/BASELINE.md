# База NLU: лексика vs majority-ensemble (laya + LLM)

Замерено 2026-09-27 на RTX 4050 (драйвер 595.91.07, CUDA 13.2).
Точка отсчёта для идей 2 (реранкер), 3 (softmax-голова), 4 (LoRA-LLM):
любое изменение сравнивать с обеими строками — live и blind.

## Стек

- Образ `dispatcher-python:latest`: torch 2.14.0+cu130, `cuda True`, `laya ok`
  (проверено `docker run --gpus all`).
- LLM: `artifacts/models/llama/Qwen3-1.7B-Q8_0.gguf` через compose-сервис
  `llama` (`ghcr.io/ggml-org/llama.cpp:server-cuda`, порт 8081).
- Веса laya: volume `emergency_dialog_models` (646.9M, multilingual снапшот),
  монтируется в `/app/models`, `HF_HOME=/app/models`, `HF_HUB_DISABLE_XET=1`.
- Каскад в обоих прогонах **лексический без векторов** (`Thresholds.preset("lexical")`).
  Пресет `fused` протух после вычета фона слота: `e5-small-torch:torch-cuda`
  падает с `ValueError` — векторы в базу не входят.
- `dispatcher/` восстановлен из `b47b6a5` (staged); `bench/run.py` пропатчен
  флагами `--ensemble/--voters/--laya-min` (unstaged) — blind-ensemble
  считается им, итог `total` без подк ключей «  из них …».

## Команды (все из корня репо)

```bash
docker compose up -d llama
curl -s -m 5 http://127.0.0.1:8081/health  # {"status":"ok"}

# live: 91 размеченная реплика STT-стиля (data/gold/live_calls.tsv)
docker run --rm -v $PWD/dispatcher:/app -w /app dispatcher-python:latest \
  python3 -m bench.live
docker run --rm --gpus all --network host -v $PWD/dispatcher:/app -w /app \
  -v emergency_dialog_models:/app/models -e HF_HOME=/app/models \
  -e HF_HUB_DISABLE_XET=1 -e DISPATCHER_LLM_URL=http://127.0.0.1:8081 \
  dispatcher-python:latest \
  python3 -m bench.live --ensemble majority --voters laya,llm --device cuda --candidate-scope all

# blind: слепые наборы, сквозной прогон через decide (data/blind/)
docker run --rm -v $PWD/dispatcher:/app -w /app dispatcher-python:latest \
  python3 -m bench.run
docker run --rm -v $PWD/dispatcher:/app -w /app dispatcher-python:latest \
  python3 -m bench.run --fresh
docker run --rm --gpus all --network host -v $PWD/dispatcher:/app -w /app \
  -v emergency_dialog_models:/app/models -e HF_HOME=/app/models \
  -e HF_HUB_DISABLE_XET=1 -e DISPATCHER_LLM_URL=http://127.0.0.1:8081 \
  dispatcher-python:latest \
  python3 -m bench.run --ensemble majority --voters laya,llm --device cuda --candidate-scope all
```

Абляции на live гонялись той же командой `--ensemble majority --candidate-scope all`
с `--voters laya` / `--voters llm` и `--arbiter laya` без `--ensemble`.

## live_calls.tsv: понимание (91 реплика)

| Конфиг | Верно | мс/реплика |
| --- | --- | --- |
| лексика | 58/91 = 63.7% | 2 |
| лексика + арбитр laya (cuda, серая зона) | 59/91 = 64.8% | 5 |
| majority [laya] (cuda) | 58/91 = 63.7% | 16 |
| majority [llm] | 58/91 = 63.7% | 29 |
| **majority [laya,llm] (cuda) — прод** | **70/91 = 76.9%** | 38 |

Промахи прода: incident 12, victim 5, caller 4 (было 18/8/5/1/1 у лексики:
incident 18, victim 8, caller 5, addr 1, service 1).
Laya и LLM поодиночке на live дают ноль — эффект только в связке (+13.2 п.п.).

## blind: сквозной прогон (96 сценариев)

Пилот расширения 2026-09-27: +32 train-формулировки в 4 бедных слота
(hazard.gas 5→13, env.weather 5→13, meta.stay_on_line 1→9,
incident.ongoing 9→17 — половина в bilet18_call03, половина в
bilet09_call03) и +24 blind-зонда пачками после якорей в 3 файлах
(04/06/18). Норм-пересечение train∩blind пустое, проверено скриптом
перед вставкой. Строка «до» — слепок выше, строки «после» — новые замеры
на том же стеке.

| Исход | лексика до (6284) | лексика после (6308) | majority [laya,llm] до | majority [laya,llm] после |
| --- | --- | --- | --- | --- |
| ответ | 3854 = 61.3% | 3856 = 61.1% | 3418 = 54.4% | **3461 = 54.9%** |
| из них ансамблем | — | — | 118 = 1.9% | 106 = 1.7% |
| не знаю | 1649 = 26.2% | 1646 = 26.1% | 2059 = 32.8% | 2030 = 32.2% |
| речевой акт | 346 = 5.5% | 349 = 5.5% | 346 = 5.5% | 349 = 5.5% |
| не расслышала | 7 = 0.1% | 7 = 0.1% | 7 = 0.1% | 7 = 0.1% |
| мимо | 428 = 6.8% | 450 = 7.1% | 454 = 7.2% | 461 = 7.3% |
| критичные факты | 526/541 = 97.2% | 526/541 = 97.2% | 522/541 = 96.5% | 526/541 = 97.2% |
| медиана ответов по сценарию | 61.7% | 61.7% | 56.7% | 55.0% |
| понимание p50 / p95, мс | 0.90 / 2.36 | 0.85 / 2.16 | 39.70 / 51.33 | 40.41 / 52.54 |

Худшие сценарии прода после: bilet05_call03 38%, bilet11_call01 38%,
bilet19_call02 38%, bilet12_call02 40%.
Лучшие после: bilet27_call03 77%, bilet23_call03 78%, bilet06_call03 79%.
(До: худшие bilet05_call03 37%, bilet08_call02 37%, bilet12_call02 39%,
bilet02_call02 40%; лучшие bilet27_call01 72%, bilet28_call01 75%,
bilet23_call03 78%. У лексики после: худшие bilet05_call03 43%,
bilet12_call02 44%, bilet11_call01 47%, bilet12_call03 47%; лучшие
bilet06_call03 79%, bilet23_call03 80%, bilet27_call03 85%.)

Голоса прода на blind после: LayaArbiter вызовов 5388, отказов 2128,
сбоев 0; LlmArbiter вызовов 5388, сбоев 0.
LOO после пилота (7501 формулировка): top-1 76.7%, медиана 78.5%.
Прямая проверка 24 зондов `cascade.understand`: газ 6/6 `hazard.gas`
(0.61–0.99); погода 5/6 (`Погодные условия какие?` → мимо, score 0.083);
линия 3/6 (`Трубку не вешайте`, `Оставайтесь на телефоне`,
`Ждите, не кладите трубку` → `rules` SPEECH_ACT, хотя top банка верный
0.38–0.98); ongoing 5/6 (`Мужчина сейчас кричит?` → `victim.sex` 0.715
против 0.646 — лемма `мужчина` перевешивает `кричать`).

### Разбор новых мимо (пилот)

Мимо лексики 428→450 (+22): ни один из 24 зондов в `--show` не светится —
все они либо отвечают, либо уходят в `rules`/чужой слот, но не в `мимо`.
Прирост — соседи: новые формулировки подняли idf-фон и перетянули
пограничные реплики других сценариев (видно по смене худших: bilet11_call01
и bilet12_call03 вошли в хвост вместо bilet29_call03).

Воры поименно:

- `Мужчина сейчас кричит?` → `victim.sex`: леммы {кричать, мужчина};
  у вора `Мужчина или женщина?` {женщина, мужчина} + `То есть кричит
  именно мужчина` {именно, кричать, мужчина} — `мужчина` бьёт `кричать`.
  Лечится данными: ongoing-формулировкам с `мужчина/он` не хватает массы
  против 11 victim.sex. Ансамбль тут уже прав: laya воздержалась (0.39),
  LLM отдал `incident.ongoing` — каскад проиграл голосованием 1:2.
- `Погодные условия какие?` → мимо: лемм `погодный/условие` нет ни в одной
  из 13 env.weather (`ветер/дождь/мороз/снег/погода` — `погода` в зонде
  тоже нет, есть только в train `какая сейчас погода`). Лечится данными:
  добавить 1–2 train с голым `погода/погодные условия`.
- `Трубку не вешайте` / `Оставайтесь на телефоне` / `Ждите, не кладите
  трубку` → `rules` SPEECH_ACT (`успокоение`: `не вешайте|оставайтесь|
  ждите|не кладите`): паттерн шире слота. Лечится правилом, не данными:
  вопрос про линию с `?` и топом банка ≥0.3 не должен глохнуть в
  `SPEECH_ACT`. Триггер узкий — только `успокоение` при наличии `?`.

Симуляция `--fresh` (свой сценарий вычеркнут): зонды не переносятся —
газ уходит в `building.gas`, погода в мимо/`fire.smoke`/`addr.spot`,
линия в `addr.refined`/`med.consciousness`. Пилотные слоты односценарные
(hazard.gas/env.weather/meta.stay_on_line — по 1 владельцу), поэтому
перенос на новую ситуацию дадут только слоты с ≥2 владельцами.
Вывод: данные для населённости, пороги/правила для механики; следующий
пилот — слот с 2+ владельцами (caller.location 14: bilet04_call01 +
bilet05_call01).

## Что это значит для идей 2/3/4

- Голосование поднимает понимание размеченных реплик (+13.2 п.п. live),
  но в сквозном прогоне переводит ответы в «не знаю»: −6.9 п.п. ответов,
  −0.7 п.п. критичных, мимо +0.4 п.п. Цель изменений — blind-ответы и мимо
  при неупавших критичных; live один недостаточен.
- Кандидаты голосования здесь — все 88 слотов (`dispatcher` `candidates()`),
  прод `dialog/` режет их до слотов сценария (ponytail-фикс). Замер через
  `dispatcher.Service` ≠ прод один в один — расхождение зафиксировано,
  выравнивание кандидатов следующая задача.
- Тесты: `../dialog/.venv/bin/python -m pytest tests/ -q` → 155 passed.

## Эксперимент: cross-encoder voter (2026-09-27)

`tools.train_reranker` обучает бинарный BertForSequenceClassification на
`intfloat/multilingual-e5-small`: вход — реплика и русское название слота,
выход — вероятность соответствия. `bench.rerank_data` строит top-5
лексического банка **без своего сценария** (7393 строки: 5721 с верным
слотом и 1672 без него; 34576 пар). В обучении нет live/blind.
Модель: 2 эпохи, batch 32, CUDA; весы в игнорируемом
`data/build/reranker/`, конфигурация обучения в `training.json`.
При смене сценариев или онтологии данные и веса нужно пересобрать.

```bash
docker run --rm -v \"$PWD/dispatcher:/app\" -w /app dispatcher-python:latest \
  python3 -m bench.rerank_data
# e5-small должен быть доступен локально; путь ниже — снапшот этого замера
docker run --rm --gpus all -v \"$PWD/dispatcher:/app\" -w /app \
  dispatcher-python:latest python3 -m tools.train_reranker \
  --source models/models--intfloat--multilingual-e5-small/snapshots/614241f622f53c4eeff9890bdc4f31cfecc418b3
docker run --rm --gpus all --network host -v \"$PWD/dispatcher:/app\" -w /app \
  -e DISPATCHER_LLM_URL=http://127.0.0.1:8081 dispatcher-python:latest \
  python3 -m bench.run --ensemble majority --voters reranker,llm \
  --device cuda --candidate-scope all
# аналогично: python3 -m bench.live --ensemble majority --voters reranker,llm --device cuda --candidate-scope all
```

На момент эксперимента blind содержит **6316**, а зафиксированная база выше —
**6308** реплик. Поэтому прямое сравнение числителей не является парным;
разницу в 8 реплик в исходных blind TXT здесь не откатывали. Все три
пороговых прогона — на одинаковых 6316 репликах.

| min_score / min_margin | live | blind-ответы | blind-мимо | критичные |
| --- | --- | --- | --- | --- |
| база majority [laya,llm], 6308 | 70/91 | 3461/6308 = 54.9% | 461/6308 = 7.3% | 526/541 |
| reranker,llm 0.50/0.05 | 67/91 | 3631/6316 = 57.5% | 427/6316 = 6.8% | 525/541 |
| reranker,llm 0.40/0.03 | — | 3609/6316 = 57.1% | 415/6316 = 6.6% | 525/541 |
| **reranker,llm 0.30/0.02** | **69/91** | **3599/6316 = 57.0%** | **403/6316 = 6.4%** | **526/541** |

Из проверенных порогов только 0.30/0.02 сохраняет критичные на уровне
базы. Он выбран по blind; live на той же конфигурации на 1/91 ниже базы.
Ответов больше и «мимо» меньше исторического ансамбля с top-5 из всех
88 слотов, но 57.0% ниже лексического baseline 61.1%. Эта таблица
**непарная**; продовый состав voter по умолчанию не менялся.
В `dispatcher.serve` опция: `--ensemble majority --voters reranker,llm
--device cuda` (веса должны лежать в `data/build/reranker` либо передать
`--rerank-path`). Исторический замер — top-5 по всем 88 слотам dispatcher,
не ограниченные слоты продового `dialog/`. Тогда тесты: 161 passed.

## Парное сравнение с кандидатами как в dialog/ (2026-09-27)

`dialog/core/nlu/cascade.py:candidates` выбирает top-5 **из слотов
сценария**, затем `Ensemble` добавляет слот каскада, если он не попал
в пятёрку (даже если он вне сценария). В `dispatcher` тот же отбор
доступен через `--candidate-scope scenario` в `bench.run` и `bench.live`;
это значение по умолчанию **только для этих бенчмарков**. Прежний
`dispatcher.Service` и `Ensemble` по умолчанию по-прежнему берут top-5
из всех слотов; для исторического режима бенчмарка — `--candidate-scope all`.
Сам `dialog/` и продовый состав laya+LLM не менялись.

На одном и том же наборе 6316 реплик (96 сценариев), одна и та же
лексика и Qwen-сервис, только состав первого voter различается:

| Кандидаты: слоты сценария | [laya,llm] | [reranker,llm], пороги 0.30/0.02 |
| --- | ---: | ---: |
| blind-ответы | 3844/6316 = 60.9% | **4063/6316 = 64.3%** |
| blind-мимо | 542/6316 = 8.6% | **415/6316 = 6.6%** |
| критичные факты | 531/541 = 98.2% | **534/541 = 98.7%** |
| live | 69/91 | 69/91 |
| понимание blind p50/p95 | 39.83/47.01 мс | 34.12/45.94 мс |

`bench.pair` проверил совпадение `(scenario, index, text)` во всех 6316
строках. По исходам: 261 переход в «ответ» (131 из «не знаю», 130 из
«мимо»), 42 потери ответа (15 в «не знаю», 27 в «мимо»); чистый прирост
219 ответов. Приобретены критичные `bilet14_call01:addr.road`,
`bilet25_call01:addr.refined`, `bilet26_call01:addr.refined`; потерь нет.
Пороги реранкера подобраны ранее на top-5 всех слотов и здесь
**не перенастраивались**. Обучающая выборка тоже строилась с all-top-5;
парный результат не доказывает качество после переобучения на scoped-top-5.
Blind здесь тот же для обоих, но всё ещё отличается на 8 реплик от таблицы
старой базы. В blind «ответ» означает любой раскрытый факт сценария,
**не подтверждает правильность слота вопроса**: при HTTP-smoke в режиме
scoped на «Где вы сейчас находитесь?» (bilet04_call01) реранкер ответил
улицей происшествия вместо расположения заявителя. На live прироста нет;
вывод о семантической точности по одному росту blind-ответов делать нельзя.
Совпадение отбора кандидатов не означает, что замер `dispatcher`
воспроизводит весь `dialog/` один в один.

```bash
# из корня репозитория, Qwen на 8081; GPU и laya volume как в командах выше
docker run --rm --gpus all --network host -v \"$PWD/dispatcher:/app\" -w /app \
  -v emergency_dialog_models:/app/models -e HF_HOME=/app/models \
  -e HF_HUB_DISABLE_XET=1 -e DISPATCHER_LLM_URL=http://127.0.0.1:8081 \
  dispatcher-python:latest python3 -m bench.run --ensemble majority \
  --voters laya,llm --device cuda --candidate-scope scenario \
  --outcomes data/build/baseline-scenario.jsonl
docker run --rm --gpus all --network host -v \"$PWD/dispatcher:/app\" -w /app \
  -e DISPATCHER_LLM_URL=http://127.0.0.1:8081 dispatcher-python:latest \
  python3 -m bench.run --ensemble majority --voters reranker,llm \
  --device cuda --candidate-scope scenario \
  --outcomes data/build/reranker-scenario.jsonl
docker run --rm -v \"$PWD/dispatcher:/app\" -w /app dispatcher-python:latest \
  python3 -m bench.pair data/build/baseline-scenario.jsonl data/build/reranker-scenario.jsonl
```

Контрольный повтор старого top-5 из всех слотов на новом blind-наборе
остановлен: laya не уместилась в GPU и переключилась на CPU. Его результаты
не включены в таблицу. Тесты `dispatcher/tests/`: 163 passed.

## Разбор scoped-ошибок и коррекция фактов (2026-09-27)

В парных JSONL исходной таблицы 261 переход `не знаю/мимо → ответ`:
131 из `не знаю`, 130 из `мимо`. Во всех 131 исходных `не знаю`
каскад выбрал слот без прямого факта сценария. Это сигнал риска, **не**
разметка истинности: `bilet01_call01:20` («Вы сами на безопасном расстоянии?»)
стал ответом про ориентир, `:58` (температура воздуха) — адресом,
`bilet05_call02:15` (названия препаратов) — адресом. Но
`bilet06_call02:103` («Номер 916-897-56-23 — так?») корректно перешёл
к `caller.phone`. Описанный выше HTTP-промах для `bilet04_call01`
не тождественен blind-реплике `:43` («Вы где находитесь сейчас?»):
она раскрыла `caller.location` в обоих JSONL.

Из семи критичных, не раскрытых ни одной из исходных конфигураций,
четыре не имеют соответствующего вопроса в blind (`bilet09_call01:addr.refined`,
`bilet11_call01:addr.building`, `bilet13_call02:addr.direction`,
`bilet19_call02:caller.phone`). `bilet23_call01:access.code` тоже не
спрашивают: «Дверь заперта?» требует `access.door`, а не код домофона.
`bilet23_call02:addr.refined` заблокирован предусловием `addr.full` в
той же реплике. Для `bilet27_call03:structure.damage#3` вопрос о длине
есть, но три факта общего слота всегда раскрывались с начала списка,
максимум два за ход. Более того, исходные три «новых критичных» реранкера
получены на вопросах **не о них**: `bilet14_call01:42` (метры до
колонок → километр трассы), `bilet25_call01:44` (документы →
уточнённый адрес), `bilet26_call01:46` (вход → почтовый адрес).
Метрика критичных считает факт по ключу, не правильность вопроса.

Испытанный защитный запрет замены чужого слота был **отклонён и удалён**:
на тех же 6316 строках `[reranker,llm]` ответы 4063→3893,
«мимо» 415→397, критичные 534→528/541, live 69→63/91.
Он предотвращал некоторые ложные раскрытия, но лишал реальных
исправлений каскада (в частности, «Что произошло?» нередко ошибочно
распознано им как чужой `structure.object`).

Принята только локальная коррекция нескольких фактов *одного* слота:
`Cascade._wrap` по вопросам сценария выбирает один факт, если
совпадение лемм однозначно; иначе прежний набор фактов сохраняется.
`bilet27_call03:9` («Какой длины эта трещина примерно?»):
до `structure.damage,structure.damage#2` (плитка/комната),
после — `structure.damage#3` («Примерно метр длиной»).
`bilet01_call03:19` («Какая именно нога пострадала?») теперь раскрывает
сведение о ноге вместо одновременно руки и ноги. На новых парных
6316 строках:

| После коррекции | [laya,llm] | [reranker,llm] |
| --- | ---: | ---: |
| ответы | 3839 | 4063 |
| мимо | 545 | 415 |
| критичные | 532/541 | 535/541 |
| live | 69/91 | 69/91 |

`bench.pair` подтвердил совпадение всех входных строк и приобретение
`bilet27_call03:structure.damage#3` без потерь критичных в обоих
повторных замерах. У реранкера исходы `не знаю↔ответ` поменялись
по одному в каждую сторону; у laya 5 ответов меньше и 3 «мимо»
больше прежнего прогона. Эти небольшие колебания при повторном
запросе Qwen не приписываем отбору факта. Обновлённые ignored-файлы:
`data/build/baseline-facts.jsonl`, `reranker-facts.jsonl`.
Ошибочные `не знаю/мимо → ответ` ансамбля не устранены.

### Не полностью слепая выборка

Перед вставкой новой формулировки проверены все scenario-train вопросы
и текущие blind-реплики (lower, `ё→е`, пунктуация → пробел, нормализация
пробелов): **801/6316 blind-строк совпадают с train**; 490 — со своим
сценарием, 311 — с другим. Среди 91 live-реплики совпадают 13.
Примеры собственного пересечения: `bilet01_call01` «Пламя высокое?»,
«Сильно горит или только дым?». Поэтому существующий банк и
обученный ранее реранкер нельзя считать независимыми от *всех*
blind-фраз; `data/build/rerank.jsonl` содержит 1325/7393 строк с
таким совпадением. Старые таблицы не перемерялись и не переименовывались.
Ни одной новой формулировки в банк не внесено: без удаления/разметки
уже пересекающихся строк увеличение банка не даёт честного
подтверждения точности. `bench.holdout` в новых экспериментах использует
blind/live **только как стоп-лист совпадений**, не как обучающие пары.

### Замена богатой части банка softmax-головой 88+1

`tools.train_softmax` обучает линейную голову поверх замороженного
`multilingual-e5-small` на вопросах сценариев: 88 слотов плюс
`ни один`; негатив — формулировка слота, которого нет среди доступных
слотов *другого* сценария. При `thin_limit=40` голова заменяет оценки
34 населённых слотов каскада, оставляя 54 бедных слота лексическому
банку. Это **не дополнительный голосующий**. Локальный holdout
train-вопросов после исключения blind/live: класс слота 72.9%,
`ни один` 91.2%; для бедных слотов голова сама по себе 61.5%
(91 отложенная формулировка). Такой синтетический отказ не равен
реальным чужим вопросам. 1343 scenario-train формулировки исключены
из обучения из-за совпадения с blind/live; веса в ignored
`data/build/softmax-clean/`.

| Пара 6316, voter [reranker,llm] | банк | softmax-clean + банк бедных |
| --- | ---: | ---: |
| blind-ответы | 4063 | 3810 |
| blind-мимо | 415 | 158 |
| критичные | 535/541 | 528/541 |
| live | 69/91 | 76/91 |

`bench.pair` подтвердил 6316 одинаковых входов: потеряны 10 ключей
критичных, приобретены 3 (не обязательно по правильным вопросам).
Примеры отказа на ранее ложном ответе: `bilet05_call02:15`
(лекарства → адрес, после head — `не знаю`). Но
`bilet01_call01:58` (температура → адрес) остался ложным ответом.
Улучшение live и уменьшение «мимо» не компенсируют потерю критичных
и правильных ответов; голова включается только явным `--softmax-path`.
До исключения пересечений её неприемлемый результат был 3919 ответов,
162 «мимо», 524/541 критичных и 77/91 live; для решения использовать
только чистый вариант.

```bash
# из dispatcher/ внутри контейнера; из корня репо монтировать как в строках выше
python3 -m tools.train_softmax \
  --source models/models--intfloat--multilingual-e5-small/snapshots/614241f622f53c4eeff9890bdc4f31cfecc418b3 \
  --out data/build/softmax-clean --device cuda
python3 -m bench.run --ensemble majority --voters reranker,llm \
  --device cuda --candidate-scope scenario --softmax-path data/build/softmax-clean \
  --outcomes data/build/softmax-clean-scenario.jsonl
python3 -m bench.live --ensemble majority --voters reranker,llm \
  --device cuda --candidate-scope scenario --softmax-path data/build/softmax-clean
python3 -m bench.pair data/build/reranker-facts.jsonl \
  data/build/softmax-clean-scenario.jsonl
```

### QLoRA вместо LlmArbiter, 1:1

`tools.train_lora_arbiter` обучает настоящий LoRA-адаптер Qwen3-1.7B
(NF4, `q_proj/v_proj`, r=8) по `bench.rerank_data`: 7393 сценарных
train-вопроса с LOO top-5, метка `0`, если верного слота в списке нет.
Промпт, перестановка вариантов, ответ одной цифрой и контракт
`choose(text, slots) -> slot | None` общие с `LlmArbiter`;
`LoraLlmArbiter` берёт logits следующего токена локально. Это замена
**одного** voter, не четвёртый голос. Весов laya не касается.
Исходный checkpoint HF Qwen3-1.7B — ~4 ГБ в ignored
`data/build/qwen3-1.7b`, не прежний GGUF Q8. `Dockerfile.lora`
фиксирует PEFT/bitsandbytes, отсутствовавшие в базовом образе.
При `prepare_model_for_kbit_training` на этой GPU было OOM из-за
перевода большого embedding в fp32; фактическое обучение замораживало
базу в её dtype, включало gradient checkpointing и обучало только
адаптер (batch 1, накопление 8).

Фильтр `bench.holdout` исключил **1328** train-строк, совпадающих после
нормализации с blind/live: 5449 обучающих, 616 внутренних
отложенных. На этих внутренних формулировках выбор присутствующего
слота 289/461, выбор `0` 143/155. Это проверка train-формулировок,
не замена blind. Веса `data/build/lora-arbiter-clean/` ignored.

| Пара 6316, voter [reranker, X] | X = llm | X = lora-clean |
| --- | ---: | ---: |
| blind-ответы | 4063 | 3545 |
| blind-мимо | 415 | 1035 |
| критичные | 535/541 | 526/541 |
| live | 69/91 | 70/91 |
| отказы самого X на blind | не замерены у исходного LLM | 4001/5399 |

`bench.pair` подтвердил совпадение 6316 входов: потеряны девять
ключей критичных, приобретённых нет. `bilet01_call01:20`
(безопасное расстояние) и `bilet05_call02:15` (лекарства) перестали
вызывать посторонний факт: вместо ответа теперь «не знаю».
Но `bilet06_call02:103` (правильно повторённый номер телефона)
перешёл из корректного ответа в «мимо». «Ни о чём» модель теперь
выбирает, но слишком часто; +1 на live не компенсирует ложные
отказы и потерю критичных. Состав voter по умолчанию не меняется.
До исключения пересечений первая QLoRA давала 3961 ответ, 718 «мимо»,
526/541 критичных и 72/91 live, а отказов было 1866/5399.
Сопоставлять следует *очищенную* модель.

```bash
# из корня репозитория, базовый dispatcher-python:latest уже собран
docker build -f dispatcher/Dockerfile.lora -t dispatcher-lora:local dispatcher
docker run --rm -v "$PWD/dispatcher:/app" -w /app \
  -e HF_HUB_DISABLE_XET=1 dispatcher-python:latest \
  python3 -c 'from huggingface_hub import snapshot_download; snapshot_download(
    "Qwen/Qwen3-1.7B", local_dir="data/build/qwen3-1.7b",
    allow_patterns=["*.safetensors", "*.json", "merges.txt", "vocab.json"])'
# при отсутствии data/build/rerank.jsonl: bench.rerank_data из раздела выше
docker run --rm --gpus all --ipc host -v "$PWD/dispatcher:/app" -w /app \
  dispatcher-lora:local python3 -m tools.train_lora_arbiter \
  --base data/build/qwen3-1.7b --data data/build/rerank.jsonl \
  --out data/build/lora-arbiter-clean --batch 1 --accum 8
docker run --rm --gpus all --ipc host -v "$PWD/dispatcher:/app" -w /app \
  dispatcher-lora:local python3 -m bench.run --ensemble majority \
  --voters reranker,lora --device cuda --candidate-scope scenario \
  --outcomes data/build/lora-clean-scenario.jsonl
docker run --rm --gpus all --ipc host -v "$PWD/dispatcher:/app" -w /app \
  dispatcher-lora:local python3 -m bench.live --ensemble majority \
  --voters reranker,lora --device cuda --candidate-scope scenario
docker run --rm -v "$PWD/dispatcher:/app" -w /app dispatcher-python:latest \
  python3 -m bench.pair data/build/reranker-facts.jsonl \
  data/build/lora-clean-scenario.jsonl
```

`../dialog/.venv/bin/python -m pytest tests/ -q` из `dispatcher/`:
164 passed, одно предупреждение `audioop` (Python 3.13 deprecation).
Само совпадение исходов `bench.pair` не является разметкой правильности
вопроса/ответа; новых фактов в сценарии и обучающих формулировок в банк
не добавляли.

### Ручной многотуровый диалог с экспериментом

`dispatcher.cli chat` существовал, но раньше поднимал только отдельный
каскад без ансамбля. Теперь явные флаги включают **тот же** `Service`
с softmax-clean, `[reranker,llm]` и scoped-кандидатами; значения по
умолчанию `Service` не меняются. Из корня репозитория:

```bash
docker compose up -d llama
docker run --rm --gpus all --network host -it \
  -v "$PWD/dispatcher:/app" -w /app \
  -e DISPATCHER_LLM_URL=http://127.0.0.1:8081 dispatcher-python:latest \
  python3 -m dispatcher.cli chat bilet04_call01 --device cuda \
  --ensemble majority --voters reranker,llm --candidate-scope scenario \
  --softmax-path data/build/softmax-clean -v
# задавайте вопросы после «оператор>», пустая строка завершает звонок
# для парной ручной проверки повторить БЕЗ --softmax-path
```

Те же опции поддерживает отдельный `dispatcher.serve` с текстовыми
`POST /sessions/open`, `/sessions/final`, `/sessions/close`; для
изоляции выбрать свой `--port` и `--no-audiosocket`. Проверено
на 8193: четыре последовательные реплики сохраняли состояние
одного звонка. Вопросы об адресе, ориентире и местонахождении заявителя
получили нужные ответы. **Контрпример:** на «Скажите, газом пахнет?»
softmax-вариант ответил сразу ФИО заявителя *и* фактом об отсутствии
запаха; тот же диалог без softmax ответил только про газ.
`normalize.segments` разбивает ввод по запятой, а голова выдаёт оценку
богатых слотов даже для лишённого темы сегмента «Скажите».
Это выявленный риск ложного раскрытия, не «ещё один правильный ответ».

Это исполняемый `dispatcher` в текстовом режиме, **не** запущенный
`dialog/` в пользовательском веб/телефонном конвейере: у последнего
свой код NLU и softmax в него не подключён. Для проверки эксперимента
не меняйте работающий продовый контейнер `dialog`; используйте
отдельный интерактивный процесс или HTTP-порт.
