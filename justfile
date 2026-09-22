set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

compose_file := "docker-compose.yml"

[private]
default:
    @just --list

generate:
    cue export -f -e compose --outfile {{compose_file}} config/system.cue config/dev.cue

up: generate
    docker compose -f {{compose_file}} up -d

down:
    docker compose -f {{compose_file}} down

test:
    cd traineebox; go test ./... -count=1 -timeout 10m
