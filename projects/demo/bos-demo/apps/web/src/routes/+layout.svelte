<script lang="ts">
	// The one place global styles load — wraps every route, (site) and admin alike.
	// bos-brand.css supplies the token values; brand-bridge.css maps them onto the
	// component-facing token names @sbx/core-ui's components actually read (see its
	// own doc comment for the cascade order). data-brand="bos"/data-theme is set
	// statically in app.html (SSR, no client JS needed) — see
	// docs/proposals/bos-admin-modules.md §5 for why this stays a plain CSS default
	// rather than going through core-ui's IStylePreset/styleStore switcher: that
	// store auto-initializes to its own hardcoded default preset ('hub-dark') on
	// every client hydration, which would silently override this brand the moment
	// any code imports it — not worth wiring until bos-demo actually needs runtime
	// brand/mode switching.
	import '@sbx/bos-svelte/theme/bos-brand.css';
	import '@sbx/core-ui/style/brand-bridge.css';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

{@render children()}
