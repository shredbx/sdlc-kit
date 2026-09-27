<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { layouts } from '@sbx/bos-svelte/layouts';
	import type { CmsPage } from './types';

	let {
		slug,
		page,
		apiBase = '/api/admin/cms'
	}: {
		slug: string;
		page: CmsPage | null;
		apiBase?: string;
	} = $props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	let layout = $state(page?.layout ?? 'default');

	let saved = $state(false);
	let saveError = $state('');

	async function save() {
		saved = false;
		saveError = '';
		const res = await fetch(`${apiBase}/${slug}`, {
			method: 'PUT',
			headers: jsonHeaders,
			body: JSON.stringify({ layout })
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
	<p>Save the Content tab first — a page must exist before it can carry a layout.</p>
{:else}
	<form onsubmit={(e) => (e.preventDefault(), save())}>
		<label>
			Layout
			<select bind:value={layout}>
				{#each layouts as l (l.id)}
					<option value={l.id}>{l.label}</option>
				{/each}
			</select>
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
	form select {
		display: block;
		width: 100%;
		margin-top: 0.25rem;
	}
	.saved {
		margin-left: 0.5rem;
		color: var(--color-status-success, #2d8b57);
	}
	.error {
		color: var(--color-status-error, #c0392b);
	}
</style>
