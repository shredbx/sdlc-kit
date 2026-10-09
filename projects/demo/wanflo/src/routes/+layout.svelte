<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { initTheme } from '$lib/stores/theme';
	import Analytics from '$lib/Analytics.svelte';
	import CookieConsent from '$lib/CookieConsent.svelte';
	let { children } = $props();

	// The inline app.html script sets the first-paint theme (no FOUC); initTheme() takes
	// ownership after hydration — wires the toggle store, cross-tab sync, and the DOM guard.
	onMount(() => {
		initTheme();
	});
</script>

{@render children()}

<!-- Analytics is consent-gated (nothing fires until Accept); the banner sits above content. -->
<Analytics />
<CookieConsent />
