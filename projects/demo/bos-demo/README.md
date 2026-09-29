# bos-demo

The `bos` product's own reference build — a generic, unbranded consumer of `bos-go`/`bos-svelte`
proving the constructor's admin/content loop, before any real client is configured on top of it.
Not a client consumer (see `docs/proposals/bos-demo-mvp.md`): it lives here, in sdlc-kit itself,
not as a submodule under `consumers/`.

## Datastore (Postgres + Redis)

Rendered from `processos-workspace/records/sbx-sdlc-kit/infrastructure/service-bundle/bos-dev.yaml`
by `sbx-sdlc-kit.infrastructure.bootstrap-bundle`. Real secrets live in `deploy/.env`, gitignored,
never committed. Data lives in `data/` (also gitignored), so it survives a `down` + `up`.

    make start
    make stop

Or directly:

    cd projects/demo/bos-demo/deploy
    docker compose -p bos-dev down    # stop
    docker compose -p bos-dev up -d   # start

## Connect

- Postgres: `localhost:54325`, user/db `postgres`, password in `deploy/.env`'s `POSTGRES_PASSWORD`
- Redis: `localhost:6325`

## The app (bos-go + bos-svelte)

`bos.yaml`, `apps/api`, `apps/web` — see the root `Makefile` for `make dev`.
