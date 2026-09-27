# Milestone M0.3e — the app bundle, definitions 3 to 5: a bundle says where it lives, `verify-link`, `bootstrap-app-bundle`

Status: done. The user asked (2026-09-27) to proceed through the app bundle until the example app runs, so these were built and gated in one run, but each
is its own definition and its own commit. Decisions applied: D24 (a consumer's bundle lives in the consumer's repository, nothing fetched), D25 (the existing
actions are extended, not duplicated), D26 (the probes are throwaway scripts; no test code enters the repository).

**Why the existing actions had to change.** `render-bundle` and `run-bundle` write to and run from `projects/services/<name>` of the repository the actions
live in. Run from a consumer, that would put the consumer's compose files into sdlc-kit. A bundle now says where it lives.

## What exists

```
processos-workspace/definitions/sbx-sdlc-kit/infrastructure/
├── schema/service-bundle.yaml                 MODIFIED  + into (optional): a folder relative to the root of the repository the run works in
├── action/render-bundle/{action.sh, assets/lib.sh}          MODIFIED  the target comes from bundle_dir
├── action/run-bundle/{action.sh, check.sh, post.sh, action.yaml, assets/lib.sh}   MODIFIED, lib.sh NEW
│                                                            bundle_dir; `up -d --build`; post.sh checks the ports docker compose resolved
├── action/verify-link/{action.yaml, action.sh, assets/lib.sh}   NEW
└── process/bootstrap-app-bundle.yaml          NEW       render-bundle -> run-bundle -> verify-link
```

- **`into`.** Left out, nothing changes: `projects/services/<name>` in the repository the actions live in. Given, the folder is relative to the repository that
  holds the run's output folder, so a consumer's bundle lands in the consumer. An `into` that is absolute or contains `..` is refused before anything is written.
  `bundle_dir` is the same text in the three actions (a test compares them); the actions cannot share a file, so it is repeated.
- **`run-bundle`** now passes `--build` (a no-op for a service that pulls an image, so `postgres-dev` behaves as before) and its post step reads the host ports
  from `docker compose config`, which resolves a port written `${VAR}:8080` through the bundle's `.env`. Before, it parsed the records' `ports` text and would have
  tried to connect to a port named `${VAR}`.
- **`verify-link`** reads the published ports of the services named `bos-api` and `bos-web` the same way, waits up to 90 seconds for the API to answer
  `/health` with `"status": "healthy"`, then up to 60 seconds for the web page to say `API: healthy`. It reports the API's own service, version and database.
  A page that says anything else fails with what it said and "the two halves are not linked". The page text is the placeholder home's contract; it changes with the pages.
- **`bootstrap-app-bundle`** is `bootstrap-bundle` plus the link check; `process-cli check` proves its inputs and handoffs.
- **`.env.example`** (template file `.env.example.jinja`, a small follow-up): it now also names the variables written inside a service's `ports`
  (`${API_PORT}:8080` gives `API_PORT=`), ahead of that service's `env` names. Before, the example listed only `env` names, so a bundle whose ports come from its
  `.env` would have had an example that leaves the ports out. Records with literal ports render exactly as before.

## Gates and evidence

| Gate | Result |
|---|---|
| `process-cli check` (which checks the new process end to end) | ok |
| `bundle_dir` in each of the three actions: default folder; `into` relative to the repository of the run; a run inside the consumer repository gives the consumer's root; a name that only starts with dots is fine; the six refused forms (`../x`, `/abs`, `a/../b`, `..`, `a/..`, `//x`) | all pass, and the three texts are identical |
| Published ports from a scratch compose file and `.env`, without the docker daemon: `${VAR}` ports, a literal port, a service with none, a service that is not in the bundle | all as expected |
| `render-bundle` with no `into`, run for real on `postgres-dev` | ok; `git status` shows no change under `projects/services/` |
| `render-bundle` with `into` in a scratch folder, and with `into: ../escape` | files land only there; the escape is refused and nothing is written |
| `verify-link` against fake servers standing in for the two halves: both good; API healthy but the page says `API: unreachable`; an API that never reports healthy; a bundle with no `bos-web` | done and reports the API's details; failed naming the page's text; failed after the poll; refused at once |
| The 32 checks above, in one throwaway script | 32 of 32 |
| `.env.example`: `postgres-dev` byte-identical to the committed file; `${VAR}` ports named ahead of the service's `env` names, a literal port adding nothing, two variables in one port both named | both pass |
| A scan of the definitions and this plan for the product's names and ports | no match |

No container was started by these gates. The real run is in the next scope (M0.3f), once the images can be built.

## Not in this scope

The Dockerfiles, the `bos env` profile for the bundle, the service records of the two app halves, the consumer's bundle record and Makefile targets, and the
first real `make bundle-up`.
