#!/usr/bin/env bash
# Register the plugin: assets/register.py adds it to this repository's own root
# .claude-plugin/marketplace.json, and excludes its folder from the uv workspace (found on the way
# — a plugin has no pyproject.toml, and the workspace's own products/* member glob would otherwise
# break on it). Both are idempotent.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

python3 "$ACTION_HOME/assets/register.py" "$PROCESS_OUTPUT" "$name" "$target"
