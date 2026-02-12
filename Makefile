BINARY_NAME=gotermsql
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build build-full install test lint clean run

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/gotermsql

build-full:
	CGO_ENABLED=1 go build -tags duckdb $(LDFLAGS) -o bin/$(BINARY_NAME)-full ./cmd/gotermsql

install:
	CGO_ENABLED=0 go install $(LDFLAGS) ./cmd/gotermsql

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .
	goimports -w .

vet:
	go vet ./...

clean:
	rm -rf bin/
	go clean

run:
	go run $(LDFLAGS) ./cmd/gotermsql $(ARGS)

tidy:
	go mod tidy

deps:
	go mod download

# Cross-compilation targets
build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/gotermsql

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/gotermsql

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/gotermsql

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/gotermsql

build-windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/gotermsql

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
