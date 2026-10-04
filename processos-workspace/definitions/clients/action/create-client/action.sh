#!/usr/bin/env bash
set -euo pipefail

# Create client record
RECORD_DIR="processos-workspace/records/clients/client"
mkdir -p "$RECORD_DIR"
RECORD_FILE="$RECORD_DIR/$INPUT_CLIENT_CODENAME.yaml"

cat > "$RECORD_FILE" <<EOF
codename: $INPUT_CLIENT_CODENAME
repo_path: $INPUT_CLIENT_REPO_PATH
repo_url: $INPUT_CLIENT_REPO_URL
status: $INPUT_CLIENT_STATUS
EOF

# Create client folder
CLIENT_DIR="consumers/clients/$INPUT_CLIENT_CODENAME"
mkdir -p "$CLIENT_DIR"
