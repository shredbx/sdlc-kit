#!/usr/bin/env bash
# Render the plugin into products/<name>/. The runner renders; this script only says what and
# where.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

process-cli render process-os.claude-plugin "$ACTION_INPUTS/plugin.yaml" --into "$target"

echo "$target" > "$ACTION_OUTPUTS/path.yaml"
