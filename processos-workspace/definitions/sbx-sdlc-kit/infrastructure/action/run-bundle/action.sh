#!/usr/bin/env bash
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

cd "$(bundle_dir)"
# --build builds the images of services that have a build and does nothing for the rest.
docker compose -p "$INPUT_BUNDLE_NAME" up -d --build
