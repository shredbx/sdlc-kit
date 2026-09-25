# Shared by check.sh, action.sh and post.sh. Sourced, not run.

# Every plugin goes in products/<name>/, named by its own plugin.json — no kind, no module: a
# plugin is laid out one way only.
name="$INPUT_PLUGIN_NAME"
target="products/$name"
