FROM python:3.12-slim

WORKDIR /app
COPY VERSION /VERSION
COPY ticketgen/ /app/
RUN pip install --no-cache-dir .

ENV PYTHONUNBUFFERED=1
CMD ["python", "worker.py"]
