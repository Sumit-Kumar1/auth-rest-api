FROM golang:1.23.3 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main ./cmd/main.go

RUN ls -la /app

FROM alpine:3.28

RUN apk add --no-cache bash

COPY --from=builder /app/main .

EXPOSE 9001

CMD ["bash"]