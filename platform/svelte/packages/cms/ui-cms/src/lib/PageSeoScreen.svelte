<script lang="ts">
	import { page } from '$app/state';
	import { invalidateAll } from '$app/navigation';
	import { SeoEditor, toSeoDraft, fromSeoDraft } from '@sbx/ui-seo';
	import type { SeoDefaults, SeoContext } from '@sbx/ui-seo/types';
	import type { CmsPage } from './types';

	let {
		slug,
		page: cmsPage,
		apiBase = '/api/admin/cms',
		siteName = 'this site'
	}: {
		slug: string;
		page: CmsPage | null;
		apiBase?: string;
		siteName?: string;
	} = $props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	// The full shared seo.SeoMeta shape (og_*/canonical_url/noindex), not just the two
	// length-validated primaries the earlier hand-rolled form exposed — @sbx/ui-seo's
	// SeoEditor + resolveSeo are the same code the public SeoHead renders from, so this
	// editor's live previews can never drift from the real <head>.
	let draft = $state(toSeoDraft(cmsPage?.seo_meta));

	const defaults: SeoDefaults = $derived({
		title: cmsPage?.title ?? slug,
		description: '',
		siteName,
		defaultImage: ''
	});
	const context: SeoContext = $derived({ pathname: page.url.pathname, origin: page.url.origin });

	let saved = $state(false);
	let saveError = $state('');

	async function save() {
		saved = false;
		saveError = '';
		const res = await fetch(`${apiBase}/${slug}`, {
			method: 'PUT',
			headers: jsonHeaders,
			body: JSON.stringify({ seo_meta: fromSeoDraft(draft) ?? {} })
		});
		if (!res.ok) {
			saveError = (await res.json())?.error ?? 'could not save';
			return;
		}
		saved = true;
		await invalidateAll();
	}
</script>

{#if !cmsPage}
	<p>Save the Content tab first — a page must exist before it can carry SEO overrides.</p>
{:else}
	<form onsubmit={(e) => (e.preventDefault(), save())}>
		<SeoEditor bind:draft {defaults} {context} />
		<button type="submit">Save</button>
		{#if saved}<span class="saved">Saved.</span>{/if}
		{#if saveError}<p class="error">{saveError}</p>{/if}
	</form>
{/if}

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}
	.saved {
		margin-left: 0.5rem;
		color: var(--color-status-success, #2d8b57);
	}
	.error {
		color: var(--color-status-error, #c0392b);
	}
</style>
