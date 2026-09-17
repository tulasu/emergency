# Semantic NLU prototype (рядом с trainer)

Изолированный прототип: реплика → ключ факта через CPU sentence embeddings.
Диалоговый `trainer/` не меняется и в Engine не подключён.

## Идея

```text
score(key) = max cosine(utterance, q) for q in fact.questions
+ threshold / margin / strong-accept
+ negation heuristic
```

Модель STS: `sergeyzh/rubert-tiny-sts` (CPU).

## Установка

```bash
cd new_core/trainer
pip install -r requirements.txt

cd ../semantic_nlu
pip install -r requirements.txt
```

Первый запуск STS скачает модель с Hugging Face (~110 MB).

Опционально — третий backend (локальный LLM, без LM Studio):

```bash
pip install -r requirements-llm.txt
```

Первый прогон с `local_llm` скачает GGUF `Qwen2.5-0.5B-Instruct` Q4 (~0.4–0.7 GB) в кэш HF.

## Чат (полная переписка)

```bash
cd new_core/semantic_nlu
python -m chat bilet04_call01
python -m chat bilet04_call01 --debug
```

## Прогон бенча

Lexical + semantic + local_llm:

```bash
cd new_core/semantic_nlu
python -m bench --all --backends lexical,semantic,local_llm
```

Только lexical/semantic:

```bash
python -m bench --all --backends lexical,semantic
```

Свой GGUF:

```bash
python -m bench --all --backends local_llm --llm-gguf D:\models\model.gguf
```

На Windows: `$env:PYTHONIOENCODING='utf-8'`.

## Метрики

- exact key-set / top-1 / false accept / reject ok
- latency p50/p95/avg
- agreement lexical <-> semantic (если оба включены)

## Результаты прогона

### 3-way (lexical / semantic / local_llm), 2026-09-17, CPU

`rubert-tiny-sts` + `Qwen2.5-0.5B-Instruct` Q4_K_M via `llama-cpp-python`.

| backend | exact (64) | p50 latency | notes |
|---------|------------|-------------|-------|
| lexical | **83%** (53) | ~1 ms | false accept на negation |
| semantic | **92%** (59) | ~5 ms | после max+strong gate |
| local_llm | **28%** (18) | ~280 ms | cold load ~0.4 s; часто пустой/битый JSON |

Agreement lexical↔semantic: **81%** (52/64).

Вывод: для этого классификатора 0.5B Instruct на CPU **не конкурент** STS/lexical по качеству; latency ~50× semantic. Имеет смысл пробовать ≥1.5B GGUF (`--llm-gguf`), но latency вырастет.

### STS earlier snapshot (до max/strong)

| TOTAL | lexical 83% | semantic 78% | ~1 ms / ~5 ms |
