<script lang="ts">
	import SiteHeader from '$lib/layout/SiteHeader.svelte';
	import SiteFooter from '$lib/layout/SiteFooter.svelte';
	import ContactDock from '$lib/layout/ContactDock.svelte';

	let { data, children } = $props();
	const locale = $derived(data.locale);
</script>

<SiteHeader {locale} />
<main id="main">
	{@render children()}
</main>
<SiteFooter {locale} />
<!-- Floating, config-driven; see ContactDock for why it is a slot rather than a button. -->
<ContactDock {locale} />
