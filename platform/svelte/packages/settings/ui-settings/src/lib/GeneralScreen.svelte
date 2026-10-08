<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import type { AdminSettings } from './types';

	let { settings, apiBase = '/api/admin/settings' }: { settings: AdminSettings; apiBase?: string } = $props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	let siteTitle = $state(settings.site_title);
	let tagline = $state(settings.tagline);
	let saved = $state(false);

	async function saveSettings() {
		saved = false;
		await fetch(apiBase, {
			method: 'PATCH',
			headers: jsonHeaders,
			body: JSON.stringify({ site_title: siteTitle, tagline })
		});
		saved = true;
		await invalidateAll();
	}
</script>

<form onsubmit={(e) => (e.preventDefault(), saveSettings())}>
	<label>Site title <input bind:value={siteTitle} required /></label>
	<label>Tagline <input bind:value={tagline} /></label>
	<button type="submit">Save</button>
	{#if saved}<span class="saved">Saved.</span>{/if}
</form>

<style>
	form label {
		display: block;
		margin-bottom: 0.75rem;
	}
	form input {
		display: block;
		width: 100%;
		margin-top: 0.25rem;
	}
	.saved {
		margin-left: 0.5rem;
		color: var(--color-status-success, #2d8b57);
	}
</style>
