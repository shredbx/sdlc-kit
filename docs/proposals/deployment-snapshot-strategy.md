# Deployment snapshot strategy (proposal, 2026-10-04)

Status: **proposal, nothing modeled.** Written to record what already exists, compare it with an outside design, and name what a second
consumer would still need. No process-os definition is proposed here (see section 6).

## 1. The problem

Development works through the sdlc-kit tree: consumers are git submodules under `consumers/clients/{client}/{project}` and read platform
packages from `platform/` on the filesystem. A deploy (Dokploy builds from the consumer's own clone) has no access to sdlc-kit. So every
deploy needs a **snapshot of exactly the platform packages the product uses**, at a known revision, inside what the builder can see.

## 2. What exists today (verified by reading the files)

Only one consumer has done this, **bestays**, and only for **Python**.

| Piece | Where (under `consumers/clients/bestie/bestays/`) | What it does |
|---|---|---|
| Decision 9 | `docs/plans/2026-10-02-cleanup-restructure-deploy-plan.md` | Deploy = a vendored snapshot of the framework inside the consumer repo; `uv.lock` committed; Docker context = the consumer repo root. Reason recorded: Dokploy needs no access to sdlc-kit. |
| The snapshot | `vendor/agent-framework/` | Source only: `pyproject.toml` and `src/` (the framework's tests, caches and docs stay in sdlc-kit). |
| The stamp | `vendor/agent-framework/VENDORED.md` | sdlc-kit commit, "uncommitted changes when copied", timestamp, file count, sha256 content hash. |
| The tool | `scripts/vendor_framework.py` | Copies the snapshot and writes the stamp. `--check` fails if the folder was edited by hand, or (when sdlc-kit is reachable) has fallen behind the source. Standard library only. |
| The manifest | `projects/bestays_app/chat-api-python/pyproject.toml` | `agent-framework = { path = "../../../vendor/agent-framework", editable = true }`: the committed manifest already points at the snapshot, never into sdlc-kit. |
| The release | `scripts/release_app.py`, `DEPLOY.md` | Refuses to release when the tree is dirty, the tag exists, or the vendored framework is not sound; then tag plus a `release/<app>` branch that Dokploy watches. |
| The workflow | `CLAUDE.md` of bestays | Edit the framework in sdlc-kit and commit it there first; run `vendor_framework.py`; commit the snapshot; bump the submodule pointer in sdlc-kit. |

Why not a path source into sdlc-kit: recorded in the same plan, section 1: uv wants every sibling workspace member on disk
(`platform/python/` was 364 MB), so a build from the client repo alone could not work.

## 3. The outside design, compared

An outside conversation (pasted by the user, 2026-10-04) proposes: keep dev manifests pointing at the workspace; at deploy time build a
disposable `deployment/` tree with `application/` and `dependencies/{go,python,svelte}/`; **rewrite** the manifests inside that copy
(Go `replace`, Python `tool.uv.sources`, npm `file:`); record each dependency's git revision in a lock file; copy only the dependency
closure; run the packager from the sdlc-kit root; never commit the rewritten manifests.

| Point | Outside design | What we have | Judgement |
|---|---|---|---|
| Snapshot kept where | generated, disposable `deployment/` tree | committed inside the consumer repo | Ours is simpler for Dokploy (it clones one repo, no build step before the build). Cost: the snapshot is committed, so it needs the drift check we already have. |
| Manifest rewrite | rewritten in the staging copy | none: the committed manifest already points at `vendor/` | Ours has no rewrite step to get wrong, but the dev loop then also uses the snapshot (bestays tests run against the vendored copy, per the plan's section 5 note). |
| Revision record | `snapshot.lock` | `VENDORED.md` (commit and content hash) | Equivalent; ours also detects hand edits. |
| Dependency closure only | yes | yes for Python (`pyproject.toml` + `src/`) | Same rule. |
| One manifest for all platforms | `.sdlc/deployment.yaml` | none | Not needed until a second platform deploys. |
| Go, Svelte | `replace =>`, `file:` | not done | See section 4. |

The outside design is right about the principle (dev topology and deploy topology are separate; the snapshot is an explicit dependency
closure with a recorded revision) and about running the packager from the sdlc-kit side. It does not beat what bestays already does for Python.

## 4. What is not covered (the gap)

- **Go and Svelte.** `bestierealestate` (the `bos` consumer) is Go plus Svelte. Its own plan
  (`consumers/clients/bestie/bestierealestate/docs/bos-consumer-plan.md`, section 15) records that its Dockerfiles build from artifacts a
  mirror sync created (a vendor folder, pre-vendored `node_modules`, `uv.lock`) and "do not build in the working tree". Its `deploy/`
  folder holds only `docker-compose.yml`. `docs/proposals/bos-system-design.md` M6 defers "the mirror/vendor pipeline redone for a
  multi-repo layout" as its own plan, needing separate approval. Not checked here: how that consumer's Go and pnpm manifests reference
  `platform/` today (read them before deciding).
- **No sdlc-kit-level rule.** No proposal or decision record states the strategy. The `deployment` capability record
  (`processos-workspace/records/sbx-sdlc-kit/architecture/capability/deployment.yaml`) is one line. Decision 9 lives only in a consumer plan.
- **The script is bestays-specific.** `vendor_framework.py` hard-codes one framework path and one destination.

## 5. Proposed rule (to approve or change)

1. **Dev and deploy topologies stay separate.** The consumer never depends on the deploy builder seeing sdlc-kit.
2. **A deploy builds from the consumer repo alone.** Everything it needs is inside it: the snapshot of the platform packages it uses, plus the
   lock file for its own third-party dependencies.
3. **The snapshot is the dependency closure**: only what the product imports, source only.
4. **Every snapshot is stamped** with the sdlc-kit commit and a content hash, and a `--check` fails on a hand edit or on drift.
5. **Snapshots are never edited in the consumer.** Change the package in sdlc-kit, commit it there first, re-snapshot, commit the snapshot,
   then bump the submodule pointer (the order bestays already documents).
6. **Per platform, the committed manifest points at the snapshot**, using the platform's own local-path mechanism:

| Platform | Mechanism | Status |
|---|---|---|
| Python | uv `path` source to `vendor/<package>` | in use (bestays) |
| Go | `replace` to a vendored module folder, or `go mod vendor` | not decided; depends on how bos's Go modules reference each other |
| Svelte / pnpm | `file:` or a workspace link to a vendored package folder | not decided; same dependency |

Open choice for Go and Svelte: commit the snapshot in the consumer (as bestays does) or generate it in a disposable tree at build time (the outside
design). Recommendation: **commit it**, for the same reason as Decision 9: Dokploy then needs nothing beyond the consumer's own clone.

## 6. What to model, and when

Per this repo's rules (no definition ahead of a real task), **nothing yet**. The trigger is M6 for `bestierealestate`. When that scope is
picked up, the candidates are:

- a decision record under `architecture/decision/` for the rule in section 5 (after the user approves it);
- a `snapshot-dependencies` action that generalizes `vendor_framework.py` (inputs: package path, destination, include list; output: the folder and
  its stamp; a `check` mode), with the platform-specific manifest line as the only per-platform part;
- the `deployment` capability's first process, which calls that action and then `release`.

Each needs the user's explicit approval as its own definition.

## 7. Open questions for the user

1. Approve the rule in section 5 as the written strategy, and record it as a decision?
2. For Go and Svelte, commit the snapshot (recommended) or generate it at build time?
3. Move the strategy's first home to a decision record now, or keep this proposal until M6?
