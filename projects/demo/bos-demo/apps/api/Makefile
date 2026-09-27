# The API's own Makefile. The product Makefile calls it as `make api-<target>`; it also works on its own
# (`make -C apps/api dev`). It reads the settings the product Makefile writes with `make env`; without that file it uses the process
# environment as it is.

.DEFAULT_GOAL := help
SHELL := /bin/sh

GO       ?= go
# the sdlc-kit root: this app is five levels below it
SDLC_KIT ?= ../../../../..
BIN      := bin/api
ENV_FILE := ../../.bos/env/api.env
# runs the following command with the shared settings exported, when the file exists
WITH_ENV := set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a;
# the framework's checker is a package of the linked framework module, so it needs no separate install
BOS      := github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/cmd/bos

.PHONY: help dev run build build-linux test check clean

help: ## this list
	@grep -h -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

dev: ## run the API from source (stop with Ctrl-C)
	@$(WITH_ENV) $(GO) run .

run: build ## run the built binary
	@$(WITH_ENV) ./$(BIN)

build: ## build bin/api
	$(GO) build -o $(BIN) .

build-linux: ## a static linux binary for the bundle's image: make build-linux GOARCH=arm64 OUT=/path/to/api
	@test -n "$(GOARCH)" -a -n "$(OUT)" || { echo "usage: make build-linux GOARCH=<the docker daemon's architecture> OUT=<file>"; exit 1; }
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) $(GO) build -trimpath -ldflags="-s -w" -o $(OUT) .

test: ## unit tests of this app
	$(GO) test ./...

check: ## go vet (the framework's own `bos check` joins when the command has it)
	$(GO) vet ./...
	@if $(GO) run $(BOS) 2>&1 | grep -q '^  bos check'; then $(GO) run $(BOS) check; else echo "bos check: not available yet (the bos command has no check yet)"; fi

clean: ## remove bin/
	rm -rf bin
