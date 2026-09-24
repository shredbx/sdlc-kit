# Shared by action.sh and check.sh. Sourced, not run.

module="${INPUT_APPLICATION_NAME//-/_}"

# The templates' input, sdlc.python.project-spec: the application as it came in, indented under
# one key, so no field of it is written again by hand.
project_spec() {
  echo "application:"
  sed 's/^/  /' "$ACTION_INPUTS/application.yaml"
  echo "module: $module"
  echo "runtime: \"$INPUT_PLATFORM_RUNTIME\""
}

# This action's output, sdlc.python.project.
project() {
  echo "path: $INPUT_BUILD_TARGET"
  echo "module: $module"
}
