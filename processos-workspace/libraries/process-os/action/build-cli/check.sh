#!/usr/bin/env bash
# continue (0)  no stamp yet, or the sources have changed since the one there.
# skip (77)     a build is there and its stamp's digest still matches the sources.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

if [[ -f "$stamp" ]] && [[ "$(stamp_digest)" == "$(source_digest)" ]]; then
  echo "$target" > "$ACTION_OUTPUTS/path.yaml"
  exit 77
fi
exit 0
