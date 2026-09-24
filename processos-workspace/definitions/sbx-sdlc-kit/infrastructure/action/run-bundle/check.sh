#!/usr/bin/env bash
# stop (other)  render-bundle hasn't run yet, or the real secrets file isn't set up.
# continue (0)  otherwise.
set -euo pipefail

root="$(git -C "$ACTION_HOME" rev-parse --show-toplevel)"
target="$root/projects/services/$INPUT_BUNDLE_NAME"

if [[ ! -f "$target/docker-compose.yml" ]]; then
  echo "$target/docker-compose.yml is missing — run render-bundle first" >&2
  exit 1
fi

if [[ ! -f "$target/.env" ]]; then
  echo "$target/.env is missing — copy $target/.env.example to $target/.env and fill in real values" >&2
  exit 1
fi

exit 0
