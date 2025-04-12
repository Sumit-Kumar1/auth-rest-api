.PHONY: build test lint clean run docker-build docker-run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=auth-rest-api
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/main.go

test:
	$(GOTEST) -v -race -cover ./...

lint:
	golangci-lint run

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/main.go
	./$(BINARY_NAME)

docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -p 9001:9001 $(BINARY_NAME)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/main.go
	docker run --rm -it -v "$(GOPATH)":/go -w /go/src/github.com/yourusername/$(BINARY_NAME) golang:1.22 go build -o "$(BINARY_UNIX)" -v ./cmd/main.go

# Development
dev:
	air -c .air.toml 