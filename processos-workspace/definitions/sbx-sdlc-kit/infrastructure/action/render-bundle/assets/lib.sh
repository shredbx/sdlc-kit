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
