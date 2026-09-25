# Shared by check.sh and action.sh. Sourced, not run.

target="$INPUT_TARGET"
stamp="$target/process-cli.build"

# process-cli's own version, from the package the wheels are built from (decision 61: it stays
# 0.1.0 for now; the stamp is what tells two builds apart).
cli_version() {
  sed -n 's/^version = "\(.*\)"/\1/p' products/process-cli/pyproject.toml
}

# One hash of every tracked file under the folders the build reads from: packages/, frameworks/
# and products/ — exactly what `uv build --all-packages` builds. Each file is read fresh off disk
# (git hash-object, not the index), so an uncommitted edit is seen; a file not yet `git add`ed is
# not.
source_digest() {
  git ls-files -- packages frameworks products | LC_ALL=C sort | xargs git hash-object | shasum -a 256 | cut -d' ' -f1
}

# The digest the stamp already there recorded, or nothing when there is no stamp.
stamp_digest() {
  sed -n 's/^digest: //p' "$stamp"
}

# Writes processos.yaml, and makes processos-workspace/definitions/ (Config.load needs both it and
# its parent, the runtime root, to exist, empty or not), only where the target holds no config yet —
# a rebuild must not overwrite one someone has since edited.
seed_config() {
  mkdir -p "$target/processos-workspace/definitions"
  [[ -f "$target/processos.yaml" ]] && return 0
  cat > "$target/processos.yaml" <<'CONFIG'
# The config of this placed copy of process-cli. process-cli finds it beside itself when nothing
# else claims one (or above where you run it, which wins if there is one). Paths are relative to
# this file.
definitions: processos-workspace/definitions   # a folder of scopes; every folder in it is one
runtime:
  root: processos-workspace
  records: records
libraries:
  - name: std                     # the shapes every scope shares, such as a path or a version
    path: libraries/std
CONFIG
  cat > "$target/processos-workspace/.gitignore" <<'IGNORE'
# What runs write. The records given to runs, in records/, is kept.
output/
runs/
IGNORE
}

# Copies processos-workspace/definitions/std into libraries/std, only where the target holds none
# yet — same reasoning as seed_config.
seed_std() {
  [[ -d "$target/libraries/std" ]] && return 0
  mkdir -p "$target/libraries"
  cp -R processos-workspace/definitions/std "$target/libraries/std"
}
