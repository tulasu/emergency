set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# postgres+caddy — этот корень (cue -> docker-compose.yml)
root_compose := "docker-compose.yml"
# dispatcher — asterisk + python + orchestrator
disp_dir := "dispatcher"
disp_compose := "dispatcher/docker-compose.yml"
disp_env := "dispatcher/.env"

[private]
default:
    @just --list

# postgres+caddy (если ещё не сгенерён)
generate:
    cue export -f -e compose --outfile {{root_compose}} config/system.cue config/dev.cue

# инфраструктура корня
infra: generate
    docker compose -f {{root_compose}} up -d

# инфраструктура + диспетчер (CPU-only по .env, модель включается переменными)
up: generate
    docker compose -f {{root_compose}} up -d
    docker compose -f {{disp_compose}} --env-file {{disp_dir}}/.env up -d

# только диспетчер
up-disp:
    docker compose -f {{disp_compose}} --env-file {{disp_dir}}/.env up -d

down:
    docker compose -f {{root_compose}} down
    docker compose -f {{disp_compose}} down

logs-disp:
    docker compose -f {{disp_compose}} logs -f

# с включённой моделью:  just up-model
up-model:
    DISPATCHER_MODEL=e5-small-tuned DISPATCHER_ARBITER=laya \
        docker compose -f {{disp_compose}} --env-file {{disp_dir}}/.env up -d

test:
    cd traineebox; go test ./... -count=1 -timeout 10m

# звонок на MicroSIP: just call            — случайный сценарий
#                     just call bilet04_call02 op_test
call scenario="" to="op_test":
    sid="{{scenario}}"; [ -n "$sid" ] || sid=$(ls {{disp_dir}}/data/scenarios | grep '^bilet' | sed 's/\.json$//' | shuf -n1); \
    echo "сценарий: $sid"; \
    curl -s -XPOST localhost:8080/calls -d "{\"to\":\"{{to}}\",\"scenario_id\":\"$sid\"}"; echo
