<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import type { AdminSettings } from './types';

	let { settings, apiBase = '/api/admin/settings/header' }: { settings: AdminSettings; apiBase?: string } =
		$props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	// Reads straight from settings.header rather than a local $state copy — a local copy
	// would only capture its initial value and go stale after invalidateAll().
	async function save(next: typeof settings.header) {
		await fetch(apiBase, {
			method: 'PATCH',
			headers: jsonHeaders,
			body: JSON.stringify(next)
		});
		await invalidateAll();
	}

	function toggleEnabled() {
		save({ ...settings.header, enabled: !settings.header.enabled });
	}

	let addOpen = $state(false);
	let title = $state('');
	let customPath = $state('');
	let webVisible = $state(true);
	let mobileVisible = $state(true);
	let newTab = $state(false);
	let addError = $state('');

	function openAdd() {
		title = '';
		customPath = '';
		webVisible = true;
		mobileVisible = true;
		newTab = false;
		addError = '';
		addOpen = true;
	}

	function addItem() {
		addError = '';
		if (!title || !customPath) {
			addError = 'Title and URL are required.';
			return;
		}
		const item = {
			type: 'link' as const,
			title,
			custom_path: customPath,
			enabled: true,
			web: webVisible,
			mobile: mobileVisible,
			new_tab: newTab
		};
		save({ ...settings.header, nav: [...settings.header.nav, item] });
		addOpen = false;
	}

	function removeItem(index: number) {
		save({ ...settings.header, nav: settings.header.nav.filter((_, i) => i !== index) });
	}

	function move(index: number, direction: -1 | 1) {
		const target = index + direction;
		if (target < 0 || target >= settings.header.nav.length) return;
		const reordered = [...settings.header.nav];
		[reordered[index], reordered[target]] = [reordered[target], reordered[index]];
		save({ ...settings.header, nav: reordered });
	}
</script>

<section>
	<label class="toggle">
		<input type="checkbox" checked={settings.header.enabled} onchange={toggleEnabled} />
		Show header
	</label>

	<ul class="nav-list">
		{#each settings.header.nav as item, i (item.title + i)}
			<li>
				<span class="title">{item.title}</span>
				<span class="path">{item.custom_path ?? '(page — not supported yet)'}</span>
				<span class="controls">
					<button onclick={() => move(i, -1)} disabled={i === 0} aria-label="Move up">↑</button>
					<button
						onclick={() => move(i, 1)}
						disabled={i === settings.header.nav.length - 1}
						aria-label="Move down">↓</button
					>
					<button onclick={() => removeItem(i)}>Remove</button>
				</span>
			</li>
		{/each}
		{#if settings.header.nav.length === 0}
			<li class="empty">No nav items yet.</li>
		{/if}
	</ul>
	<button onclick={openAdd}>Add nav item</button>
</section>

<Modal open={addOpen} title="Add nav item" onclose={() => (addOpen = false)}>
	<label>Title <input bind:value={title} /></label>
	<label>URL <input bind:value={customPath} placeholder="/about" /></label>
	<label><input type="checkbox" bind:checked={webVisible} /> Show on web</label>
	<label><input type="checkbox" bind:checked={mobileVisible} /> Show on mobile</label>
	<label><input type="checkbox" bind:checked={newTab} /> Open in new tab</label>
	{#if addError}<p class="error">{addError}</p>{/if}
	{#snippet footer()}
		<button onclick={() => (addOpen = false)}>Cancel</button>
		<button onclick={addItem}>Add</button>
	{/snippet}
</Modal>

<style>
	.toggle {
		display: block;
		margin-bottom: 1rem;
	}
	.nav-list {
		list-style: none;
		padding: 0;
		margin: 0 0 1rem;
	}
	.nav-list li {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.5rem 0;
		border-bottom: 1px solid var(--color-border, #ddd);
	}
	.empty {
		color: var(--color-text-muted, #888);
	}
	.title {
		font-weight: 600;
		min-width: 8rem;
	}
	.path {
		flex: 1;
		color: var(--color-text-muted, #666);
		font-size: 0.85rem;
	}
	.controls {
		display: flex;
		gap: 0.25rem;
	}
	label {
		display: block;
		margin-bottom: 0.75rem;
	}
	input:not([type='checkbox']) {
		display: block;
		width: 100%;
		margin-top: 0.25rem;
	}
	input[type='checkbox'] {
		margin-right: 0.4rem;
	}
	.error {
		color: var(--color-status-error, #c0392b);
	}
</style>
