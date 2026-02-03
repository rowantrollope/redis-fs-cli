BINARY   := redis-fs-cli
PKG      := github.com/rowantrollope/redis-fs-cli
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-X main.version=$(VERSION)"
GOFILES  := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: all build clean test lint run

all: build

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/redis-fs-cli

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

test:
	go test ./internal/...

test-integration:
	go test -v ./test/integration/...

test-e2e:
	go test -v ./test/e2e/...

test-all: test test-integration test-e2e

lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

fmt:
	gofmt -w $(GOFILES)

tidy:
	go mod tidy

# Cross-compilation
.PHONY: build-all
build-all:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64   ./cmd/redis-fs-cli
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64   ./cmd/redis-fs-cli
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-darwin-amd64  ./cmd/redis-fs-cli
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64  ./cmd/redis-fs-cli
