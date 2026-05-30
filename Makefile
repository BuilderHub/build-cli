BINARY := builderhub
VERSION ?= dev
LDFLAGS := -s -w -X github.com/builderhub/build-cli/internal/cmd.version=$(VERSION)

.PHONY: build install test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

clean:
	rm -rf bin/
