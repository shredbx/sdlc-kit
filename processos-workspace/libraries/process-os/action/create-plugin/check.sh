#!/usr/bin/env bash
# continue (0)  nothing at the target yet.
# skip (77)     the plugin is there and still conforms to the template. Say where it is.
# stop (other)  something is there that does not conform. Never write over it.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

[[ -e "$target" ]] || exit 0

if process-cli conform process-os.claude-plugin "$ACTION_INPUTS/plugin.yaml" "$target"; then
  echo "$target" > "$ACTION_OUTPUTS/path.yaml"
  exit 77
fi

echo "$target is there and does not conform to process-os.claude-plugin" >&2
exit 1
