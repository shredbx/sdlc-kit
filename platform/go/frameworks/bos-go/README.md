# bos-go

The Go half of a bos app: settings, a fixed middleware chain, the health and root routes and graceful shutdown around a chi router, so an API's `main` is a few lines.

```bash
go build ./...    # build and type-check this module
go vet ./...      # vet this module; it has no tests yet
go run ./cmd/bos env -f bos.yaml -o .bos/env    # write both halves' environment files, for a native run
go run ./cmd/bos env -p bundle -f bos.yaml -o deploy    # write the one .env a docker compose bundle reads
```

## Overview

`bos-go` is the code every bos API starts with, taken from a running API and configured rather than rewritten. `config.Load` reads the settings from the environment. `server.NewApp` builds a chi router with a fixed chain, in this order: request id, request logger, JSON panic recovery, security headers, CORS and, when an auth service is given, authentication extraction. It serves `GET` and `HEAD /health`, `GET /` (service, version, environment and the endpoints you list) and an `/api` group behind the CSRF check, where you mount your own routes. `server.Run` serves a handler and, on SIGINT or SIGTERM, drains in-flight requests for up to 30 seconds.

`cmd/bos` is the command line. `bos env` writes the app's settings from `bos.yaml` (package `core/bosyaml`), so every way of running the app reads them from one file and cannot disagree. Profile `dev` (the default) writes the two files a native run reads, `api.env` and `web.env`. Profile `bundle` writes the one `.env` a docker compose bundle reads.

Security headers, the CSRF check and authentication are the `auth` package, imported as it is; the error body of a recovered panic is `httputil`'s. Not here yet: `bos dev`, `build` and `check`, pages, content sources and presets.

## Install

`bos-go` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`.

From a consumer, list it in your `go.work` and, in your `go.mod`, require it and replace it **and its whole in-repo closure**: a `replace` does not propagate, and a `require` without one makes the go tool look the module up on the network. The paths are relative to your `go.mod`:

```
require github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go v0.0.0

replace (
	github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go => <path to platform/go/frameworks/bos-go>
	github.com/shredbx/sbx-core/pkg/auth       => <path to platform/go/packages/identity/auth>
	github.com/shredbx/sbx-core/pkg/httputil   => <path to platform/go/packages/http/httputil>
	github.com/shredbx/sbx-core/pkg/money      => <path to platform/go/packages/money>
	github.com/shredbx/sbx-core/pkg/repository => <path to platform/go/packages/persistence/repository>
	github.com/shredbx/sbx-core/pkg/user       => <path to platform/go/packages/identity/user>
)
```

The ported packages keep their original import names until the naming pass.

The `bos` command needs no separate install. Build it from this module: `GOWORK=off go -C <path to bos-go> build -o <where you want it> ./cmd/bos`. It uses `gopkg.in/yaml.v3` from the module cache.

## Usage

A whole API with no routes of its own:

```go
package main

import (
	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/core/config"
	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/server"
)

func main() {
	cfg := config.Load("ACME", config.Defaults{AppName: "acme-api", Port: "5000"})
	server.Run(cfg, server.NewApp(cfg, server.Deps{}))
}
```

`GET /health` then answers:

```
{"status":"healthy","service":"acme-api","version":"0.1.0","environment":"development","database":"disconnected"}
```

The settings of both halves, from one file:

```yaml
# bos.yaml
prefix: ACME
site_name: Acme
api:
  port: 5000
web:
  port: 3000
```

`bos env` then writes `.bos/env/api.env`:

```
ENVIRONMENT=dev
PORT=5000
DATABASE_URL=
JWT_SECRET=<64 hex characters, generated once and kept>
ACME_CORS_ORIGINS=http://localhost:3000
```

and `.bos/env/web.env`:

```
PORT=3000
PUBLIC_API_URL=http://localhost:5000
PUBLIC_SITE_NAME="Acme"
```

## Configuration

`config.Load(prefix, defaults)` reads these variables. A field you leave empty in `Defaults` falls back to the framework's own value. The prefix is the string you pass to `Load`, so product-named variables stay the consumer's.

| Variable | Default | Meaning |
|---|---|---|
| `APP_NAME` | `Defaults.AppName` | reported by `/health` and `/` |
| `APP_VERSION` | `Defaults.AppVersion`, else `0.1.0` | reported by `/health` and `/` |
| `ENVIRONMENT` | `Defaults.Environment`, else `development` | `production` adds the HSTS header |
| `PORT` | `Defaults.Port`, else `8080` | the listening port |
| `DATABASE_URL` | `Defaults.DatabaseURL`, else empty | for the consumer's own wiring |
| `JWT_SECRET` | empty | for the consumer's own wiring |
| `BASE_URL` | `Defaults.BaseURL`, else empty | the public address of the site |
| `REDIS_URL` | empty | rate limiting and the session cache; empty means none |
| `<PREFIX>_CORS_ORIGINS` | `Defaults.CORSOrigins`, else `http://localhost:*` | comma-separated; `*` stops the process at startup |

## Wiring

`server.Deps` has four optional fields:

- `DB`: anything with `Ping(ctx) error`; a pgx pool works. Pass a nil interface, not a nil pool, for "no database"; `/health` then reports `disconnected`.
- `Auth`: when set, authentication extraction joins the chain.
- `API`: mounts your routes on `/api`, which already has the CSRF check.
- `Endpoints`: the list that `GET /` reports.

With no `API` function, the `/api` group answers unknown paths with not-found after the CSRF check.

## bos.yaml and the bos command

`bos version` prints `bos 0.0.0`. `bos env [-f bos.yaml] [-o .bos/env] [-p dev|bundle]` reads the file and writes into the folder: profile `dev` writes `api.env` and `web.env`, profile `bundle` writes the one `.env`. The keys of `bos.yaml`:

| Key | Default | Meaning |
|---|---|---|
| `prefix` | required | upper case letters, digits and underscores, starting with a letter; scopes the API's product-named variable (`<PREFIX>_CORS_ORIGINS`) |
| `site_name` | required | one line; the web's `PUBLIC_SITE_NAME` |
| `environment` | `dev` | the API's `ENVIRONMENT` |
| `api.port` | required, 1 to 65535 | the API's `PORT`, and the port in the web's `PUBLIC_API_URL`; in the bundle profile, the host port the API's container publishes on |
| `api.database_url` | empty | the API's `DATABASE_URL`; empty means no database |
| `web.port` | required, 1 to 65535 | the web's `PORT`, and the port in the API's allowed origin; in the bundle profile, the host port the web's container publishes on |

- The two ports must differ.
- An unknown key is refused, so a misspelt setting is not silently ignored.
- The API's allowed origin and the web's API address are derived from the two ports, and both assume `localhost`.
- `bos env` generates the JWT secret once (32 random bytes as 64 hex characters) and keeps it on later runs, reading it back from whichever file holds it in that profile. A file holding the secret is readable by its owner only.
- The `dev` profile's files are shell-sourceable (values with shell syntax are double-quoted). The `bundle` profile's `.env` is read by docker compose, which expands `$` inside double quotes but not inside single quotes, so such a value is single-quoted instead; a value that itself holds a single quote or a line break cannot be written this way, and is refused, naming the variable.
- Exit codes: 0 done; 1 the file or a setting was refused, and nothing is written; 2 a usage error, including an unknown profile.

## Tests

None yet: tests are added once the core is validated. `go build ./...` and `go vet ./...` are the checks.
