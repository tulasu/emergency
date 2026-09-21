set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

compose_file := "docker-compose.yml"
version := trim(read("VERSION"))

[private]
default:
    @just --list

generate:
    cue export -f -e compose --outfile {{compose_file}} config/system.cue config/dev.cue

up: generate
    docker compose -f {{compose_file}} up -d

down:
    docker compose -f {{compose_file}} down

build-app:
    cd traineebox; go build -trimpath -ldflags="-s -w -X main.version={{version}}" -o ../bin/traineebox-app.exe ./cmd/app
