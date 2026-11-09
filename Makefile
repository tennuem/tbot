SHELL := /bin/sh

GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: build test lint

build:
	$(GO) build -o bin/tbot ./cmd/app

test:
	$(GO) test -race -timeout=60s ./...

lint:
	$(GOLANGCI_LINT) config verify
	$(GOLANGCI_LINT) fmt --diff
	$(GOLANGCI_LINT) run
