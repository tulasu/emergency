set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# единый compose: postgres + traineebox + dialog + ticketgen + asterisk + caddy
root_compose := "docker-compose.yml"
asterisk_compose := "config/asterisk/docker-compose.yml"

[private]
default:
    @just --list

# cue -> docker-compose.yml (источник правды — config/*.cue, yml не правим руками)
generate:
    cue export -f -e compose --outfile {{root_compose}} config/system.cue config/dev.cue

# всё из одной точки: инфра + миграции + seed-admin
# --build обязателен: иначе docker compose кэширует старый бинарь
# initializer'а и новые подкоманды (import-bank/import-scenarios/seed-demo)
# будут unknown command.
up: generate
    docker compose -f {{root_compose}} up -d --build
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer migrate up
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer seed-admin
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer import-bank
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer import-scenarios
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer seed-demo

down:
    docker compose -f {{root_compose}} down

# пересобрать образы без перезапуска контейнеров
build:
    docker compose -f {{root_compose}} build

logs *args:
    docker compose -f {{root_compose}} logs -f {{args}}

# отдельно при нужде (CI/прод)
migrate:
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer migrate up

seed:
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer seed-admin

import-bank:
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer import-bank

import-scenarios:
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer import-scenarios

seed-demo:
    docker compose -f {{root_compose}} run --rm traineebox /app/initializer seed-demo

# legacy: только asterisk в host network (локальный dialog на 127.0.0.1)
up-asterisk:
    docker compose -f {{asterisk_compose}} up -d

down-asterisk:
    docker compose -f {{asterisk_compose}} down

test:
    cd traineebox; go test ./... -count=1 -timeout 10m

# Звонок на MicroSIP: just call <attempt_id> [to=demo]. Требует jq.
# attempt_id — из последней строки `just seed-demo` (`attempt=<uuid>`).
# Никаких файлов на хосте не создаётся — копипаст из вывода.
call attempt_id to="demo":
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ -z "{{attempt_id}}" ]]; then
        echo "usage: just call <attempt_id> [to=demo]" >&2
        echo "run 'just seed-demo' and copy 'attempt=<uuid>' from its last line" >&2
        exit 1
    fi
    login_resp="$(curl -s -X POST -H 'Content-Type: application/json' \
        -d '{"login":"demo","password":"demo1234"}' http://localhost:8080/auth/login)"
    token="$(printf '%s' "$login_resp" | jq -r '.token // .session_token // empty')"
    if [[ -z "$token" ]]; then
        echo "login failed: $login_resp" >&2
        exit 1
    fi
    curl -s -X POST -H "Authorization: Bearer ${token}" \
        -H 'Content-Type: application/json' \
        -d "{\"to\":\"{{to}}\"}" \
        "http://localhost:8080/attempts/{{attempt_id}}/call"
    rc=$?
    echo
    exit $rc
