set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

# postgres+caddy — этот корень (cue -> docker-compose.yml)
root_compose := "docker-compose.yml"

[private]
default:
    @just --list

# postgres+caddy (если ещё не сгенерён)
generate:
    cue export -f -e compose --outfile {{root_compose}} config/system.cue config/dev.cue

# инфраструктура корня
infra: generate
    docker compose -f {{root_compose}} up -d

up: generate
    docker compose -f {{root_compose}} up -d

down:
    docker compose -f {{root_compose}} down

test:
    cd traineebox; go test ./... -count=1 -timeout 10m

# звонок на MicroSIP: just call <scenario_id> [to]
call scenario to="op_test":
    echo "сценарий: {{scenario}}"; \
    curl -s -XPOST localhost:8080/calls -d "{\"to\":\"{{to}}\",\"scenario_id\":\"{{scenario}}\"}"; echo
