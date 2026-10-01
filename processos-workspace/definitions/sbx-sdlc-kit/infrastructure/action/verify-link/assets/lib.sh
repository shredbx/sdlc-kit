# Shared by action.sh. Sourced, not run.

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

# The host port a service publishes, as docker compose resolves it (a port written as
# ${VAR}:8080 is read through the bundle's .env). Fails, saying so, when there is none.
published_port() {
  local dir="$1" service="$2" port
  port="$(docker compose -p "$INPUT_BUNDLE_NAME" -f "$dir/docker-compose.yml" --project-directory "$dir" config --format json \
    | jq -r --arg s "$service" '.services[$s].ports[0].published // empty')"
  if [[ -z "$port" ]]; then
    echo "the bundle has no service '$service' that publishes a port" >&2
    return 1
  fi
  echo "$port"
}
