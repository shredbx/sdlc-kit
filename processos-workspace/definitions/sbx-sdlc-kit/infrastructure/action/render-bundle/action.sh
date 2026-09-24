#!/usr/bin/env bash
# Render, then copy out: render's own --into is confined to processos-workspace/output/, so the
# compose files land in scratch first, then get copied into the real target.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

spec="$(mktemp)"
trap 'rm -f "$spec"' EXIT
bundle_spec > "$spec"

scratch="render-bundle-$INPUT_BUNDLE_NAME"
rm -rf "$scratch"
process-cli render sbx-sdlc-kit.infrastructure.bundle-compose "$spec" --into "$scratch"

root="$(git -C "$ACTION_HOME" rev-parse --show-toplevel)"
target="$root/projects/services/$INPUT_BUNDLE_NAME"
mkdir -p "$target"
cp -R "$scratch"/. "$target"/
