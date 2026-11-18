BINARY_NAME=redis-top
VERSION ?= 0.1.0
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: all build test lint clean run-demo

all: test build

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/redis-top

test:
	go test -v -race -coverprofile=coverage.out ./...

run-demo:
	go run ./cmd/redis-top --demo

clean:
	rm -rf bin/ coverage.out dist/
