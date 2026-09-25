# platform/ — where packages, kits and frameworks live

Loaded automatically when working under `platform/`. Full design and the reasons behind every rule
below: `docs/proposals/bos-system-design.md` (section 3.1 is the layout).

## Layout

`platform/<lang>/{packages/<kit>/<package>, frameworks/<name>}` for `go` and `svelte`.
`platform/python` is one family (process-kit) belonging to another track and sits outside this layout.

Three levels: **role** (`packages/` vs `frameworks/`; a `tools/` may join later), **kit**, **package**.

- **package** — one library, one job, one language, own module; depends only downward.
- **kit** — one functionality's family of packages: the top-level folder under `packages/` (`seo`,
  `calendar`, `contacts`, `persistence`). It grows by adding packages; nothing moves. Later it also carries
  its bos wiring (handlers, migrations, routes, admin pages) — where that lives is open decision D15.
- **framework** — composes kits; no domain of its own (`bos-go`, `bos-svelte`).

## Naming and placing a package

1. **Placement test:** which functionality does this package serve? Put it in that kit. Create the kit when
   its first package lands — never pre-scaffold. A package no single feature owns gets a shared kit named
   for what it does (`persistence`, `http`, `money`).
2. **Naming test:** a kit name must tell a newcomer what is inside without its parent folder. Generic nouns
   (`property`, `transaction`, `collection`, `dictionary`, `units`) are package names only, under a kit that
   qualifies them (`real-estate/property`, `reference-data/dictionary`). Never `utils`, `common`, `core`,
   `shared` or `misc` as a kit.
3. **A package named like its kit is the kit folder:** `packages/seo`, not `packages/seo/seo`. Every other
   package is a subfolder (`packages/calendar/ical`). A kit with no package of its own name is just a group
   (`identity/`, `contacts/`).
4. The same kit names apply on both stacks (`seo` holds the Go package `seo` and the Svelte package `ui-seo`).
5. **Not bos:** SDLC/workspace tooling (`sdlc`, `vault`, `project`, `workspace`, `trace`, `dokploy`, …)
   never enters these trees.
6. Used by two or more feature kits → it moves **down** into a shared kit, never sideways.

**Dependency direction:** shared kits (`persistence`, `http`, `money`, `location`, `localization`,
`formatting`, `notifications`, `jobs`, `ui`, `reference-data`) never import feature kits; no cycles; a
feature kit imports another only through a declared edge. Known violation to invert: `scheduler` (`jobs`)
imports `news/feed`.

Directory placement is independent of import identity: kits are directories only.

## Rules for every port

- **Verbatim first.** Ported sources are byte-identical to the shredbx original. Authored files are only
  workspace roots, `go.mod`/`go.sum`, CI and docs. Any refactor (formatting included) is its own commit.
- **Port only what the app uses.** Other product packages wait for a consumer that needs them.
- **Names stay verbatim until M5:** Go module paths `github.com/shredbx/sbx-core/pkg/<name>`, npm names
  `@sbx/*`. Do not rename imports while porting.
- **Go modules:** one module per package (a sub-package with its own `go.mod` is a nested module inside its
  parent's folder). In-repo dependencies are declared as `require github.com/shredbx/sbx-core/pkg/<x> v0.0.0`
  plus a relative `replace`, so each module tidies and builds outside the workspace too. Third-party versions
  are pinned to the original `sbx-core/go.mod`; `go.sum` and indirect requirements come from `go mod tidy`
  (run leaves first). Confirm what a package imports with `go mod tidy`, not with a text scan.
  **`replace` does not propagate:** a module replaces its whole in-repo dependency closure, direct and
  transitive (`vcard` requires two in-repo modules and needs nine `replace` lines). Compute the closure from
  the ported `go.mod` files.
- **Prove with the source's own tests:** same pass/skip/fail counts as the original, per package. Scoped
  runs only; the full battery is CI's job.
- **pnpm 11 fails an install on an unreviewed dependency build script.** Read the script, then record the
  decision in `platform/svelte/pnpm-workspace.yaml` under `allowBuilds` (`false` keeps it denied); never
  approve blindly.
- **The word "capability" is reserved** for the 13 SDLC capabilities. Product areas are "domain areas".
- **Nothing in `platform/` names a client.** Client values live in the client's own repo.
- Check `git check-ignore` / staged-vs-on-disk counts after adding files: the root `.gitignore` carries a
  Python template whose `lib/` rule would swallow `src/lib/` (negated for `platform/svelte`).
