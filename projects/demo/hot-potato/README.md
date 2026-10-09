# Hot Potato app-framework demo

This demo is a copy of the pinned Hot Potato SvelteKit app, adapted to exercise the Svelte app
framework. The source submodule remains unchanged.

## Run

From this directory:

```sh
pnpm install
make dev
```

The app uses the Node adapter and request-time server loading. Checked-in home-page content is
validated with a Zod schema, read through the framework's asynchronous read-only repository, and
registered per request in a SvelteKit server hook. The route loader passes the result to the
presentational `HomePage` component.

## Checks

```sh
pnpm test
pnpm check
pnpm build
```
