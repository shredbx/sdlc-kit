# The web app's own Makefile. The product Makefile calls it as `make web-<target>`; it also works on its own
# (`make -C apps/web dev`). It reads the settings the product Makefile writes with `make env`.

.DEFAULT_GOAL := help
SHELL := /bin/sh

PNPM     ?= pnpm
ENV_FILE := ../../.bos/env/web.env
WITH_ENV := set -a; [ -f $(ENV_FILE) ] && . $(ENV_FILE); set +a;

.PHONY: help dev build run test check clean

help: ## this list
	@grep -h -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

dev: ## run the dev server (stop with Ctrl-C)
	@$(WITH_ENV) $(PNPM) dev --port $${PORT:-4010}

build: ## production build into build/
	@$(WITH_ENV) $(PNPM) build

run: build ## run the production build
	@$(WITH_ENV) node build

test: ## unit tests of this app (none yet: tests come after the core is validated)
	@echo "apps/web has no unit tests yet"

check: ## type check (the framework's own `bos-web check` joins when the command exists)
	$(PNPM) check
	@if $(PNPM) exec bos-web --version >/dev/null 2>&1; then $(PNPM) exec bos-web check; else echo "bos-web check: not available yet"; fi

clean: ## remove build output
	rm -rf build .svelte-kit
