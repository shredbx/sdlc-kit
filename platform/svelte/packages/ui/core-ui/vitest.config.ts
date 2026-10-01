import { fileURLToPath } from 'node:url';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig } from 'vitest/config';

// Vitest scope — co-located unit tests under src/.
//
// The svelte() plugin (the plain @sveltejs/vite-plugin-svelte, NOT sveltekit() —
// core-ui is a library package, not a SvelteKit app) transforms `.svelte` files so
// a unit test can import a real component and mount it. Required once a test imports
// a Svelte component directly — e.g. layouts/TabbedPageShell.test.ts mounts
// TabbedPageShell.svelte (#0290 F2). Pure node-logic tests (track, branding, money,
// navigation rules, …) are unaffected — they import no `.svelte` and keep the default
// node environment. The component test scopes jsdom to itself via a per-file
// `// @vitest-environment jsdom` docblock (the same mechanism BR's web-svelte uses),
// so jsdom never touches the logic tests.
export default defineConfig({
	plugins: [svelte()],
	test: {
		include: ['src/**/*.test.ts']
	},
	// Component tests mount real Svelte components via Svelte's `mount`. Forcing the
	// `browser` package entry points ONLY while Vitest runs (process.env.VITEST) gives
	// the client build for tests (where `mount` works) without altering any other build.
	// The DOM itself comes from the per-file `// @vitest-environment jsdom` docblock.
	// Mirrors the proven BR web-svelte vitest config.
	//
	// `$app/*` aliases: core-ui is a PLAIN svelte library (svelte() plugin, not
	// sveltekit()), so SvelteKit's `$app/state` / `$app/navigation` virtual modules
	// don't exist here. A component that imports them (TabbedPageShell reads
	// `$app/state`'s page + `$app/navigation`'s before/afterNavigate) would fail vite
	// import-analysis. The aliases resolve those specifiers to local stubs so the
	// component transforms; each component test then `vi.mock`s them for real behaviour.
	// Scoped to VITEST so no real build (or svelte-check) ever sees them.
	resolve: process.env.VITEST
		? {
				conditions: ['browser'],
				alias: {
					'$app/state': fileURLToPath(
						new URL('./src/lib/test-support/app-stubs/state.ts', import.meta.url)
					),
					'$app/navigation': fileURLToPath(
						new URL('./src/lib/test-support/app-stubs/navigation.ts', import.meta.url)
					)
				}
			}
		: undefined
});
