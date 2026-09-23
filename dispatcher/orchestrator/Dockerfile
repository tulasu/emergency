FROM golang:1.22-alpine
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/orchestrator .
EXPOSE 8080
CMD ["/out/orchestrator"]
