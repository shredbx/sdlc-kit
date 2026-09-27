# Milestone M0.3f — the app bundle runs: two Dockerfiles and a `bos env -p bundle` profile

Status: done. This is the running-bundle half of what the user asked for (2026-09-27): "proceed till we establish consumer example application ... running as
well with minimum functionality from config", extended, once running, to cover the bundle too. Decisions applied: D24, D25, D26. The one network action
(pulling the two base images) was approved by the user before it happened.

## What exists

```
platform/go/frameworks/bos-go/
├── docker/Dockerfile                 NEW    assembles the API's image from a linux binary built beforehand; no module downloads
└── {cmd/bos/main.go, core/bosyaml/bosyaml.go}   MODIFIED   bos env gains -p dev|bundle
platform/svelte/frameworks/bos-svelte/
└── docker/Dockerfile                 NEW    assembles the web's image from a `pnpm build` made beforehand; no package downloads
```

**The Dockerfiles.** Each takes a folder of prebuilt output as its build context (a binary named `api`; a `build/` folder and `package.json`), so
building the image needs only its own small base image (`alpine:3.20`, `node:20-alpine`) and nothing from the network. Building the artifacts
themselves — the linux binary, the SvelteKit build — is the consumer's own job (its Makefile), because that is where the source lives.

**`bos env -p bundle`.** Profile `dev` (the default, unchanged) writes `api.env` and `web.env` for a native run. Profile `bundle` writes the one
`.env` a docker compose bundle reads: the two published ports, `ENVIRONMENT`, `DATABASE_URL`, the JWT secret (read back from `.env` this time, not
`api.env`, so each profile is self-contained), the API's allowed origin and the web's site name. Compose expands `$` inside a double-quoted `.env`
value but not inside a single-quoted one, so a value that needs quoting is single-quoted; a value that itself holds a single quote or a line break
cannot be written that way and is refused, naming the variable, rather than silently becoming a different value.

The `bos-go` record and README were updated to describe the profile and the quoting rule, re-rendered, and byte-compared against the install.

## A finding, found and fixed while running the gates: a bundle name can collide with a running client stack

A consumer's development machine may already run that same client's production-style docker-compose stack — this one does, under the product's plain
name. A bundle record named the same collides: docker compose derives its project name and default network from that name, and two projects with the
identical name share the same network regardless of which folder or which compose file they come from. Container names are a separate matter (they
did not collide here, by chance of what services each stack happens to run), but the network did.

Found immediately after the first `make bundle-up` in the consumer's own gates (not in this repository — sdlc-kit holds no consumer information):
the two new containers were stopped and removed by their exact names, never with `docker compose down` (which resolves containers by the shared
project label and network, and touching it was the exact risk to avoid). The pre-existing stack's containers were checked before and after — same
start times, zero restarts, still healthy — and were untouched throughout. The fix is a naming discipline, not a schema change: give a bundle likely
to run alongside a client's own stack a name that cannot collide (documented, with the reasoning, in the consumer's own Makefile and README).

**Not fixed here, and worth a future scope:** `render-bundle`, `run-bundle` and `verify-link` do not check whether another docker compose project
already uses the bundle's exact name before acting. A guard here would need its own design and approval; this milestone's fix is scoped to the one
bundle that hit it.

## Gates and evidence

| Gate | Result |
|---|---|
| `bos-go` build and vet, offline, `GOWORK=off`, after the profile change | ok |
| `process-cli validate`, `render`, `conform` and a byte compare of the re-rendered `bos-go` README | ok; identical |
| The dev-profile probe (31 checks) unchanged after the change | 31 of 31 |
| A new bundle-profile probe: a fresh secret then kept; `docker compose config` resolving every variable; a real (cached) container receiving a site name with quotes, `$`, `${}`, a backtick, a backslash and `#` exactly, and a database URL with `&`, `?`, `%`, `$` exactly; a port change moving both published ports and the derived origin; a single quote refused for the bundle profile (but accepted for `dev`); an unknown profile as a usage error; a missing file refused | 15 of 15 |
| A scan of both Dockerfiles and the changed framework files for the product's names and ports | no match |
| The consumer's offline artifact build (a linux API binary, the web production build) with the network denied | ok |
| The one approved network step: pulling the two base images | done once; everything after is offline |
| The consumer's `make bundle-up`: render, build both images from the cache alone, start, and `verify-link` reporting the API healthy and the web page linked | done; confirmed independently by `curl` on both published ports and by `docker compose ps` |
| The bundle-name collision: found, the two containers stopped by name, the pre-existing stack's containers confirmed untouched before and after, the bundle renamed, re-run confirmed on its own network and compose project | all as described above |
| `make bundle-down` | stopped and removed only the bundle's own two containers and its own network |
| The original project's `git status` | unchanged throughout |

The consumer's own records, Makefile and the result of running the bundle are documented in its own plan (`docs/bos-consumer-plan.md`), which sdlc-kit does
not duplicate.

## Not in this scope

A guard against a bundle name colliding with another running compose project; `bos dev`, `build` and `check`; the formalized (non-example) Makefile;
brand and tokens (M1).
