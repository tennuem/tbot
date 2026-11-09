SHELL := /bin/sh

GO ?= go
GOLANGCI_LINT ?= golangci-lint
OPENSPEC ?= ./scripts/openspec
OPENSPEC_VERSION ?= 1.16.1
OPENSPEC_TOOLS ?= codex,opencode,oh-my-pi

.PHONY: build test lint openspec-install openspec-init openspec-validate

build:
	$(GO) build -o bin/tbot ./cmd/app

test:
	$(GO) test -race -timeout=60s ./...

lint:
	$(GOLANGCI_LINT) config verify
	$(GOLANGCI_LINT) fmt --diff
	$(GOLANGCI_LINT) run

openspec-install:
	mkdir -p bin
	curl -fsSL https://openspec.dev/cli/install.sh -o bin/openspec-install.sh
	env OPENSPEC_VERSION="$(OPENSPEC_VERSION)" OPENSPEC_INSTALL_DIR="$(CURDIR)/bin" sh bin/openspec-install.sh
	rm bin/openspec-install.sh

openspec-init:
	env DO_NOT_TRACK=1 OPENSPEC_NO_UPDATE_CHECK=1 "$(OPENSPEC)" init --tools "$(OPENSPEC_TOOLS)" --profile core --no-animation

openspec-validate:
	env DO_NOT_TRACK=1 OPENSPEC_NO_UPDATE_CHECK=1 "$(OPENSPEC)" validate --all --strict --no-interactive
