<!--
PageTransition — route-level transition wrapper for SvelteKit.

Wraps the root {@render children()} slot in +layout.svelte. Uses SvelteKit's
onNavigate lifecycle + the browser's native View Transitions API
(document.startViewTransition) to animate route swaps via ::view-transition-old(root)
and ::view-transition-new(root) pseudo-elements on <html>.

Variants: fade (default), slide, blur, mask-wipe — defined in sibling
page-transitions.css and selected via [data-page-transition='...'] on <html>.

Usage (in +layout.svelte):
  <script>
    import { PageTransition } from '$lib/components/transitions';
    let { children } = $props();
  </script>

  <PageTransition type="fade">
    {@render children()}
  </PageTransition>

Per Decision #0212, the page type defaults to the brand's `transitions.page`
value (configured at the layout level by passing `type` explicitly).

Per Decision #0216, this is the rebuild of the original 153-LOC hand-rolled
version (PR #60) that motivated the evidence-before-implementation gate.
The native API does what the state machine reimplemented — see
research-plan.yml at .sbx/.runtime/tasks/2604-073/artifacts/ for the full
research trail (Svelte blog, MDN, Chrome devs 2025, Firefox 147 RN).

Browser support (April 2026): Baseline Newly Available — Chrome 111+,
Safari 18+, Firefox 144+ (~90.54% caniuse). Older browsers: feature
detection early-returns and navigation proceeds without animation.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onNavigate } from '$app/navigation';
	import './page-transitions.css';

	interface Props {
		type?: 'fade' | 'slide' | 'blur' | 'mask-wipe';
		/** Duration override in ms. Defaults to --duration-normal token. */
		duration?: number;
		children: Snippet;
	}

	let { type = 'fade', duration, children }: Props = $props();

	onNavigate((navigation) => {
		// SSR guard + feature detection (graceful degradation for old browsers).
		if (typeof document === 'undefined' || !document.startViewTransition) return;

		// Same-route navigation (hash change, reload) — no transition.
		if (!navigation.to || navigation.from?.url.pathname === navigation.to.url.pathname) {
			return;
		}

		// Set the variant on <html> so the sibling CSS pseudo-element selectors
		// pick the right keyframes. Optional duration override via CSS custom
		// property — consumed by --sbx-page-duration in the keyframe definitions.
		const root = document.documentElement;
		root.dataset.pageTransition = type;
		if (duration) root.style.setProperty('--sbx-page-duration', `${duration}ms`);

		return new Promise((resolve) => {
			document.startViewTransition(async () => {
				resolve();
				await navigation.complete;
			});
		});
	});
</script>

{@render children()}
