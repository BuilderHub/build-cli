BINARY := builderhub
VERSION ?= dev
LDFLAGS := -s -w -X github.com/builderhub/build-cli/internal/cmd.version=$(VERSION)

.PHONY: build install test test-coverage lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/builderhub

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/builderhub

test:
	go test ./...

test-coverage:
	go test -race -coverprofile=coverage.out ./internal/client ./internal/config ./internal/output
	go tool cover -func=coverage.out

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

clean:
	rm -rf bin/
