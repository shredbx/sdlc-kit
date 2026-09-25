#!/usr/bin/env bash
# The platform profile of the application: what it runs on and what builds it. A language the
# platform does not run stops the process here, before anything is written.
set -euo pipefail

case "$INPUT_APPLICATION_PLATFORM" in
  python) languages="python"                tool="uv"  ;;
  node)   languages="typescript javascript" tool="npm" ;;
  *)      echo "no profile for platform $INPUT_APPLICATION_PLATFORM" >&2; exit 1 ;;
esac

if [[ " $languages " != *" $INPUT_APPLICATION_LANGUAGE "* ]]; then
  echo "platform $INPUT_APPLICATION_PLATFORM does not run $INPUT_APPLICATION_LANGUAGE" >&2
  exit 1
fi

cat > "$ACTION_OUTPUTS/platform.yaml" <<EOF
name: $INPUT_APPLICATION_PLATFORM
language: $INPUT_APPLICATION_LANGUAGE
framework: $INPUT_APPLICATION_FRAMEWORK
runtime: "$INPUT_BUILD_RUNTIME"
tool: $tool
EOF
