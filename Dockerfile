FROM golang:1.25-alpine AS builder
WORKDIR /auth-rest-api

COPY internal/ ./internal
COPY go.mod go.sum main.go ./

RUN go mod download && \
  CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -extldflags '-static'" -o main .

FROM alpine:3.22 AS production

RUN apk --no-cache add ca-certificates && \
  addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /auth-rest-api

COPY --from=builder /auth-rest-api/main /auth-rest-api/main

RUN chown appuser:appgroup /auth-rest-api/main && \
  chmod +x /auth-rest-api/main

USER appuser

ARG HTTP_PORT=9001
EXPOSE ${HTTP_PORT}

CMD ["./main"]
