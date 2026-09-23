#!/bin/sh
# Старт воркера из переменных окружения (.env): пустая переменная — флаг не
# передаётся, argparse не спотыкается о «--ensemble ''».
set -e
set -- python3 -u -m dispatcher.serve --port "${DISPATCHER_PORT:-8000}"
[ -n "$DISPATCHER_MODEL" ] && set -- "$@" --model "$DISPATCHER_MODEL"
[ -n "$DISPATCHER_BACKEND" ] && set -- "$@" --backend "$DISPATCHER_BACKEND"
[ -n "$DISPATCHER_DEVICE" ] && set -- "$@" --device "$DISPATCHER_DEVICE"
[ -n "$DISPATCHER_ARBITER" ] && set -- "$@" --arbiter "$DISPATCHER_ARBITER"
[ -n "$DISPATCHER_STT" ] && set -- "$@" --stt "$DISPATCHER_STT"
[ -n "$DISPATCHER_ENSEMBLE" ] && set -- "$@" --ensemble "$DISPATCHER_ENSEMBLE"
[ "$DISPATCHER_IMPROV" = "1" ] && set -- "$@" --improv
echo "$@"
exec "$@"
