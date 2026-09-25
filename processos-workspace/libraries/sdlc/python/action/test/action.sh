#!/usr/bin/env bash
# The project's tests, in its own environment.
set -euo pipefail
cd "$INPUT_PROJECT_PATH"

uv run --quiet pytest -q
