# Infrastructure services: design for the first bundle (postgres + pgAdmin)

**Status:** validated in brainstorming, 2026-09-25 — not yet implemented. Each definition below
still needs its own explicit approval before it's written, per CLAUDE.md's per-item workflow.

## What this covers

A repeatable shape for standing up off-the-shelf infrastructure (databases, their admin tools,
and later our own built services) as docker bundles other products and experiments can run
locally — starting with exactly one: postgres + pgAdmin. Redis, Mongo, their own management
tools, and any self-built service (whisper wrapper, embeddings, the future AI assistant) are
explicitly **not** modeled yet — this milestone stops at one working bundle.

## Placement

Two different things, two different homes — do not conflate them:

- **Workspace/definitions side** — nests under the `infrastructure` capability, one of the 13
  already locked and recorded under `architecture/capability/` (never a new top-level folder;
  see `processos-workspace/definitions/CLAUDE.md`):
  `processos-workspace/definitions/sbx-sdlc-kit/infrastructure/{type,schema,action,process,template}/`
- **Filesystem/runtime side** — rendered, runnable docker artifacts. Nests under `projects/`,
  per the already-decided `filestructure-nesting` record (`products/`, `experiments/` nest there
  too, to keep repo root clean): `projects/services/<bundle-name>/`. Neither `projects/` nor
  `projects/services/` exist on disk yet — first real use materializes them, per this workspace's
  own lazy-nesting rule.

## Types & schemas

```
type/service-kind.yaml    — string, enum: [datastore, tool, application]
type/port-list.yaml       — sequence, items: string   ("host:container" entries)
type/env-var-list.yaml    — sequence, items: string   (required env var NAMES, never values)
type/service-ref-list.yaml— sequence, items: string   (other service records' ids)
type/service-list.yaml    — sequence, items: service  (full service mappings, render-time only)

schema/service.yaml
  name, kind (service-kind), description, image: string, all required
  ports: port-list, env: env-var-list, volumes: (tbd — see open item below), all optional
  management_tool: string, optional — another service's id; only meaningful when kind: datastore

schema/bundle.yaml
  name, description: string, required
  services: service-ref-list, required — member service ids

schema/bundle-spec.yaml   — ephemeral, render-input only, never a stored record
  bundle: bundle, services: service-list   (the joined, resolved form of a bundle)
```

`service-kind`/`port-list`/`env-var-list`/`service-ref-list` are kept as distinct named types
rather than one generic reusable list, matching process-os's own precedent (`word-list`,
`package-list`, `file-list` are structurally identical but kept semantically separate).

**Open item, deliberately unresolved:** whether `service.volumes` needs its own type or can reuse
`port-list`'s shape — decide against the real postgres render, not by guessing now.

## This milestone's records

```
processos-workspace/records/sbx-sdlc-kit/infrastructure/
  service/postgres.yaml   kind: datastore, image: postgres:16, ports: ["54320:5432"],
                          env: [POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB],
                          management_tool: pgadmin
  service/pgadmin.yaml    kind: tool, image: dpage/pgadmin4:8, ports: ["5050:80"],
                          env: [PGADMIN_DEFAULT_EMAIL, PGADMIN_DEFAULT_PASSWORD]
  bundle/postgres-dev.yaml  services: [postgres, pgadmin]
```

## Rendering pipeline

The git-tracked `bundle` record only holds service ids — the template needs the full joined
data. Confirmed against real precedent (`sdlc.python.create-project`'s `action.sh`/`lib.sh`,
which builds a temp YAML by reading `$ACTION_INPUTS/application.yaml` and re-indenting it under a
new key before calling `render`) rather than guessed:

- `infrastructure/template/bundle-compose/` — `files/docker-compose.yml.jinja` (one compose
  service block per member, referencing `${POSTGRES_PASSWORD}` etc.) + `files/.env.example.jinja`
  (one line per env var name, no values). Tested by rendering a real `service-bundle-spec`
  through it (`process-cli render`, `--into` a scratch folder) — output is valid, correct YAML.
- `infrastructure/action/render-bundle` (shell) — reads the bundle record, resolves each member
  service id via `process-cli show record`, assembles the temp `bundle-spec` YAML the same way
  `create-project` assembles `project-spec`, renders with `process-cli render --into` a scratch
  folder under `processos-workspace/output/`, then copies the result into
  `projects/services/<bundle-name>/`. **Correction from the original draft of this doc:**
  `--into` is hard-constrained to `runtime.output` (confirmed in `process_framework/framework.py`
  — "not inside the output folder" — not any real path). `sdlc.python.create-project`'s own
  `--into "$INPUT_BUILD_TARGET"` only looks like it writes anywhere because process-os's *own*
  `processos.yaml` sets `runtime.output: .` (the whole repo) — a workspace-specific choice we
  haven't made and aren't making here, to keep `processos-workspace/output/` as scratch-only per
  this repo's existing governance. Render-then-copy is the fix, confirmed working.
- `infrastructure/action/run-bundle` (shell) — `docker compose up -d` in that folder (Compose
  auto-reads a sibling `.env`); `post.sh` verifies with `pg_isready` + a pgAdmin health check —
  this is the "test with terminal tools" step closing out the milestone
- `infrastructure/process/bootstrap-bundle` — steps `[render-bundle, run-bundle]`

## Secrets

`projects/services/<bundle-name>/docker-compose.yml` and `.env.example` are git-tracked — no
secrets, only `${VAR}` references and variable names. Only `.env` in that same folder is
gitignored, holding the real password; Docker Compose reads it automatically, no extra plumbing.
1Password integration (`op://...` refs, matching shredbx's own `.env.infra` convention) is
explicitly deferred to when a real production deployment target exists.

## Explicitly out of scope for this milestone

- Redis, Mongo, and their management tools (redis-commander, mongo-express, ...) — same shape,
  not modeled until a real need exists
- Any self-built service (whisper wrapper, embeddings, the future AI assistant) — its source
  code, once real, lands in `platform/<lang>/apps/` (confirmed decision); only its deployment
  bundle would eventually live under this same `infrastructure/` + `projects/services/` shape
- 1Password-backed secrets, and any non-dev deployment target
- A boundary/lint checker enforcing any of this — deferred per
  `architecture/decision/nested-claude-md-governance/`; a checker comes after the rule is real
  and drift has actually recurred, not ahead of it

## Related governance added this session

- `processos-workspace/definitions/CLAUDE.md` — locks the 13-capability namespace, auto-loaded
  by Claude Code whenever a file under `sbx-sdlc-kit/` is read
- `architecture/decision/nested-claude-md-governance/` — why that mechanism was chosen
