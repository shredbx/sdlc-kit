# Milestone M0.1 — `bos-go`, extracted from a running API

Status: done; committed locally, not pushed. Decisions applied: D24 (names and local links),
D25 (moved and configured, not rewritten), D26 (no tests until the core is validated). Roadmap: `docs/proposals/bos-app-roadmap.md`, M0.
M0.0 (renaming the ported modules) was not chosen: the ported modules keep `github.com/shredbx/sbx-core/pkg/*` until the naming pass.

## What exists

```
platform/go/frameworks/bos-go/            module github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go
├── go.mod, go.sum                        chi v5.1.0, go-chi/cors v1.2.1, the ported auth and httputil;
│                                         replace to relative paths for the whole in-repo closure: auth, httputil, money, repository, user
├── core/config/config.go                 Config, Defaults, Load(prefix, defaults), CorsOrigins
├── server/app.go                         Deps, NewApp(cfg, deps), Run(cfg, handler)
├── server/middleware/recoverer.go        JSONRecoverer
├── server/routes/health.go               GET and HEAD /health, HealthResponse, Pinger
├── server/routes/root.go                 GET /
└── README.md                             rendered from the record, byte-identical to the render
platform/go/go.work                       + ./frameworks/bos-go
processos-workspace/records/sbx-sdlc-kit/modeling/package/bos-go/{package.yaml, readme.yaml}
```

The record's `uses` names `auth` and `httputil`, which have no unit records yet; the schema accepts it, and the ids resolve when those
kits get their records.

## What was moved, and what was configured

The code is the running API's, taken from its `main` package and its handler package; the line ranges are in the consumer's plan (its section
on the extraction map), not here. Each piece was compared with its original, whitespace aside.

| Piece | Against the original | Recorded difference |
|---|---|---|
| `HealthResponse`, `writeJSON`, `getEnv`, `JSONRecoverer` | identical | none |
| Server start and 30 s graceful shutdown (`Run`) | identical | the settings come from `cfg` and the handler is passed in |
| The chain (request id, logger, JSON recoverer, security headers, CORS, auth extraction) | identical order and options | the recoverer is this module's; the CORS allowlist and the environment come from `cfg`; auth extraction is added only when `Deps.Auth` is set |
| `GET` and `HEAD /health` | identical | the database ping goes through the `Pinger` interface (a pgx pool satisfies it); the pool is no longer reached through the app |
| Root info | identical keys | the endpoint list is the consumer's (`Deps.Endpoints`); with none it names `/health` |
| `getCorsOrigins` (`CorsOrigins`) | identical, including the refusal of `*` | the variable is `<prefix>_CORS_ORIGINS` (the consumer passes its prefix); the fallback is a parameter (default `http://localhost:*`); the list is read in `Load`, so a forbidden value stops startup |
| `loadConfig` (`Load`) | the spine's fields only | the default name, port, database address and base URL are the consumer's (`Defaults`); framework fallbacks are version `0.1.0`, environment `development`, port `8080`, no database |

Left in the app, not moved: the settings of the kits that have not moved yet (object storage, the lead-alert channel, the video and maps
and fonts keys, the canvas image origins, the feed-sources path, the upload limits). They move with their kit.

Security headers, CSRF and authentication extraction are the ported `auth` package, imported as it is; the recoverer's error body is the
ported `httputil`.

## One addition the golden files forced

A chi group with no routes never runs its middleware, so in an empty app a `POST` to an unknown `/api` path got a 404 where the baseline
refuses it with the CSRF check. The running API never showed this because its `/api` group always has routes. The group now has a catch-all
that answers with chi's own not-found, **only when the consumer mounts no routes**, so the check stays in front of unknown paths. With routes
mounted nothing is added, and the default not-found and method-not-allowed answers are unchanged.

## Gates and evidence

| Gate | Result |
|---|---|
| `go mod tidy`, then `go build ./...` and `go vet ./...` with `GOFLAGS=-mod=readonly`, `GOPROXY=off`, `GOWORK=off`, resolver canary `GODEBUG=netdns=go+2` | clean; the canary printed nothing |
| Cold build (fresh build cache) under `sandbox-exec` with all network denied | clean |
| The same build in workspace mode (`platform/go/go.work`) | clean |
| Every line of `go.sum` (66) is already recorded by the original project's `go.sum` or a ported module's | 66 of 66 |
| The in-repo closure that tidy found equals the five modules replaced | auth, httputil, money, repository, user |
| A scan of `bos-go` for the product's names, its ports, its default database address and its kit settings | no match |
| `gofmt -l` | nothing to format |
| `process-cli validate` of both records; `conform` of the render; byte compare of the installed README with the render; `process-cli check` | ok; ok; identical; ok |
| `platform/tools/check_kit_edges.py` | 0 errors |
| The original project's `git status` compared with its state before | unchanged |
| A throwaway program in the scratchpad (not in any repository) replays the 18 golden cases of the baseline against an app built on `bos-go` | 18 of 18 identical; a run on defaults answers as documented |

The replay is a preview of the M0.2 parity run, not a test in the repository (D26). M0.2 gives the consumer's harness a mode that targets the
new app.

Accepted risk (roadmap, section 5): the module compiles in the auth package's dependencies (a JWT library, a Postgres driver, a Redis
client, the crypto library) because the middleware is imported as it is; no service is contacted. The `go` directive is 1.26 and the local
`go` is 1.25.6: the gates ran with the cached go1.26.2 toolchain (`GOTOOLCHAIN=local`), as in the earlier Go scopes.

## Not in this scope

No tests (D26), no `bos.yaml` reader, no `cmd/bos`, no pages, sources or presets. Next: M0.2, the consumer skeleton and a mode of the
consumer's harness that targets the new app.
