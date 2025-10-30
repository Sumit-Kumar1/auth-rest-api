FROM golang:1.25 AS builder

WORKDIR /auth-rest-api

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -extldflags '-static'" -o main .

FROM alpine:3.19

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /auth-rest-api

COPY --from=builder /auth-rest-api/main /auth-rest-api/main

RUN chown appuser:appgroup /auth-rest-api/main && \
    chmod +x /auth-rest-api/main

USER appuser

ARG PORT=8000
EXPOSE ${PORT}

CMD ["./main"]