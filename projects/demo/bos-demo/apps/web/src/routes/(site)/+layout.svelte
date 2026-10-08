<script lang="ts">
	import { getHeaderPreset, getFooterPreset } from '@sbx/bos-svelte/chrome';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	// A page never touches chrome — it only resolves the app's default (docs/proposals/
	// bos-site-chrome.md §5). Falls open (renders nothing) if a preset id isn't registered,
	// same fails-open resolution order bos-constructor.md already uses for content.
	const header = $derived(data.settings.header.enabled ? getHeaderPreset(data.settings.header.preset) : undefined);
	const footer = $derived(data.settings.footer.enabled ? getFooterPreset(data.settings.footer.preset) : undefined);
</script>

{#if header}
	<header.component config={data.settings.header} siteTitle={data.settings.site_title} />
{/if}

{@render children()}

{#if footer}
	<footer.component config={data.settings.footer} />
{/if}
