# Milestone M0.3c — the app bundle, definition 1: the `service` schema learns to be built from source

Status: done. Definition 1 of the four the roadmap lists for the app bundle (`docs/proposals/bos-app-roadmap.md`, section 4.2). Approved as a tree on
2026-09-27; definitions are approved one at a time, so the template, the `verify-link` action and the `bootstrap-app-bundle` process each get their own
tree next. Decisions applied: D25 (a new definition extends an existing one before it adds a near-duplicate), D26 (no test code in the repository).

**Scope.** Let a `service` record say three things the two app services need and the datastore records never did: how to build its image from source, which
other services it waits for, and when it counts as healthy. Nothing renders them yet, and nothing runs.

## What exists

```
processos-workspace/definitions/sbx-sdlc-kit/infrastructure/schema/
├── service.yaml              MODIFIED  + build, depends_on, healthcheck (all optional)
├── service-build.yaml        NEW       {context: string, required; dockerfile: string, optional}
└── service-healthcheck.yaml  NEW       {test: string, required; interval: string, optional}
```

| Field | Type | Meaning |
|---|---|---|
| `build.context` | string, required | the docker build context, relative to the folder the bundle's `docker-compose.yml` is rendered into |
| `build.dockerfile` | string, optional | the Dockerfile to use when it is not `Dockerfile` in the context |
| `depends_on` | `service-ref-list` (the existing type: other services' ids) | services this one waits for |
| `healthcheck.test` | string, required | a shell command run inside the container that exits 0 when the service is healthy |
| `healthcheck.interval` | string, optional | how often it runs, as a compose duration such as `5s`; the template supplies a default |

**Choices, as proposed and confirmed.**

- `image` stays required. For a built service it names the image the build produces, so the three existing records, the `bootstrap-bundle` process and
  its rendered output are untouched.
- `depends_on` reuses `service-ref-list`. The wait condition is not a field: the template (definition 2) uses "healthy" when the other service has a
  `healthcheck` and "started" when it does not, so a condition can never contradict the health check.
- `healthcheck` holds only `test` and `interval`. More settings join when something needs them.

## Gates and evidence

| Gate | Result |
|---|---|
| `process-cli check` | ok; both new schemas are listed in the scope |
| The three existing service records against the extended schema, unchanged | all validate |
| A scratch record using `build`, `depends_on` and `healthcheck`, and records using each field alone or none of them | all validate |
| Bad shapes: a build without a context, a build or healthcheck that is not a mapping, a healthcheck without a test, a `depends_on` that is not a list, an unknown key in `build`, an unknown key in `healthcheck` | all 7 refused, each message naming the field |
| The `postgres-dev` bundle rendered through the `bundle-compose` template, before and after | `docker-compose.yml` and `.env.example` byte-identical; also identical to the committed files under `projects/services/postgres-dev/` |
| A bundle that includes the extended scratch record, rendered through the unchanged template | renders; the new fields are not emitted yet (that is definition 2) |
| A scan of the definitions and this plan for the product's names and ports | no match |

The records and the joined specs used were scratch files outside the repository. The `render-bundle` action was not run, because it writes into the tracked
`projects/services/`, and nothing was started with docker.

## Design input for the next definitions (findings, not decisions)

- **The original's Dockerfiles do not carry over.** They are generated for a vendored deployment mirror (`vendor/`, a pnpm-deployed `node_modules`, a secrets
  entrypoint), which does not fit a bundle built from linked directories. The reusable parts are the multi-stage shape, the Go and Node base images, and the
  health-check URLs. The bundle's Dockerfiles are therefore new and small; where they live is a question for the template definition.
- **The build context has to be the sdlc-kit root.** The Go `replace` paths and the pnpm links of an app point up to it, so an image can only be consistent
  if it is built from a context that keeps that tree shape.
- **Base images are a one-time registry pull**, a network action that needs the user's explicit approval before it happens.
- `service-kind`'s comment still says an application's code "lives in platform/<lang>/apps/"; today an app's code lives in its consumer's `apps/`. Noticed,
  not changed here.

## Not in this scope

The `bundle-compose` template extension (rendering `build:`, `depends_on` with a condition and `healthcheck:`), the `verify-link` action, the
`bootstrap-app-bundle` process, the service records for the two app halves, the Dockerfiles, and any docker run.
