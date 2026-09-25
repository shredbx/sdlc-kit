#!/usr/bin/env bash
# assets/build_docs.py reads and writes; this script only says which product.
set -euo pipefail

python3 "$ACTION_HOME/assets/build_docs.py" "$INPUT_PRODUCT"

echo "processos-workspace/records/product/$INPUT_PRODUCT/readme.yaml" > "$ACTION_OUTPUTS/path.yaml"
