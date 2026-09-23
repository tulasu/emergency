FROM python:3.12-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    espeak-ng ffmpeg \
    && rm -rf /var/lib/apt/lists/*
COPY pyproject.toml ./
# CPU-only: onnxruntime для моделей, без torch (~2ГБ).
RUN pip install --no-cache-dir \
    "pymorphy3==2.0.6" "razdel==0.5.0" "numpy>=2.0" "PyYAML>=6.0" "setuptools<81" \
    "onnxruntime>=1.20" "tokenizers>=0.20" \
    "vosk>=0.3.45" \
    "faster-whisper>=1.0"
COPY dispatcher ./dispatcher
COPY tools ./tools
EXPOSE 8000
CMD ["python3", "-m", "dispatcher.serve", "--port", "8000"]
