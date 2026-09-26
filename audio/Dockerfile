FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY audio/go.mod audio/go.sum ./
RUN go mod download
COPY audio/ ./
RUN go build -o /app/audio ./cmd/audio

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /app/audio /app/audio
# Unprivileged runtime: listens on $AUDIO_HTTP_ADDR (:8002).
USER nobody
HEALTHCHECK --interval=30s --timeout=3s CMD curl -fsS http://127.0.0.1:8002/health >/dev/null || exit 1
EXPOSE 8002
CMD ["/app/audio"]
