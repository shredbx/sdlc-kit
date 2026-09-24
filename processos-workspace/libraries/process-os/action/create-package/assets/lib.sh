# Shared by check.sh, action.sh and post.sh. Sourced, not run.

# Where the package goes, and the names uv and Python know it by, from its kind (a kit package when
# it gives none). The schema cannot say "a module for a kit only", so it is said here.
kind="${INPUT_PACKAGE_KIND:-kit}"
name="$INPUT_PACKAGE_NAME"
module="${INPUT_PACKAGE_MODULE:-}"

case "$kind" in
  kit)
    [[ -n "$module" ]] || { echo "a kit package needs a module, and $name gives none" >&2; exit 1; }
    target="packages/$module/$name"
    dist="$module-$name"
    source="${module//-/_}/${name//-/_}"
    ;;
  framework | product)
    [[ -z "$module" ]] || { echo "a $kind has no module, and $name gives $module" >&2; exit 1; }
    target="${kind}s/$name"
    dist="$name"
    source="${name//-/_}"
    ;;
esac

# The template's input, process-os.package-spec: the package as it came in, indented under one
# key, so no field of it is written again by hand, and the names above.
package_spec() {
  echo "package:"
  sed 's/^/  /' "$ACTION_INPUTS/package.yaml"
  echo "kind: $kind"
  echo "dist: $dist"
  echo "source: $source"
}
