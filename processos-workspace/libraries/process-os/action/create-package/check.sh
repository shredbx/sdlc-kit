#!/usr/bin/env bash
# continue (0)  nothing at the target yet.
# skip (77)     the package is there and still conforms to the template. Say where it is.
# stop (other)  something is there that does not conform. Never write over it.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

[[ -e "$target" ]] || exit 0

spec="$(mktemp)"
trap 'rm -f "$spec"' EXIT
package_spec > "$spec"

if process-cli conform process-os.package "$spec" "$target"; then
  echo "$target" > "$ACTION_OUTPUTS/path.yaml"
  exit 77
fi

echo "$target is there and does not conform to process-os.package" >&2
exit 1
