.DEFAULT_GOAL := help
SHELL := /bin/sh

PNPM ?= pnpm

.PHONY: help dev

help: ## this list
	@grep -h -E '^[a-zA-Z0-9_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'

dev: ## run the Hot Potato demo (stop with Ctrl-C)
	@$(PNPM) dev
