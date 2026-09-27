<script lang="ts">
	import { getLayout } from '@sbx/bos-svelte/layouts';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	// Resolved via the layout registry instead of a hardcoded <h1>+SectionList — the
	// page's own title is <svelte:head> only now; content declares its headline
	// through a block (bos-layout-presets.md §3: "code declares, data selects").
	const layout = $derived(getLayout(data.page.layout));
</script>

<svelte:head>
	<title>{data.page.title}</title>
</svelte:head>

<layout.component sections={data.page.published_content ?? []} />
