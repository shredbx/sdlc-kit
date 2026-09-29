<script lang="ts">
	import { TabbedPageShell } from '@sbx/core-ui/components/layouts';
	import type { Snippet } from 'svelte';

	let { children, basePath = '/admin/settings' }: { children: Snippet; basePath?: string } = $props();

	const tabs = [
		{ id: 'general', label: 'General', href: basePath },
		{ id: 'header', label: 'Header', href: `${basePath}/header` },
		{ id: 'footer', label: 'Footer', href: `${basePath}/footer` }
	];
</script>

<TabbedPageShell title="Settings" {tabs}>
	{@render children()}
</TabbedPageShell>
