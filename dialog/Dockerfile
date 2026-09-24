FROM python:3.12-slim

WORKDIR /app
COPY dialog/ ./dialog/
ENV PYTHONUNBUFFERED=1 \
    DIALOG_WAV_DIR=/tmp/dialog-wav
# Unprivileged runtime: binds 8000/9001 only, wavs stay on a tmp volume.
USER nobody
HEALTHCHECK --interval=30s --timeout=3s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8000/health')"
EXPOSE 8000 9001
CMD ["python3", "-m", "dialog.serve", "--port", "8000"]
