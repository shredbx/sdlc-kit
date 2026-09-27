// Vitest resolution stub for SvelteKit's `$app/state` virtual module.
//
// core-ui is a PLAIN svelte library (svelte() plugin, not sveltekit()), so the
// `$app/*` virtual modules SvelteKit normally injects do not exist here. A component
// that imports `$app/state` (e.g. layouts/TabbedPageShell.svelte reads
// `page.url.pathname`) therefore fails vite's import-analysis under the plain plugin.
//
// This stub gives vite something to RESOLVE; it is NOT the value a test sees at
// runtime — every component test that mounts such a component `vi.mock('$app/state',
// …)` with its own mutable `page`, which overrides this entirely. The shape here only
// has to type-check and export the right names. Wired via `resolve.alias` in
// vitest.config.ts (VITEST-only — it never affects any real build).

export const page = { url: { pathname: '/' } };
