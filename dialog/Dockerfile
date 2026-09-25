FROM python:3.11-slim

WORKDIR /app
COPY dialog/ ./dialog/
# Runtime deps already used by the code (yaml/numpy/pymorphy3/razdel).
# Prod ML (torch/onnx/cuda/llama-server) stays out of this CPU image.
RUN pip install --no-cache-dir pyyaml numpy pymorphy3 razdel
ENV PYTHONUNBUFFERED=1 \
    DIALOG_WAV_DIR=/tmp/dialog-wav
# Unprivileged runtime: binds 8000/9001 only, wavs stay on a tmp volume.
USER nobody
HEALTHCHECK --interval=30s --timeout=3s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/health')"
EXPOSE 8000 9001
CMD ["python3", "-m", "dialog.serve"]
