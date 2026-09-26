# GPU synth daemon: Silero v4_ru on torch cu130 (same base as dispatcher GPU).
# CUDA runtime comes from the torch wheels — no CUDA base image needed,
# driver is injected by the NVIDIA container toolkit (gpus: all in compose).
FROM python:3.12-slim
WORKDIR /app
# ffmpeg converts the bundled Russian neural fallback to telephony PCM.
# gcc + libc6-dev: torchaudio build deps on slim.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg gcc libc6-dev curl \
    && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir --index-url https://download.pytorch.org/whl/cu130 \
    "torch==2.14.0" "torchaudio==2.14.0"
RUN pip install --no-cache-dir "numpy>=2.0" "piper-tts==1.6.1"
RUN python -m piper.download_voices --download-dir /opt/piper ru_RU-denis-medium
COPY audio-synth/ ./
ENV PYTHONUNBUFFERED=1 \
    SYNTH_PORT=8003 \
    SYNTH_DEVICE=cuda \
    TORCH_HOME=/models/torch \
    HF_HOME=/models
# Root (like dialog prod): torch/HF caches write under /models volume.
HEALTHCHECK --interval=30s --timeout=3s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8003/health')"
EXPOSE 8003
CMD ["python3", "app.py"]
