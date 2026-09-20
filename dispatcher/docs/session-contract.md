# Шаг 1. Контракт встройки: один звонок = одна сессия

Статус: только план + интерфейс, кода нет.
Граница шага: фиксируем, как SIP-конвейер `VAD → STT → понимание → TTS`
разговаривает с `dispatcher/session.py`. Ничего не меняем в ядре.

## 1. Формула

```
один SIP-звонок = один Session.open(scenario, cascade)
```

`Session` уже покрывает весь контракт (`session.py`, `test_session.py`).
Медиа-сервис не выдумывает состояние — только держит реестр
`session_id → Session` и проксирует события конвейера в методы сессии.

| событие конвейера | метод | возврат |
| --- | --- | --- |
| ответ на звонок | `Open(scenario_id)` → `opening()` | текст приветствия (играть первым) |
| STT partial | `on_partial(text)` | ничего |
| VAD «договорил» | `on_final(text)` | `Reply(text, audio_id, style, mood)` → в TTS |
| barge-in (оператор перебил) | `cancel()` | `audio_id` оборванной реплики → остановить TTS |
| BYE / `ChannelDestroyed` | `close()` | ничего, сессия удаляется из реестра |

Правила, уже зашитые в сессии и обязательные для транспорта:

- `on_partial` не порождает ход (`turns` пуст, наружу ничего) — частички
  можно слать вслепую, фильтр `<6 символов / дубликат` внутри.
- `on_final` без предварительных partial обязан давать тот же ответ,
  что с ними (спекуляция `_draft` по совпадению текста).
- `cancel` снимает спекуляцию и возвращает `audio_id` текущей реплики
  (или `None`, если рвать нечего). Вызов без partial — безопасен.
- `close` вызывает `cascade.save()` (сейчас no-op) и освобождает слот реестра.

## 2. Методы (эскиз, не код)

```proto
service Dispatcher {
  rpc Open(OpenRequest)     returns (OpenResponse);   // Session.open + opening
  rpc Partial(PartialRequest) returns (Empty);        // on_partial
  rpc Final(FinalRequest)   returns (Reply);          // on_final
  rpc Cancel(CancelRequest) returns (CancelResponse); // cancel
  rpc Close(CloseRequest)   returns (Empty);          // close
}
message OpenRequest  { string scenario_id = 1; string session_id = 2; }
message OpenResponse { string opening_text = 1; }
message PartialRequest { string session_id = 1; string text = 2; }
message FinalRequest   { string session_id = 1; string text = 2; }
message CancelRequest  { string session_id = 1; }
message CancelResponse { optional string stop_audio_id = 1; }
message CloseRequest   { string session_id = 1; }
message Reply { string text = 1; optional string audio_id = 2;
                string style = 3; int32 mood = 4; }
```

HTTP-эквивалент 1:1: `POST /sessions/open|partial|final|cancel|close`.
`style` — строка из `Style` (`plain/short/confirm/correct/dont_know/
mishear/slow_down/ack`), `mood` — `0..2`. `audio_id` пока всегда `None`
(место под пред-отрендеренное аудио, заполнит TTS-шаг).

Ошибки: неизвестный `scenario_id` на `Open` → `NOT_FOUND` (404),
сессии нет — звонок не открывается; любой метод по неизвестному/закрытому
`session_id` → `NOT_FOUND`; `Close` идемпотентен (повтор — no-op).

## 3. `session_id`

`session_id = call_id` из шага 3 (`sip-plan.md`): `uuid4 hex`, генерирует
`calls-gateway` при `POST /calls`, дальше едет в channel-переменной
и во всех запросах медиа-сервиса. Отдельный формат не вводим.
Сценарий привязывается один раз в `Open` и не меняется до `Close`.

## 4. Что общее, что на звонок (находки scout)

| объект | время жизни | почему |
| --- | --- | --- |
| `Ontology`, `LexicalBank`, `Encoder`, `VectorBank`, `LayaArbiter` | **singleton** на процесс, строятся раз при старте | тяжёлые (энкодер, векторы `data/build`, прогрев роутера ~минута), дальше только чтение |
| `Cascade` / `LayaUnderstander` | **per-call**, новый на каждый `Open` | `Cascade.state: NluState` мутабелен (`last_slots/last_family/turn`), плюс срез `scoped`/`values` под сценарий — шарить между звонками нельзя |
| `Session` (+ `Renderer`, `CallState`) | **per-call** | журнал `turns`, спекуляция `_draft`, `seed` звонка |

`Understander` — протокол (`understand/preview/save`), сессия не знает,
что внутри: `Cascade` (банк) и `LayaUnderstander` (целиком модель)
взаимозаменяемы. `preview` обязан не мутировать состояние
(у `Cascade` так, у `LayaUnderstander` — no-op `→ None`).
Вывод для транспорта: partial/final одного звонка вызывать строго
последовательно (один поток/очередь на `session_id`); синглтоны потокобезопасны,
т.к. после сборки только читаются.

Сборка на `Open` повторяет `cli._build`: `scenario = load_all()[scenario_id]`,
`Cascade(scenario, lexical=SINGLETON_BANK, vectors=SINGLETON_VECTORS, ...)`,
`Session.open(scenario, cascade, onto)`.

## 5. `say()` как референс

`say(text)` = `on_final(text)` без partial — текстовый дубль звонка.
Требование к любому транспорту: `replay --stream` обязан совпадать
с `replay` посимвольно (тест `test_stream_matches_text`).
Отладка без SIP: `cli chat/replay <scenario_id>` — эталон; расхождение
звонка с ним — баг транспорта, не ядра. Новую обвязку принимать только
прогоном `tests/test_session.py` через её методы.

## 6. Что НЕ в этом шаге

Реализации gRPC/HTTP, реестра сессий, RTP/STT/TTS-обвязки, конфигов Asterisk.
Следующий шаг — медиа-сервис поверх этого контракта.
