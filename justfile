set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

compose_file := "docker-compose.yml"

[private]
default:
    @just --list

generate:
    cue export -f -e compose --outfile {{compose_file}} system.cue dev.cue

up: generate
    docker compose -f {{compose_file}} up -d

down:
    docker compose -f {{compose_file}} down
