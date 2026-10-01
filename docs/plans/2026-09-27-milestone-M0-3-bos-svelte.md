# Milestone M0.3 (minimal) — `bos-svelte`, and the consumer's example app running end to end

Status: done. The code is gated, and the unit record and its rendered README were approved and written on 2026-09-27
(`processos-workspace/records/sbx-sdlc-kit/modeling/package/bos-svelte/{package.yaml, readme.yaml}`; the README is byte-identical to the render). Decisions applied: D24 (names and local links), D25 (moved and
configured, not rewritten), D26 (no tests until the core is validated). Roadmap: `docs/proposals/bos-app-roadmap.md`, M0.

**Scope.** Only what is needed to run the first homepage that talks to the running API, with the settings coming from one place: the web
framework, the consumer's web app, and `make dev`. The docker bundle, the sdlc-kit action and process for app bundles, `bos.yaml` and the `bos`
command stay in the rest of M0.3.

## What exists

```
platform/svelte/frameworks/bos-svelte/           package @sbx/bos-svelte
├── package.json                                 exports "." and "./server"; peers @sveltejs/kit ^2 and svelte ^5; no runtime dependencies
└── src/lib/
    ├── server/security-headers.ts               MOVED  applySecurityHeaders, isHttpsRequest, securityHeadersHandle
    ├── server/api-pass-through.ts               MOVED  the same-origin /api pass-through (without the silent refresh)
    ├── server/index.ts                          NEW    createHandle({ apiUrl }) = sequence(securityHeadersHandle, apiPassThroughHandle)
    ├── health.ts                                NEW    loadApiHealth(apiUrl): healthy | unreachable, never throws
    ├── PlaceholderHome.svelte                   NEW    "<site name>: it works." and the API's status and self-report
    └── index.ts                                 NEW    public exports
```

The consumer's side (in its own repository): `pnpm-workspace.yaml` listing `apps/web` and the framework folder by local directory, `apps/web`
(a five-line `hooks.server.ts`, a page `load`, a page, the adapter and Vite configuration), and the example Makefiles.

## What was moved, and what was not

The code is the running web app's server hook. Compared with the original, comments and whitespace aside:

| Piece | Against the original |
|---|---|
| `applySecurityHeaders`, the header set, `isHttpsRequest`, `securityHeadersHandle` | identical, apart from an `export` on two names and one type import; comments naming the product's hosts and embeds were removed |
| The `/api` pass-through: cookies and body forwarded, hop-by-hop headers stripped, `x-forwarded-for` set (never appended) from `getClientAddress()`, the answer returned as it is | identical; the API address is a parameter instead of a module constant; the retry-after-refresh branch is not moved |

Left in the app until the auth kit moves: the silent refresh on a 401, the auth cookies and their lifetimes, the navigation-time user lookup, the
preview password guard, and media initialisation.

New code, because nothing runs today: `createHandle`, `loadApiHealth`, `PlaceholderHome`. The settings are environment variables written by the
consumer's `make env` (the API address, the port, the site name).

## The one network action

`@sveltejs/kit` and `@sveltejs/adapter-node` were not in this machine's pnpm store or its metadata cache, so the web app could not be installed
offline. With the user's approval (2026-09-27) a **single install from the npm registry** fetched them, pinned to the versions the running web
app uses (kit 2.50.2, adapter-node 5.5.2): 74 packages, of which 22 were downloaded and 52 came from the store, with lifecycle scripts off. It
wrote the consumer's `pnpm-lock.yaml`. Every install, build and run after that is offline.

## Gates and evidence

| Gate | Result |
|---|---|
| A clean reinstall from the lockfile with the network denied (`--offline --frozen-lockfile`) | 74 reused, 0 downloaded |
| `svelte-kit sync` and `svelte-check`, then `vite build` (adapter-node), both with the network denied | 0 errors, 0 warnings; built |
| `make dev` (the settings, the API and the web together) under a sandbox that denies all outbound network: the homepage, the API status, the pass-through | see below |
| A scan of `bos-svelte`, its record and this plan for the product's names and the consumer's ports | no match (the first scan looked at names only and missed a port in a comment; it is now the generic example port) |
| `process-cli validate` of both records, `render`, `conform`, and a byte compare of the installed README | ok; identical |
| `process-cli check`, and `platform/tools/check_kit_edges.py` | ok; 0 errors (the framework declares no in-repo dependency) |
| The original project's `git status` | unchanged |

**Run under the network-denying sandbox: 27 of 27 checks.**

- **Homepage.** `GET /` is 200; the heading carries the site name from the settings; the page says `API: healthy` and shows the name, version,
  environment and database the API reported about itself; the title is the site name; the page carries the security headers (and no HSTS over
  plain http); an unknown page is the web's own 404 with the same headers.
- **Pass-through.** `GET /api/x` reaches the API and its plain-text 404 comes back with the API's own headers untouched; a `POST` without
  `X-Requested-With` is refused by the API (403); with the header it reaches routing (404). Against a header-echo server standing in for the API:
  method, path, query and body arrive intact, the cookie is forwarded, the host header is the API's, `X-Forwarded-For` is the real client address
  and a spoofed one is replaced, and `Set-Cookie` comes back.
- **API down.** The page still renders (200) and says `API: unreachable`; a request to `/api` with nothing behind it is a server error, not a hang.

A finding, not fixed here: the pass-through has no timeout of its own, as in the original, so an API that accepts a connection and never answers
holds the request open. It came up while writing the probe (a leaked socket in the probe did exactly that).

## Not in this scope

No tests (D26), no brand, tokens, pages or presets (M1 onward), no `bos-web` command, no bundle. The framework is not yet a member of the
`platform/svelte` workspace; the consumer's workspace links it by local directory, which is what D24 requires.
