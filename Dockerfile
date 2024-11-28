FROM golang:1.23.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/main.go

FROM alpine:3.19

RUN adduser -D appuser

WORKDIR /app

COPY .env .

COPY --from=builder /app/main /app/main

RUN chown appuser:appuser /app/main && \
    chmod +x /app/main

USER appuser

EXPOSE 9001

CMD ["./main"]