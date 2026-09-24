#!/usr/bin/env bash
# Verification: every member service's own host port actually accepts a connection. Generic —
# works for any future bundle, not just this one — since it only depends on each service's own
# ports field, not on knowing what kind of service it is.
set -euo pipefail

for id in $(yq '.services[]' "$ACTION_INPUTS/bundle.yaml"); do
  ports="$(process-cli show record "sbx-sdlc-kit/infrastructure/service/$id.yaml" | yq '.ports[]' 2>/dev/null || true)"
  for p in $ports; do
    host_port="${p%%:*}"
    ok=0
    for _ in $(seq 1 15); do
      if nc -z localhost "$host_port" 2>/dev/null; then
        ok=1
        break
      fi
      sleep 1
    done
    if [[ "$ok" -ne 1 ]]; then
      echo "$id: nothing listening on localhost:$host_port after 15s" >&2
      exit 1
    fi
  done
done
