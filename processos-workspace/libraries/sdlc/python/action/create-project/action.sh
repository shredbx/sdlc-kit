#!/usr/bin/env bash
# Render the project into the build target: the common files, then the entry point of the
# application's framework. The runner renders; this script only says what and where.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

spec="$(mktemp)"
trap 'rm -f "$spec"' EXIT
project_spec > "$spec"

process-cli render sdlc.python.application "$spec" --into "$INPUT_BUILD_TARGET"
process-cli render "sdlc.python.entry-$INPUT_PLATFORM_FRAMEWORK" "$spec" --into "$INPUT_BUILD_TARGET"

project > "$ACTION_OUTPUTS/project.yaml"
