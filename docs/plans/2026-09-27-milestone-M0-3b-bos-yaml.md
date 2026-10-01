# Milestone M0.3b — `bos.yaml` and the `bos` command: one file both halves take their settings from

Status: done. The code is gated, and the `bos-go` record changes and the re-rendered README were approved and written on 2026-09-27 (the README is
byte-identical to the render). Decisions applied: D24 (local links, nothing fetched),
D25 (moved or configured before written; this is the roadmap's only new code in M0), D26 (no tests until the core is validated). Roadmap:
`docs/proposals/bos-app-roadmap.md`, M0, deliverable 5.

**Scope.** Replace the settings block a consumer's example Makefile carried (shell text writing the two env files) with a file the consumer owns,
`bos.yaml`, and a command that turns it into the two env files. The docker bundle, `bos dev`, `bos build` and `bos check` stay in later scopes.

## What exists

```
platform/go/frameworks/bos-go/
├── core/bosyaml/bosyaml.go     NEW   File{Prefix, SiteName, Environment, API{Port, DatabaseURL}, Web{Port}}; Load (strict: unknown key refused);
│                                     APIEnv(secret) and WebEnv(); Lines() renders shell-sourceable NAME=value lines
├── cmd/bos/main.go             NEW   `bos version`; `bos env [-f bos.yaml] [-o .bos/env]`
└── go.mod, go.sum              MODIFIED  + gopkg.in/yaml.v3 v3.0.1
```

**The link.** Neither half states the other's address. The API's allowed origin is derived from `web.port` and the web's API address from `api.port`,
so changing one port in `bos.yaml` moves both halves. The API's variable that scopes the product name comes from `prefix`, so the framework holds no
product name. `bos env` keeps an existing JWT secret and generates one (32 random bytes as 64 hex characters) only when there is none. It writes each
file through a temporary file and a rename, so a reader never sees half a file.

**Deviations from the tree that was confirmed for this scope.**

| Confirmed | Done | Why |
|---|---|---|
| `bos.yaml` holds an app name | it does not | nothing reads one: the API takes its name from its own defaults and the web shows the site name |
| `bos.yaml` holds an allowed origin | it does not; the origin is derived from `web.port` | a second place to state a port is the disagreement `bos.yaml` exists to remove |
| the environment is not mentioned | `environment` is a field (default `dev`) | the shell block this replaces wrote a fixed `dev`; the API's HSTS header depends on it |
| the consumer's API Makefile is not mentioned | its `check` target now asks the `bos` usage text whether `check` exists | it ran `bos check` as soon as `cmd/bos` existed, and this scope adds only `version` and `env` |
| the env files are written as before | `api.env` is mode 0600 (it holds the secret); it was 0644 | the content is unchanged |

## Gates and evidence

| Gate | Result |
|---|---|
| `go build ./...` and `go vet ./...` in `bos-go`, with the network denied at the go tool (`GOPROXY=off`, `-mod=readonly`, the checksum database left on): `GOWORK=off` and workspace mode | ok |
| A cold build of `cmd/bos` (empty build cache, `GOWORK=off`) under a sandbox that denies all network | ok, from the module cache alone |
| The 8 lines this scope added to `go.sum` (checksums of yaml's own test dependencies) against the same lines in other `go.sum` files of the repository, written when they were verified online | all 8 identical |
| `make env` against the files the replaced shell block wrote (the same secret) | `api.env` and `web.env` byte-identical |
| A probe of the command against scratch files: version and usage (exit 2), a fresh 64-hex secret then kept on a second run, a port change moving both halves, a site name and a database URL with shell syntax surviving a real `sh` round trip, environment default, the default `-f` and `-o`, no leftover temporary file, and 13 refusals (unknown or misspelt key, missing or lower-case prefix, no site name, port 0, too large, missing, equal ports, bad environment, empty file, not YAML, missing file) each exiting 1, naming the file and writing nothing | 31 of 31 |
| The consumer's `make check` and `make parity` | both apps' checks ok; 18 of 18 golden cases identical |
| The end-to-end probe of the example app under a network-denying sandbox, its developer path starting from `make dev` | 27 of 27 |
| A scan of `bos-go`, its record and this plan for the product's names and ports | no match |
| `process-cli validate` of the changed `bos-go` records, `render`, `conform`, and a byte compare of the installed README | ok; identical |
| `process-cli check`, and `platform/tools/check_kit_edges.py` | ok; 0 errors (yaml is third-party, so no kit edge changes) |
| The original project's `git status` | unchanged |

The probe is a throwaway script; no test code enters the repository (D26).

## Noted for later

**Ports from a registry.** The user's ruling on 2026-09-27: the ports will later be assigned from an open-port registry in the sdlc workspace (the
available ports, managed per project), and the Makefile's development run passes them on. Not introduced now. It shapes one thing already visible
here: `bos env` will need the two ports as arguments that override `bos.yaml`, whose values then serve as defaults. See the roadmap, section 4.3.

## Not in this scope

`bos dev`, `build` and `check`; the docker bundle, its action and process; the formalized Makefile. Reading `bos.yaml` for the Go API's own defaults
(today the consumer's `main` passes them in code, as before) is left to the scope that needs it.
