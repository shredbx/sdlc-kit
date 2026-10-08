# Handoff — merging `worktree-bestierealestate--stage-2` (static pages, layout regions, SEO)

For whoever (human or a fresh Claude Code session, likely the **main** session/checkout) picks up
merging this branch into `main`. Assumes no memory of the conversation that did this work — only
this repository's own files plus whatever project memory the same account carries forward.

## What this branch adds, on top of `main`

Four commits on `worktree-bestierealestate--stage-2`, oldest to newest (`f7ec70f` is the branch tip
at the time of this note — confirm with `git log`), full detail in
`docs/plans/2026-09-28-bos-static-pages-seo-layout.md`:

1. `529554b` — the `AdminModule` protocol, the `cms` kit's `List` capability + a `layout` column,
   the bos default theme, header/footer chrome, a layout-preset registry, and a production visual
   pass — this is the bulk of the change (123 files).
2. `d18bb1e` — wires the already-ported `@sbx/ui-seo` into `cms` pages (editor + public `<head>`).
3. `a977a2f` — infrastructure capability: a `volumes` field + bos-demo's own persistent
   postgres/redis dev bundle (unrelated to the static-pages work itself — pre-existing dirty state
   in the worktree from an earlier point in the same overall session, committed here rather than
   left to rot, per explicit instruction not to lose it).
4. `f7ec70f` — doc status updates + two self-caught mistakes logged (a `PATCH /settings` gotcha, a
   layout-registry design correction).

**Scope D's real content (7 published pages, real header/footer nav) is data in the dev Postgres
database, not code** — nothing to look for in the diff for that part. A fresh environment starting
from this branch will have the mechanism but not that content; re-seed it by hand if needed (the
exact `curl` calls are in the plan doc's Scope D section, or just use the admin UI at `/admin`).

## No client-repo changes

Despite how the original request read, **nothing in the `consumers/clients/bestie/bestierealestate`
submodule changed** in this work — it was read directly (read-only) once, to ground the plan in BR's
real homepage/legal-page structure, per the standing no-client-info-in-sdlc-kit rule. Confirm before
merging:

```bash
git -C consumers/clients/bestie/bestierealestate status --porcelain   # expect: empty
git diff --stat HEAD~4 -- consumers/clients/bestie/bestierealestate   # expect: no output (gitlink unchanged)
```

If either shows anything, stop and investigate before merging — it would mean something touched the
submodule that this note doesn't know about.

## Before merging

- `process-cli check` from the repo root (schema/type/record changes landed in commit `a977a2f`).
- The usual gates already passed in-session and are not being asked to be re-proven blind: Go
  build/vet/test clean on `cms` + `bos-demo/apps/api`; `svelte-check` clean on `bos-demo/apps/web`
  (0 errors throughout, same pre-existing `state_referenced_locally` warning class only). Worth a
  fresh run if meaningful time has passed or anything else landed on `main` in the meantime.
- Two known, disclosed gaps are NOT blockers, just not done (see the plan doc's closing section):
  real markdown rendering for `Prose` (legal pages currently have no `<h1>`), and an actual
  browser-rendered visual check (no browser-automation tool was available in that session).

## Where the `bos` track stands after this merges

Layout regions, chrome, SEO, and 7 real static pages are done for `bos-demo`. **Next, per the plan
itself, is content packages** (e.g. a `contacts` kit) — deliberately not started; which one to build
first needs its own discussion with the user before any tree is proposed, per this repo's standing
per-scope-approval discipline (`CLAUDE.md`, "Workflow with the user").
