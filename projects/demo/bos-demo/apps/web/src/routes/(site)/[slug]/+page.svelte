<script lang="ts">
	import { getLayout } from '@sbx/bos-svelte/layouts';
	import { SeoHead } from '@sbx/ui-seo';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	// Resolved via the layout registry instead of a hardcoded <h1>+SectionList — the
	// page's own title is <svelte:head> only now; content declares its headline
	// through a block (bos-layout-presets.md §3: "code declares, data selects").
	const layout = $derived(getLayout(data.page.layout));
</script>

<SeoHead
	title={data.page.title ?? data.page.slug}
	description={data.page.body_markdown ?? ''}
	seoMeta={data.page.seo_meta}
	siteName={data.settings.site_title}
	defaultImage=""
/>

<layout.component sections={data.page.published_content ?? []} />
