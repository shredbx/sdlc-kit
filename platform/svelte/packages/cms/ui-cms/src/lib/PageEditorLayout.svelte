<script lang="ts">
	import { TabbedPageShell } from '@sbx/core-ui/components/layouts';
	import type { Snippet } from 'svelte';
	import type { CmsPage } from './types';

	let {
		slug,
		page,
		children,
		listPath = '/admin/pages',
		publicBasePath = ''
	}: {
		slug: string;
		page: CmsPage | null;
		children: Snippet;
		listPath?: string;
		publicBasePath?: string;
	} = $props();

	const base = $derived(`${listPath}/${slug}`);
	const tabs = $derived([
		{ id: 'content', label: 'Content', href: base },
		{ id: 'seo', label: 'SEO', href: `${base}/seo` },
		{ id: 'settings', label: 'Settings', href: `${base}/settings` }
	]);
	const title = $derived(page ? (page.title ?? slug) : `New page: ${slug}`);
</script>

<TabbedPageShell {title} back={{ href: listPath, label: 'Pages' }} {tabs}>
	{#snippet actions()}
		<a href="{publicBasePath}/{slug}" target="_blank" rel="noopener">View public page</a>
	{/snippet}
	{@render children()}
</TabbedPageShell>
