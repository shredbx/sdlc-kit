/**
 * Transitions — mount/exit/route-level motion components.
 *
 * Decision #0212: three architecturally distinct families.
 *   PageTransition — route-level onNavigate hook (SvelteKit global).
 *   RevealGroup    — section-level preset override via Svelte context.
 *   use-reveal     — Svelte action for IntersectionObserver reveals.
 */

export { default as PageTransition } from './PageTransition.svelte';
export { default as RevealGroup } from './RevealGroup.svelte';
export { reveal } from './use-reveal.js';
export type { RevealParams } from './use-reveal.js';
