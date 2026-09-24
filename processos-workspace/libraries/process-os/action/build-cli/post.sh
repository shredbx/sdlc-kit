#!/usr/bin/env bash
# Seed the config and the std library, only where the target holds neither yet, then prove the
# build works: run the executable's own --version and check against what was just seeded. The
# outer process-cli's own PROCESS_CLI_CONFIG (this repository's) is unset for that call, or check
# would silently validate this repository's config instead of the one just seeded.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

seed_config
seed_std

"$target/process-cli" --version > /dev/null
(cd "$target" && env -u PROCESS_CLI_CONFIG ./process-cli check)
