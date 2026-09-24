#!/usr/bin/env bash
# assets/gather.py reads the 15 records and prints the process-os.command-reference this template
# renders from — this action's own equivalent of create-package's package_spec(). The runner renders.
set -euo pipefail

target="products/process-claude-plugin/skills/using-process-os"
cli_target="products/process-cli/src/process_cli"
spec="$(mktemp)"
trap 'rm -f "$spec"' EXIT

python3 "$ACTION_HOME/assets/gather.py" "$INPUT_PRODUCT" > "$spec"
process-cli render process-os.skill-reference "$spec" --into "$target"
process-cli render process-os.skill-reference "$spec" --into "$cli_target"

echo "$target/tools" > "$ACTION_OUTPUTS/path.yaml"
echo "$cli_target/tools" > "$ACTION_OUTPUTS/cli-path.yaml"
