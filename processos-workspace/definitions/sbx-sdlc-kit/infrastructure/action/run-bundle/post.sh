#!/usr/bin/env bash
# Verification: every host port the bundle publishes actually accepts a connection. Generic —
# works for any bundle — since it reads the ports docker compose resolved, not what kind of
# service publishes them.
set -euo pipefail
source "$ACTION_HOME/assets/lib.sh"

dir="$(bundle_dir)"
for host_port in $(published_ports "$dir"); do
  ok=0
  for _ in $(seq 1 15); do
    if nc -z localhost "$host_port" 2>/dev/null; then
      ok=1
      break
    fi
    sleep 1
  done
  if [[ "$ok" -ne 1 ]]; then
    echo "nothing listening on localhost:$host_port after 15s" >&2
    exit 1
  fi
done
