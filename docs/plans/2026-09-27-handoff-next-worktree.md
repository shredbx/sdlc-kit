# Handoff — resuming the `bos` track in a new worktree, after this branch merges to `main`

This is the resumption note for whoever (human or a fresh Claude Code session) opens a **new** worktree after
`worktree-bestierealestate` merges to `main`. It assumes no memory of the conversation that did this work — only
this repository's own files, plus whatever project memory the same account carries forward.

## What just merged

Everything this branch accumulated: the platform's Go and Svelte packages ported from `shredbx` (Scopes 1 to 6e),
the unit/package record model (Scopes U0 to U2c — one `package.yaml` + rendered `README.md` per unit), and the
`bos` decomposition through **milestone M0**: the `bos-go` and `bos-svelte` frameworks extracted from a running
production app, and the first consumer's example app running both natively (`make dev`) and as a docker bundle
(`make bundle-up`), each gated offline and against the original app's own golden files. Full design:
`docs/proposals/{bos-system-design,bos-constructor,bos-app-roadmap}.md` (decisions D1 to D26, as prose inside
those documents). Milestone-by-milestone detail: `docs/plans/2026-09-2[4-7]-*.md`, especially the `M0.*` series.

**Separately, not part of this**: `feature/agent-framework` is another, independently active branch (agent
framework work only, under `platform/python/frameworks/agent-framework/`). It is not related to the `bos` track
and this handoff says nothing further about it — start from `main` for `bos` work regardless of that branch's
own state.

## Setting up the new worktree

```bash
git -C /Users/solo/Projects/workspaces/sdlc-kit fetch origin
git -C /Users/solo/Projects/workspaces/sdlc-kit worktree add <new-path> -b <new-branch-name> origin/main
```

Then, **the one gotcha that matters most**: this repository's own commits **never** stage the consumer submodule's
pointer (standing rule — see `feedback_module-paths-in-sdlc-kit` / `feedback_no-client-info-in-sdlc-kit` in project
memory, and the many `M consumers/clients/bestie/bestierealestate` lines throughout this session's own reports).
Because of that, the gitlink `main`'s tree records for `consumers/clients/bestie/bestierealestate` is frozen at a
very early commit (`9715beb`, from when the submodule was first added) — **not** the current state. A plain
`git submodule update --init` will silently check that old commit out, with no error, and everything built through
M0 (`bos.yaml`, `apps/api`, `apps/web`, the bundle records, `deploy/`) will simply be missing.

```bash
git -C <new-path> submodule update --init consumers/clients/bestie/bestierealestate
git -C <new-path>/consumers/clients/bestie/bestierealestate fetch origin
git -C <new-path>/consumers/clients/bestie/bestierealestate checkout main
git -C <new-path>/consumers/clients/bestie/bestierealestate pull
```

The consumer repo's own `origin/main` was pushed as part of this handoff and should be at `165609e` or later.
Confirm with:

```bash
git -C <new-path>/consumers/clients/bestie/bestierealestate log -1 --oneline
```

If it prints anything other than `165609e` (or a commit after it), stop and re-check before doing anything else —
the whole M0 result depends on that submodule being current.

## Verifying the new worktree is healthy

```bash
process-cli check                                                        # from sdlc-kit's own root
process-cli --config <new-path>/consumers/clients/bestie/bestierealestate/processos.yaml check   # the consumer's own workspace
make -C <new-path>/consumers/clients/bestie/bestierealestate parity      # 18 of 18 golden cases
make -C <new-path>/consumers/clients/bestie/bestierealestate dev         # open http://localhost:4003
```

`make dev` should show "BestieRealEstate: it works." and "API: healthy". `make bundle-up` also works, but needs
`make bundle-bases` once per machine first (the one step that reaches the network — see the gotcha below).

## Where the `bos` track stands, and what's next

M0 is done. **Next: M1 — branding and tokens** (colors, logo, assets, fonts), per the roadmap's M1 section
(`docs/proposals/bos-app-roadmap.md`). Nothing about M1 is modeled yet — discuss and plan it fresh with the user,
per this repository's own "Workflow with the user" (`CLAUDE.md`): discuss intent → plan the shape → per-definition
approval → before/after tree → implement → `process-cli check`. Read the milestone roadmap and the consumer's own
`docs/bos-consumer-plan.md` (in the consumer repo — it, not sdlc-kit, holds this product's own state and recon)
before proposing a tree.

## Gotchas carried forward, so they aren't rediscovered the hard way

- **The `bos-svelte` `/api` pass-through has no timeout of its own.** An API that accepts a connection and never
  answers holds the request open indefinitely. Known, not fixed (see the M0.3 plan).
- **Docker Compose project names are a global namespace, not scoped by folder.** This machine also runs the
  original app's own docker-compose stack under the plain product name. Never name a bundle identically to a
  client's own running stack — the consumer's bundle is deliberately named `bestierealestate-dev`, not the plain
  name. Full incident: `docs/plans/2026-09-27-milestone-M0-3f-bundle-run.md`. `render-bundle`/`run-bundle`/
  `verify-link` do **not** guard against this themselves yet — a future scope, not built.
- **The submodule pointer discipline is not optional.** Never stage `consumers/clients/bestie/bestierealestate`'s
  pointer change in an sdlc-kit commit, however tempting it is to "just bump it" after a lot of consumer work.
- **`process-cli render`/`conform`** take folders **relative to the workspace's `output/` root**, not the repo
  root — an absolute or `output/`-prefixed `--into` path nests wrong (`processos-workspace/output/output/...`).
- **A unit's README is always a render, never a hand edit.** Edit `readme.yaml`, then render, `conform`, and byte-
  compare against the installed `README.md` before committing.
