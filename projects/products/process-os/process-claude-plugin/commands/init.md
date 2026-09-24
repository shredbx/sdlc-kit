---
description: Bootstrap process-cli in this repository — install it if missing, start the workspace if it hasn't been, record that this repo uses it, then ask what to build.
argument-hint: "<optional: the scope name to start the workspace with>"
---

# /init

$ARGUMENTS

This is a bootstrap request — `process-cli` may not even be installed yet, so this is not routed
into one of the ordinary command groups. Use the skill `using-process-os`: read its own "Install
process-cli first, if it is not already here" section and carry it out in order — install if
missing, `process-cli init` (`$ARGUMENTS` as the scope name if one was given, otherwise ask),
`process-cli check`, then this repository's own `CLAUDE.md` brought up to date — saying which of
those actually ran and which were already done, so a repeat call is cheap and legible, not a
surprise. Only once all of it is settled, ask what to actually build: an empty scope has nothing
yet for any other command or tool to act on, so this is never the end of the conversation by
itself.
