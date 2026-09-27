# Shared by action.sh, check.sh and post.sh. Sourced, not run.

# The folder the bundle's compose files live in. A bundle that says `into` puts them there,
# relative to the repository the run works in (the one holding PROCESS_OUTPUT); otherwise they
# are in projects/services/<name> of the repository the actions live in. Kept identical in every
# action of this capability that needs the folder (render-bundle, run-bundle, verify-link).
bundle_dir() {
  local root
  if [[ -n "${INPUT_BUNDLE_INTO:-}" ]]; then
    case "/$INPUT_BUNDLE_INTO/" in
      /../* | */../* | //*)
        echo "bundle into: '$INPUT_BUNDLE_INTO' must be a folder inside the repository (relative, no ..)" >&2
        return 1
        ;;
    esac
    root="$(git -C "$PROCESS_OUTPUT" rev-parse --show-toplevel)"
    echo "$root/$INPUT_BUNDLE_INTO"
  else
    root="$(git -C "$ACTION_HOME" rev-parse --show-toplevel)"
    echo "$root/projects/services/$INPUT_BUNDLE_NAME"
  fi
}

# Every host port the bundle publishes, one per line, as docker compose resolves them (so a
# port written as ${VAR}:8080 is read through the bundle's .env). Reads the file only; the
# docker daemon is not needed.
published_ports() {
  local dir="$1"
  docker compose -p "$INPUT_BUNDLE_NAME" -f "$dir/docker-compose.yml" --project-directory "$dir" config --format json \
    | jq -r '.services[].ports[]? | .published // empty'
}
