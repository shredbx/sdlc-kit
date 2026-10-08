# bos-demo — the bos product's own reference build. One entry point:
#   per app   make api-<target>   make web-<target>       (runs that app's own Makefile under apps/)
#   both      make dev | build | test | check | clean     (every app in APPS, one after the other; dev runs both at once)
#   datastore make start | stop                            (bos-postgres + bos-redis, the bos-dev bundle; see README.md)
#
# The framework is not fetched from anywhere: the apps link to the sdlc-kit checkout by local directory
# (go.work / replace for the API, the pnpm workspace for the web). `make links` checks that those directories are there.

.DEFAULT_GOAL := help
SHELL := /bin/sh

APPS     := api web
SDLC_KIT ?= ../../..
GO       ?= go
# the settings both apps read are in bos.yaml; the `bos` command, built from the framework's own module, writes them into ENV_DIR
BOS_SRC  := $(SDLC_KIT)/platform/go/frameworks/bos-go
BOS      := .bos/bin/bos
ENV_DIR  := .bos/env

.PHONY: help bos env links dev build test check clean start stop api-% web-%

help: ## this list
	@echo "Product targets:"
	@grep -h -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'
	@echo "Per app:"
	@echo "  api-<target>     run <target> from apps/api/Makefile, for example: make api-dev"
	@echo "  web-<target>     run <target> from apps/web/Makefile, for example: make web-build"

# ---- the datastore bundle (bos-postgres + bos-redis) -------------------------------------------------------------

start: ## start bos-postgres + bos-redis (the bos-dev bundle)
	docker compose -p bos-dev -f deploy/docker-compose.yml up -d

stop: ## stop bos-postgres + bos-redis
	docker compose -p bos-dev -f deploy/docker-compose.yml down

# ---- one place for the settings both apps read; the files land in .bos/ (git-ignored) ----------------------------

bos:
	@mkdir -p $(dir $(BOS))
	@GOWORK=off $(GO) -C $(BOS_SRC) build -o $(CURDIR)/$(BOS) ./cmd/bos

env: bos ## write .bos/env/api.env and web.env from bos.yaml (keeps an existing secret)
	@$(BOS) env -f bos.yaml -o $(ENV_DIR)

links: ## check that the local sdlc-kit directories the apps link to exist
	@ok=0; for d in platform/go/frameworks/bos-go platform/svelte/frameworks/bos-svelte; do \
	  if [ -d "$(SDLC_KIT)/$$d" ]; then echo "  ok       $$d"; else echo "  MISSING  $$d"; ok=1; fi; \
	done; exit $$ok

# ---- per app ----------------------------------------------------------------------------------------------------

api-%:
	@$(MAKE) --no-print-directory -C apps/api $*

web-%:
	@$(MAKE) --no-print-directory -C apps/web $*

# ---- every app --------------------------------------------------------------------------------------------------

dev: env ## run the API and the web together (Ctrl-C stops both)
	@$(MAKE) -j2 api-dev web-dev

build: ## build every app
	@for a in $(APPS); do $(MAKE) --no-print-directory -C apps/$$a build || exit 1; done

test: ## run every app's tests
	@for a in $(APPS); do $(MAKE) --no-print-directory -C apps/$$a test || exit 1; done

check: ## the framework's own checks, per app (the Go half and the web half)
	@for a in $(APPS); do $(MAKE) --no-print-directory -C apps/$$a check || exit 1; done

clean: ## remove every app's build output and the generated .bos/
	@for a in $(APPS); do $(MAKE) --no-print-directory -C apps/$$a clean; done
	rm -rf .bos
