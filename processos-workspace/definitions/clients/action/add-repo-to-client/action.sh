#!/usr/bin/env bash
set -euo pipefail

REPO_PATH="consumers/clients/$INPUT_CLIENT_REPO_CODENAME/$INPUT_CLIENT_REPO_REPO_NAME"
REPO_URL="$INPUT_CLIENT_REPO_REPO_URL"

# Create client-repo record
REPO_RECORD_DIR="processos-workspace/records/clients/repo"
mkdir -p "$REPO_RECORD_DIR"
REPO_RECORD_FILE="$REPO_RECORD_DIR/$INPUT_CLIENT_REPO_CODENAME-$INPUT_CLIENT_REPO_REPO_NAME.yaml"

cat > "$REPO_RECORD_FILE" <<EOF
codename: $INPUT_CLIENT_REPO_CODENAME
repo_name: $INPUT_CLIENT_REPO_REPO_NAME
repo_url: $INPUT_CLIENT_REPO_REPO_URL
EOF

# Add git submodule entry to .gitmodules
git submodule add "$REPO_URL" "$REPO_PATH"

# Initialize and update the submodule
git submodule update --init "$REPO_PATH"

# Create basic repo structure
cd "$REPO_PATH"

# Create processos.yaml if it doesn't exist
if [ ! -f processos.yaml ]; then
  cat > processos.yaml <<EOF
# The config of this workspace: where its definitions are, and where runs work. process-cli finds it by
# looking upward from the current folder. Paths are relative to this file.
version: 0.1.0
definitions: processos-workspace/definitions
runtime:
  root: processos-workspace
  records: records
  output: output
  runs: runs
libraries:
  - {name: sbx-sdlc-kit, path: ../../../../processos-workspace/definitions/sbx-sdlc-kit, access: readonly}
EOF
fi

# Create processos-workspace structure
mkdir -p processos-workspace/definitions
mkdir -p processos-workspace/records
touch processos-workspace/definitions/.gitkeep

# Create projects folder
mkdir -p projects

# Create CLAUDE.md if it doesn't exist
if [ ! -f CLAUDE.md ]; then
  cat > CLAUDE.md <<EOF
# CLAUDE.md — $INPUT_CLIENT_REPO_REPO_NAME (client repo)

A process-os workspace for client integration, built on top of sdlc-kit's
shared scope (reused read-only via \`processos.yaml\`'s \`libraries:\` entry — see that file).

## Layout

- \`processos-workspace/\` — this workspace's own process-os definitions, records, and runtime state.
- \`projects/\` — the application(s) built for this client.
EOF
fi

# Create README.md if it doesn't exist
if [ ! -f README.md ]; then
  cat > README.md <<EOF
# Client Workspace

A process-os workspace for one client's integration, built on top of sdlc-kit's
shared scope (reused read-only via \`processos.yaml\`'s \`libraries:\` entry — see that file).

## Layout

- \`processos-workspace/\` — this workspace's own process-os definitions, records, and runtime state.
- \`projects/\` — the application(s) built for this client.
EOF
fi

cd -
