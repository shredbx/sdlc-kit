<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { CmsPage } from './types';

	let { slug, page, apiBase = '/api/admin/cms' }: {
		slug: string;
		page: CmsPage | null;
		apiBase?: string;
	} = $props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	// Only the two length-validated overrides (seo.SeoMeta.Validate: meta_title <=60,
	// meta_description <=160) — og_*/canonical_url/noindex/keywords are real fields on the same
	// shared type but not exposed by this minimal pass (platform/CLAUDE.md "port only what the
	// app uses"; add them here when this demo actually needs them, same file, same shape).
	let metaTitle = $state(page?.seo_meta?.meta_title ?? '');
	let metaDescription = $state(page?.seo_meta?.meta_description ?? '');

	let saved = $state(false);
	let saveError = $state('');

	async function save() {
		saved = false;
		saveError = '';
		const res = await fetch(`${apiBase}/${slug}`, {
			method: 'PUT',
			headers: jsonHeaders,
			body: JSON.stringify({
				seo_meta: { meta_title: metaTitle, meta_description: metaDescription }
			})
		});
		if (!res.ok) {
			saveError = (await res.json())?.error ?? 'could not save';
			return;
		}
		saved = true;
		await invalidateAll();
	}
</script>

{#if !page}
	<p>Save the Content tab first — a page must exist before it can carry SEO overrides.</p>
{:else}
	<form onsubmit={(e) => (e.preventDefault(), save())}>
		<label>
			Meta title <span class="hint">(≤60 characters)</span>
			<input bind:value={metaTitle} maxlength="60" />
		</label>
		<label>
			Meta description <span class="hint">(≤160 characters)</span>
			<textarea rows="3" bind:value={metaDescription} maxlength="160"></textarea>
		</label>
		<button type="submit">Save</button>
		{#if saved}<span class="saved">Saved.</span>{/if}
		{#if saveError}<p class="error">{saveError}</p>{/if}
	</form>
{/if}

<style>
	form label {
		display: block;
		margin-bottom: 0.75rem;
	}
	form input,
	form textarea {
		display: block;
		width: 100%;
		margin-top: 0.25rem;
	}
	.hint {
		font-weight: normal;
		color: var(--color-text-muted, #888);
		font-size: 0.85rem;
	}
	.saved {
		margin-left: 0.5rem;
		color: var(--color-status-success, #2d8b57);
	}
	.error {
		color: var(--color-status-error, #c0392b);
	}
</style>
