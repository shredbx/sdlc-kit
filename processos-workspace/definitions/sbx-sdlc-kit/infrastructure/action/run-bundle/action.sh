#!/usr/bin/env bash
set -euo pipefail

root="$(git -C "$ACTION_HOME" rev-parse --show-toplevel)"
target="$root/projects/services/$INPUT_BUNDLE_NAME"

cd "$target"
docker compose -p "$INPUT_BUNDLE_NAME" up -d
