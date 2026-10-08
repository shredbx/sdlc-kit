<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { renderers } from '@sbx/bos-svelte/renderers';
	import type { SectionData } from '@sbx/bos-svelte/renderers';
	import type { CmsPage } from './types';

	let { slug, page, apiBase = '/api/admin/cms' }: {
		slug: string;
		page: CmsPage | null;
		apiBase?: string;
	} = $props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	// The cms API has no per-section endpoints (unlike the old flat pages model this replaces) —
	// Save writes title + body_markdown + published + the WHOLE content array in one PUT
	// (Repository.UpsertBySlug's field-presence contract). So sections are edited locally
	// (add/remove/reorder are plain array ops) and persisted only on Save. $state, not
	// $derived — this is an editable form seeded from the load, not a live view of it (same
	// pattern as every other admin form in this app); page can be null (a slug with no live
	// row yet — UpsertBySlug creates it on first Save).
	let title = $state(page?.title ?? '');
	let bodyMarkdown = $state(page?.body_markdown ?? '');
	let published = $state(page?.published ?? false);
	let sections: SectionData[] = $state((page?.published_content ?? []).map((s) => ({ ...s })));

	let saved = $state(false);
	let saveError = $state('');

	async function save() {
		saved = false;
		saveError = '';
		const res = await fetch(`${apiBase}/${slug}`, {
			method: 'PUT',
			headers: jsonHeaders,
			body: JSON.stringify({
				title,
				body_markdown: bodyMarkdown,
				published,
				content: sections
			})
		});
		if (!res.ok) {
			saveError = (await res.json())?.error ?? 'could not save';
			return;
		}
		saved = true;
		await invalidateAll();
	}

	let addOpen = $state(false);
	let pickedRendererId = $state(renderers[0]?.id ?? '');
	let fieldValues: Record<string, string> = $state({});

	const pickedRenderer = $derived(renderers.find((r) => r.id === pickedRendererId));

	function openAdd() {
		pickedRendererId = renderers[0]?.id ?? '';
		fieldValues = {};
		addOpen = true;
	}

	function addSection() {
		if (!pickedRenderer) return;
		sections = [...sections, { kind: pickedRenderer.id, [pickedRenderer.id]: { ...fieldValues } }];
		addOpen = false;
	}

	function removeSection(i: number) {
		sections = sections.filter((_, idx) => idx !== i);
	}

	function move(index: number, direction: -1 | 1) {
		const target = index + direction;
		if (target < 0 || target >= sections.length) return;
		const reordered = [...sections];
		[reordered[index], reordered[target]] = [reordered[target], reordered[index]];
		sections = reordered;
	}
</script>

<form onsubmit={(e) => (e.preventDefault(), save())}>
	<label>Title <input bind:value={title} /></label>
	<label>Body (markdown-only pages, no sections) <textarea rows="4" bind:value={bodyMarkdown}
		></textarea></label>
	<label class="checkbox"><input type="checkbox" bind:checked={published} /> Published</label>

	<section>
		<h2>Sections</h2>
		{#if sections.length === 0}
			<p>No sections yet.</p>
		{/if}
		<ul class="sections">
			{#each sections as section, i (i)}
				<li>
					<span class="kind">{section.kind}</span>
					<span class="preview">{JSON.stringify(section[section.kind])}</span>
					<span class="controls">
						<button type="button" onclick={() => move(i, -1)} disabled={i === 0} aria-label="Move up">↑</button>
						<button
							type="button"
							onclick={() => move(i, 1)}
							disabled={i === sections.length - 1}
							aria-label="Move down">↓</button
						>
						<button type="button" onclick={() => removeSection(i)}>Remove</button>
					</span>
				</li>
			{/each}
		</ul>
		<button type="button" onclick={openAdd}>Add section</button>
	</section>

	<button type="submit">Save</button>
	{#if saved}<span class="saved">Saved.</span>{/if}
	{#if saveError}<p class="error">{saveError}</p>{/if}
</form>

<Modal open={addOpen} title="Add section" onclose={() => (addOpen = false)}>
	<label>
		Kind
		<select bind:value={pickedRendererId}>
			{#each renderers as r (r.id)}
				<option value={r.id}>{r.label}</option>
			{/each}
		</select>
	</label>
	{#if pickedRenderer}
		{#each pickedRenderer.fields as field (field.key)}
			<label>
				{field.label}
				{#if field.type === 'markdown'}
					<textarea rows="4" bind:value={fieldValues[field.key]}></textarea>
				{:else}
					<input bind:value={fieldValues[field.key]} />
				{/if}
			</label>
		{/each}
	{/if}
	{#snippet footer()}
		<button onclick={() => (addOpen = false)}>Cancel</button>
		<button onclick={addSection}>Add</button>
	{/snippet}
</Modal>

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
	form label.checkbox {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}
	form label.checkbox input {
		width: auto;
		margin: 0;
	}
	.sections {
		list-style: none;
		padding: 0;
		margin: 0 0 1rem;
	}
	.sections li {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.5rem 0;
		border-bottom: 1px solid var(--color-border, #ddd);
	}
	.kind {
		font-weight: 600;
		text-transform: capitalize;
		min-width: 5rem;
	}
	.preview {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--color-text-muted, #666);
		font-size: 0.85rem;
	}
	.controls {
		display: flex;
		gap: 0.25rem;
	}
	.saved {
		margin-left: 0.5rem;
		color: var(--color-status-success, #2d8b57);
	}
	.error {
		color: var(--color-status-error, #c0392b);
	}
</style>
