FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install -tags "no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb" \
    github.com/pressly/goose/v3/cmd/goose@v3.26.0
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/internal/migrations /app/internal/migrations

EXPOSE 18080

CMD ["/app/api"]
