# Шаг 3. SIP-сигналинг на Asterisk

Состояние звонка в Asterisk не хранится — только две channel-переменные,
поэтому ферма за Kamailio не потребует переделки (шаг 7).

## Регистрации

`pjsip.conf`: транспорт UDP 5060, шаблон `[registrant]` (alaw only,
`rewrite_contact` для NAT). Прод — realtime или include-файлы.

## Звонок по команде

Go-оркестратор (шаг 5) делает ARI originate:

```bash
curl -u user:pass -X POST "http://asterisk:8088/ari/channels" \
  -d endpoint=PJSIP/op_001 -d extension=s -d context=trainer-out \
  -d "variables=scenario_id=bilet04_call01,call_id=<uuid4hex>"
```

Dialplan отдаёт канал в Stasis-приложение `trainer` — дальше медиа-сервис
(шаг 4). Ошибки звонка (нет регистрации → 404 от ARI) — наружу как есть.

## Потолок

~200–500 параллельных на бокс. Дальше: Kamailio впереди + ещё Asterisk,
Go/Python не меняются. Свой Go-B2BUA не пишем: готовый SIP/RTP-стек
дешевле месяцев отладки re-INVITE/NAT/SRTP.
