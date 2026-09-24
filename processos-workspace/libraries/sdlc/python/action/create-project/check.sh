#!/usr/bin/env bash
# continue (0)  nothing at the target yet.
# skip (77)     the project is there and still conforms to its templates. Say where it is:
#               the next node needs it all the same.
# stop (other)  something is there that does not conform. Never write over it.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

[[ -e "$INPUT_BUILD_TARGET" ]] || exit 0

spec="$(mktemp)"
trap 'rm -f "$spec"' EXIT
project_spec > "$spec"

if process-cli conform sdlc.python.application "$spec" "$INPUT_BUILD_TARGET" \
  && process-cli conform "sdlc.python.entry-$INPUT_PLATFORM_FRAMEWORK" "$spec" "$INPUT_BUILD_TARGET"; then
  project > "$ACTION_OUTPUTS/project.yaml"
  exit 77
fi

echo "$INPUT_BUILD_TARGET is there and does not conform to its templates" >&2
exit 1
