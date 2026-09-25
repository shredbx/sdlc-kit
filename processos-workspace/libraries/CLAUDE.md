# processos-workspace/libraries — verbatim upstream copies

Each folder here is a byte-for-byte copy of a scope from process-os, referenced through
`libraries:` in `processos.yaml` (ported 2026-09-25, Milestone 9; see
`docs/plans/2026-09-25-process-os-port-completion-design.md`).

- Never edit a library in place. To change one, fork it into `../definitions/` under this
  repo's own namespace rules.
- Do not run the `process-os.*` actions until they are refined: they hard-code upstream's
  layout (`products/`, `packages/process-kit/`, `frameworks/`, `dist/`) and would write to
  wrong paths here. `sdlc` is layout-agnostic and safe.
- Notes about a library go in this file, not inside a scope folder — `process-cli check`
  rejects any file in a scope that isn't a kind folder or namespace.
