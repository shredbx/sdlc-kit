<script lang="ts">
	import { getLayout } from '@sbx/bos-svelte/layouts';
	import { SeoHead } from '@sbx/ui-seo';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const layout = $derived(data.found ? getLayout(data.page.layout) : undefined);
</script>

<SeoHead
	title={data.found ? (data.page.title ?? 'Home') : data.siteName}
	description={data.found ? (data.page.body_markdown ?? '') : ''}
	seoMeta={data.found ? data.page.seo_meta : undefined}
	siteName={data.settings.site_title}
	defaultImage=""
/>

{#if data.found && layout}
	<layout.component sections={data.page.published_content ?? []} />
{:else}
	<main class="page">
		<h1>{data.siteName}</h1>
		<p>No published "home" page yet — create one in <a href="/admin">/admin</a>.</p>
	</main>
{/if}

<style>
	.page {
		max-width: 40rem;
		margin: 0 auto;
		padding: 3rem 1rem;
		font-family: sans-serif;
	}
</style>
