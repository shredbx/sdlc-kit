# Shared by action.sh. Sourced, not run.

# sbx-sdlc-kit.infrastructure.bundle-compose's own input, service-bundle-spec: the bundle as it
# came in, plus every member service's full record, resolved by id.
bundle_spec() {
  echo "bundle:"
  sed 's/^/  /' "$ACTION_INPUTS/bundle.yaml"
  echo "services:"
  local id
  for id in $(yq '.services[]' "$ACTION_INPUTS/bundle.yaml"); do
    process-cli show record "sbx-sdlc-kit/infrastructure/service/$id.yaml" \
      | awk 'NR==1 {print "  - " $0; next} {print "    " $0}'
  done
}

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
