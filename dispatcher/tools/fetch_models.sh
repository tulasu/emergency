#!/usr/bin/env bash
# Скачивание всего, что нужно ядру сверх лексики.
#
#   bash tools/fetch_models.sh            всё
#   bash tools/fetch_models.sh vectors    только векторный слой
#   bash tools/fetch_models.sh arbiter    только арбитр
#
# Докачивает прерванное (curl -C -), уже скачанное не трогает.
set -u
cd "$(dirname "$0")/.." || exit 1
HF=https://huggingface.co
WHAT=${1:-all}

get() {  # get <папка> <url> <имя файла>
    mkdir -p "$1"
    if [ -s "$1/$3" ]; then echo "  уже есть: $1/$3"; return; fi
    echo "  качаю $1/$3"
    curl -fL -C - --retry 8 --retry-delay 3 -o "$1/$3" "$2" \
        || echo "  !! не вышло: $2"
}

hf_model() {  # hf_model <репозиторий> <папка> — веса и токенизатор для transformers
    local repo=$1 dir=models/$2
    for f in config.json model.safetensors tokenizer.json tokenizer_config.json \
             special_tokens_map.json; do
        get "$dir" "$HF/$repo/resolve/main/$f" "$f"
    done
    # у части моделей токенизатор словарём, а не json
    [ -s "$dir/tokenizer.json" ] || get "$dir" "$HF/$repo/resolve/main/vocab.txt" vocab.txt
    [ -s "$dir/tokenizer.json" ] || get "$dir" "$HF/$repo/resolve/main/sentencepiece.bpe.model" sentencepiece.bpe.model
}

if [ "$WHAT" = all ] || [ "$WHAT" = vectors ]; then
    echo "== энкодер для рантайма: e5-small в onnx int8 (129 МБ)"
    get models/e5-small "$HF/intfloat/multilingual-e5-small/resolve/main/onnx/tokenizer.json" tokenizer.json
    get models/e5-small "$HF/intfloat/multilingual-e5-small/resolve/main/onnx/model_qint8_avx512_vnni.onnx" model_qint8_avx512_vnni.onnx

    echo "== энкодеры для сравнения (1.2 ГБ)"
    hf_model cointegrated/rubert-tiny2        rubert-tiny2
    hf_model sergeyzh/rubert-tiny-turbo       rubert-tiny-turbo
    hf_model intfloat/multilingual-e5-small   e5-small-torch
    hf_model sergeyzh/LaBSE-ru-turbo          labse-ru-turbo
fi

if [ "$WHAT" = all ] || [ "$WHAT" = arbiter ]; then
    echo "== арбитр: llama.cpp и модели (1.8 ГБ)"
    B=b11062
    get models/llama \
        "https://github.com/ggml-org/llama.cpp/releases/download/$B/llama-$B-bin-ubuntu-cuda-12.8-x64.tar.gz" \
        llama-cuda.tar.gz
    get models/llama \
        "$HF/unsloth/Qwen3-0.6B-GGUF/resolve/main/Qwen3-0.6B-Q4_K_M.gguf" \
        Qwen3-0.6B-Q4_K_M.gguf
    get models/llama \
        "$HF/bartowski/Qwen_Qwen3-1.7B-GGUF/resolve/main/Qwen_Qwen3-1.7B-Q4_K_M.gguf" \
        Qwen3-1.7B-Q4_K_M.gguf
fi

echo
echo "итого в models:"
du -sh models/* 2>/dev/null
