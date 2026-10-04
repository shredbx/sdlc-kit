#!/usr/bin/env bash
set -euo pipefail

PROJECT_PATH="consumers/clients/$INPUT_PROJECT_CODENAME/$INPUT_PROJECT_REPO_NAME/projects/$INPUT_PROJECT_PROJECT_NAME"

# Create project folder
mkdir -p "$PROJECT_PATH"

# Create project record
PROJECT_RECORD_DIR="processos-workspace/records/clients/project"
mkdir -p "$PROJECT_RECORD_DIR"
PROJECT_RECORD_FILE="$PROJECT_RECORD_DIR/$INPUT_PROJECT_CODENAME-$INPUT_PROJECT_REPO_NAME-$INPUT_PROJECT_PROJECT_NAME.yaml"

cat > "$PROJECT_RECORD_FILE" <<EOF
codename: $INPUT_PROJECT_CODENAME
repo_name: $INPUT_PROJECT_REPO_NAME
project_name: $INPUT_PROJECT_PROJECT_NAME
EOF

# Create basic project structure
cd "$PROJECT_PATH"

# Create package.json
cat > package.json <<EOF
{
  "name": "$INPUT_PROJECT_PROJECT_NAME",
  "version": "0.1.0",
  "private": true,
  "description": "Project $INPUT_PROJECT_PROJECT_NAME for client $INPUT_PROJECT_CODENAME"
}
EOF

# Create pnpm-workspace.yaml
cat > pnpm-workspace.yaml <<EOF
packages:
  - 'packages/*'
  - 'services/*'
EOF

# Create README.md
cat > README.md <<EOF
# $INPUT_PROJECT_PROJECT_NAME

Project for client $INPUT_PROJECT_CODENAME.
EOF

cd -
