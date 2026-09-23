#!/bin/sh
# llama-server на GPU: голос LLM в каскаде и ответы вне сценария.
# CUDA 13 runtime берётся из колёс torch в .venv — отдельный toolkit не нужен.
#   tools/llama_server.sh [модель.gguf] [порт]
cd "$(dirname "$0")/.." || exit 1
D=models/llama/cuda13/llama-b11062
MODEL=${1:-models/llama/Qwen3-1.7B-Q4_K_M.gguf}
PORT=${2:-8081}
export LD_LIBRARY_PATH="$D:$PWD/.venv/lib/python3.12/site-packages/nvidia/cu13/lib:$LD_LIBRARY_PATH"
exec "$D/llama-server" -m "$MODEL" -ngl 99 -c 4096 -np 2 \
  --host 127.0.0.1 --port "$PORT" --no-webui
