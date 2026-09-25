# platform/ — where packages, kits and frameworks live

Loaded automatically when working under `platform/`. Full design and the reasons behind every rule
below: `docs/proposals/bos-system-design.md` (section 3.1 is the taxonomy).

## Layout

`platform/<lang>/{packages/<group>/<package>, kits/<domain>, frameworks/<name>}` for `go` and `svelte`.
`platform/python` is one family (process-kit) belonging to another track and sits outside this
taxonomy.

- **package** — one job, one language, depends only downward.
- **kit** — one domain area across both stacks (handlers, repositories, migrations, routes,
  components, admin pages). Finer-grained than groups: a kit draws packages from several groups.
- **framework** — composes kits; no domain of its own (`bos-go`, `bos-svelte`).

## Placing a package

Test, in order — the first that fits wins:

1. Pure value type or logic, no I/O, no product vocabulary → `values`.
2. Technical plumbing (Go) or UI foundation (Svelte), no product vocabulary → `foundation`.
3. Product vocabulary → the domain area it serves: `identity`, `content`, `media`, `engagement`,
   `analytics`, `real-estate`. Create the folder when its first package lands — never pre-scaffold.
4. Used by two or more domain areas → it moves **down** (`values` / `foundation`), never sideways.

**Dependency direction:** `values` ← `foundation` ← domain areas. A domain area may import another only
through a declared edge. Known violation to invert: `scheduler` (foundation) imports `feed` (content).

The same groups apply to both stacks. Directory placement is independent of import identity: groups are
directories only.

## Rules for every port

- **Verbatim first.** Ported sources are byte-identical to the shredbx original. Authored files are only
  workspace roots, `go.mod`/`go.sum`, CI and docs. Any refactor (formatting included) is its own commit.
- **Names stay verbatim until M5:** Go module paths `github.com/shredbx/sbx-core/pkg/<name>`, npm names
  `@sbx/*`. Do not rename imports while porting.
- **Go modules:** one module per package (a sub-package with its own `go.mod` is a nested module inside its
  parent's folder). In-repo dependencies are declared as `require github.com/shredbx/sbx-core/pkg/<x> v0.0.0`
  plus a relative `replace`, so each module tidies and builds outside the workspace too. Third-party versions
  are pinned to the original `sbx-core/go.mod`; `go.sum` and indirect requirements come from `go mod tidy`
  (run leaves first). Confirm what a package imports with `go mod tidy`, not with a text scan.
- **Prove with the source's own tests:** same pass/skip/fail counts as the original, per package. Scoped
  runs only; the full battery is CI's job.
- **The word "capability" is reserved** for the 13 SDLC capabilities. Product areas are "domain areas".
- **Nothing in `platform/` names a client.** Client values live in the client's own repo.
- Check `git check-ignore` / staged-vs-on-disk counts after adding files: the root `.gitignore` carries a
  Python template whose `lib/` rule would swallow `src/lib/` (negated for `platform/svelte`).
