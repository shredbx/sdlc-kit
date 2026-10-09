<script lang="ts">
	// Existing policy body, carried into the new shell unchanged.
	import CookiePolicy from '$lib/CookiePolicy.svelte';

	let { data } = $props();
	const locale = $derived(data.locale);
</script>

<CookiePolicy lang={locale} />
