FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY traineebox/go.mod traineebox/go.sum ./
RUN go mod download
COPY traineebox/ ./
RUN go build -o /app/traineebox ./cmd/app
RUN go build -o /app/initializer ./cmd/initializer

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /app/traineebox /app/traineebox
COPY --from=build /app/initializer /app/initializer
# Unprivileged runtime: listens on $TRAINEEBOX_HTTP_ADDR (:8080).
USER nobody
# Any HTTP status (even 404 on /) proves the server is up; refused conn fails.
HEALTHCHECK --interval=30s --timeout=3s CMD curl -s -o /dev/null http://127.0.0.1:8080/ || exit 1
EXPOSE 8080
CMD ["/app/traineebox"]
