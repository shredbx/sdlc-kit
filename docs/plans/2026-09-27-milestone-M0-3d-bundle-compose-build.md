# Milestone M0.3d — the app bundle, definition 2: the `bundle-compose` template renders what a service now says

Status: done. Definition 2 of the app bundle (`docs/proposals/bos-app-roadmap.md`, section 4.2); the tree was confirmed on 2026-09-27, and the user asked
to proceed through the rest of the bundle until the example app runs. Decisions applied: D25 (an existing definition is extended, not duplicated), D26.

## What exists

```
processos-workspace/definitions/sbx-sdlc-kit/infrastructure/template/bundle-compose/files/docker-compose.yml.jinja   MODIFIED
```

For a service that has the new fields (`docs/plans/2026-09-27-milestone-M0-3c-service-build.md`) the template now emits:

| Field | Compose output |
|---|---|
| `build` | `build:` with `context:` and, when given, `dockerfile:`; both as JSON strings, so any path is valid YAML. `image:` stays, as the tag of what is built. |
| `healthcheck` | `healthcheck:` with `test: ["CMD-SHELL", "<test>"]` (the command as a JSON string), `interval:` from the record or `5s`, `timeout: 3s`, `retries: 20` (about 100 seconds to become healthy on a slow first start) |
| `depends_on` | `depends_on:` as a mapping, one entry per id, with `condition: service_healthy` when that service has a `healthcheck` and `service_started` when it does not |

A service with none of these fields gets none of these keys. A `depends_on` id that is not a member of the bundle fails the render (the engine is strict about
undefined names) and the message names the id: `'dict object' has no attribute 'acme-apii'`.

## Gates and evidence

| Gate | Result |
|---|---|
| `process-cli check` | ok |
| The `postgres-dev` bundle rendered through the extended template | `docker-compose.yml` and `.env.example` byte-identical to the committed files under `projects/services/postgres-dev/` |
| A scratch bundle of four services (a database with a health check whose command holds quotes, `$`, a backtick, a backslash and ` # `; a cache with no health check; two built application services), rendered and read back as YAML, 15 assertions | all pass: `build` with and without `dockerfile`; `depends_on` healthy against a service with a health check and started against one without; the hostile command round-trips exactly; interval given and defaulted; a service with none of the fields has none of the keys; ports, environment and container names unchanged; an unknown `depends_on` id fails and is named |
| `docker compose config -q` on that render (client side; the daemon is not needed) | accepted |
| A scan of the definitions and this plan for the product's names and ports | no match |

The renders were scratch files under the output root, removed afterwards. Nothing was written into `projects/services/` and no container was started.

## Not in this scope

The `verify-link` action, the `bootstrap-app-bundle` process, the service records for the two app halves, the Dockerfiles, and any docker run.
