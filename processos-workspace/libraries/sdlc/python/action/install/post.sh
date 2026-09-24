#!/usr/bin/env bash
# Verification: the environment imports the application's module.
set -euo pipefail
cd "$INPUT_PROJECT_PATH"

uv run --quiet python -c "import $INPUT_PROJECT_MODULE"
