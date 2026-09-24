#!/usr/bin/env bash
# Render the package into packages/<module>/<name>/. The runner renders; this script only says
# what and where.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

spec="$(mktemp)"
trap 'rm -f "$spec"' EXIT
package_spec > "$spec"

process-cli render process-os.package "$spec" --into "$target"

echo "$target" > "$ACTION_OUTPUTS/path.yaml"
