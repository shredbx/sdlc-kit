#!/usr/bin/env bash
# The environment, on the application's runtime, then the package of its framework.
set -euo pipefail
cd "$INPUT_PROJECT_PATH"

uv sync --quiet --python "$INPUT_PLATFORM_RUNTIME"

case "$INPUT_PLATFORM_FRAMEWORK" in
  none)  ;;
  typer) uv add --quiet typer ;;
esac
