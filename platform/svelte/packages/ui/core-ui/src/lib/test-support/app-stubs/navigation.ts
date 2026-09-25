// Vitest resolution stub for SvelteKit's `$app/navigation` virtual module.
//
// See ./state.ts for the full rationale: core-ui is a plain svelte library, so the
// `$app/*` virtual modules don't exist under the svelte() plugin and a component that
// imports them fails vite's import-analysis. This stub exists only so vite can RESOLVE
// the specifier; component tests `vi.mock('$app/navigation', …)` to capture the
// afterNavigate/beforeNavigate callbacks, overriding these no-ops. Wired via
// `resolve.alias` in vitest.config.ts (VITEST-only — never affects any real build).

export function afterNavigate(_cb: (nav?: unknown) => void): void {}
export function beforeNavigate(_cb: (nav?: unknown) => void): void {}
