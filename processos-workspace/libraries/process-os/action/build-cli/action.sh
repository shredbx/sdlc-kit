#!/usr/bin/env bash
# Build the wheels, make the zipapp with shiv (form B, decision 59), and write the stamp. Needs
# uv, Python 3.13, and a network the first time shiv or a compiled dependency is not already
# cached (analysis.md §5.6, §10).
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

mkdir -p "$target"
uv build --all-packages --wheel --quiet
uvx shiv --quiet -p '/usr/bin/env python3.13' -e process_cli.main:main -o "$target/process-cli" dist/*.whl

cat > "$stamp" <<EOF
version: $(cli_version)
commit: $(git rev-parse HEAD)
date: $(date -u +%Y-%m-%d)
digest: $(source_digest)
EOF

echo "$target" > "$ACTION_OUTPUTS/path.yaml"
