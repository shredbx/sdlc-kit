#!/usr/bin/env bash
# Build the wheel and the source archive, then list the two of this version.
set -euo pipefail
cd "$INPUT_PROJECT_PATH"

uv build --quiet

{
  echo "path: $INPUT_PROJECT_PATH"
  echo "files:"
  for file in dist/*-"$INPUT_APPLICATION_VERSION"[.-]*; do
    echo "  - $file"
  done
} > "$ACTION_OUTPUTS/artifact.yaml"
