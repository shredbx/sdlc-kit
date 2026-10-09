<script lang="ts">
	// /contacts — direct channels only. No form, no signup, no third party (owner ruling
	// 2026-08-24). Zero backend and zero spam surface, the same way the portfolio already
	// asks people to get in touch.
	import ContactBlock from '$lib/sections/ContactBlock.svelte';
	import Seo from '$lib/sections/Seo.svelte';

	let { data } = $props();
	const locale = $derived(data.locale);
</script>

<Seo
	{locale}
	titleKey="contacts.meta.title"
	descriptionKey="contacts.meta.description"
	path="contacts"
/>

<ContactBlock {locale} variant="full" titleKey="contacts.title" bodyKey="contacts.lead" />
