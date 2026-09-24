FROM python:3.12-slim

WORKDIR /app
COPY VERSION /VERSION
COPY ticketgen/ /app/
RUN pip install --no-cache-dir .

ENV PYTHONUNBUFFERED=1
# Unprivileged runtime: nothing here binds <1024 or writes outside /tmp.
USER nobody
HEALTHCHECK --interval=30s --timeout=3s CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8001/health')"
# Pure function over HTTP: traineebox worker drives jobs via POST /draft.
# Catalog mounted at /catalog (or $TICKETGEN_CATALOG_PATH); --catalog missing
# used to crash-loop the image (spec R).
EXPOSE 8001
CMD ["python", "server.py", "--catalog", "/catalog"]
