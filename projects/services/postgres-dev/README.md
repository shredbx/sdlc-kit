# postgres-dev

Postgres + pgAdmin for local development — rendered from
`processos-workspace/records/sbx-sdlc-kit/infrastructure/service-bundle/postgres-dev.yaml` by
`sbx-sdlc-kit.infrastructure.bootstrap-bundle`. Real secrets live in `.env`, next to this file —
gitignored, never committed.

## Stop/start the bundle

From this folder:

    make stop
    make start

Or directly:

    cd projects/services/postgres-dev
    docker compose -p postgres-dev down    # stop
    docker compose -p postgres-dev up -d   # start

Or re-render and re-run the whole pipeline from its own records (from the repo root):

    process-cli run sbx-sdlc-kit.infrastructure.bootstrap-bundle \
      --records bundle=processos-workspace/records/sbx-sdlc-kit/infrastructure/service-bundle/postgres-dev.yaml

## Connect

- Postgres: `localhost:54321`, user/db `postgres`, password in `.env`'s `POSTGRES_PASSWORD`
- pgAdmin: http://localhost:5050, login from `.env`'s `PGADMIN_DEFAULT_EMAIL`/`PGADMIN_DEFAULT_PASSWORD`

This file and the Makefile are hand-written for this bundle only. Formalizing them into
`bundle-compose`'s own rendered output (so every future bundle gets them for free) is a real,
deliberately deferred next step — not done as part of this milestone.
