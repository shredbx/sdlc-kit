# Verification: the uv workspace takes the package in, and it imports. The workspace's members are
# globs over packages/<module>/, frameworks/ and products/, so nothing else needs to change.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

uv sync --quiet --all-packages --group dev
uv run --quiet --package "$dist" python -c "import ${source//\//.}"
