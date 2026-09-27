<script lang="ts">
	import { PageEditorLayout } from '@sbx/ui-cms';
	import type { Snippet } from 'svelte';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();
</script>

<PageEditorLayout slug={data.slug} page={data.page}>
	{@render children()}
</PageEditorLayout>
