<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import type { AdminSettings } from './types';

	let { settings, apiBase = '/api/admin/settings/footer' }: { settings: AdminSettings; apiBase?: string } =
		$props();

	const jsonHeaders = { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' };

	async function save(next: typeof settings.footer) {
		await fetch(apiBase, {
			method: 'PATCH',
			headers: jsonHeaders,
			body: JSON.stringify(next)
		});
		await invalidateAll();
	}

	function toggleEnabled() {
		save({ ...settings.footer, enabled: !settings.footer.enabled });
	}

	let addSectionOpen = $state(false);
	let sectionHeading = $state('');

	function openAddSection() {
		sectionHeading = '';
		addSectionOpen = true;
	}

	function addSection() {
		if (!sectionHeading) return;
		save({
			...settings.footer,
			sections: [...settings.footer.sections, { heading: sectionHeading, links: [] }]
		});
		addSectionOpen = false;
	}

	function removeSection(index: number) {
		save({ ...settings.footer, sections: settings.footer.sections.filter((_, i) => i !== index) });
	}

	let addLinkOpen = $state(false);
	let addLinkSectionIndex = $state(-1);
	let linkTitle = $state('');
	let linkPath = $state('');
	let linkWeb = $state(true);
	let linkMobile = $state(true);
	let addLinkError = $state('');

	function openAddLink(index: number) {
		addLinkSectionIndex = index;
		linkTitle = '';
		linkPath = '';
		linkWeb = true;
		linkMobile = true;
		addLinkError = '';
		addLinkOpen = true;
	}

	function addLink() {
		addLinkError = '';
		if (!linkTitle || !linkPath) {
			addLinkError = 'Title and URL are required.';
			return;
		}
		const sections = settings.footer.sections.map((sec, i) =>
			i === addLinkSectionIndex
				? {
						...sec,
						links: [
							...sec.links,
							{ title: linkTitle, custom_path: linkPath, enabled: true, web: linkWeb, mobile: linkMobile }
						]
					}
				: sec
		);
		save({ ...settings.footer, sections });
		addLinkOpen = false;
	}

	function removeLink(sectionIndex: number, linkIndex: number) {
		const sections = settings.footer.sections.map((sec, i) =>
			i === sectionIndex ? { ...sec, links: sec.links.filter((_, li) => li !== linkIndex) } : sec
		);
		save({ ...settings.footer, sections });
	}
</script>

<section>
	<label class="toggle">
		<input type="checkbox" checked={settings.footer.enabled} onchange={toggleEnabled} />
		Show footer
	</label>

	{#each settings.footer.sections as sec, si (sec.heading + si)}
		<div class="footer-section">
			<div class="section-header">
				<h3>{sec.heading}</h3>
				<button onclick={() => removeSection(si)}>Remove section</button>
			</div>
			<ul>
				{#each sec.links as link, li (link.title + li)}
					<li>
						<span class="title">{link.title}</span>
						<span class="path">{link.custom_path ?? '(page — not supported yet)'}</span>
						<button onclick={() => removeLink(si, li)}>Remove</button>
					</li>
				{/each}
				{#if sec.links.length === 0}
					<li class="empty">No links yet.</li>
				{/if}
			</ul>
			<button onclick={() => openAddLink(si)}>Add link</button>
		</div>
	{/each}
	{#if settings.footer.sections.length === 0}
		<p class="empty">No footer sections yet.</p>
	{/if}
	<button onclick={openAddSection}>Add section</button>
</section>

<Modal open={addSectionOpen} title="Add footer section" onclose={() => (addSectionOpen = false)}>
	<label>Heading <input bind:value={sectionHeading} /></label>
	{#snippet footer()}
		<button onclick={() => (addSectionOpen = false)}>Cancel</button>
		<button onclick={addSection}>Add</button>
	{/snippet}
</Modal>

<Modal open={addLinkOpen} title="Add link" onclose={() => (addLinkOpen = false)}>
	<label>Title <input bind:value={linkTitle} /></label>
	<label>URL <input bind:value={linkPath} placeholder="/about" /></label>
	<label><input type="checkbox" bind:checked={linkWeb} /> Show on web</label>
	<label><input type="checkbox" bind:checked={linkMobile} /> Show on mobile</label>
	{#if addLinkError}<p class="error">{addLinkError}</p>{/if}
	{#snippet footer()}
		<button onclick={() => (addLinkOpen = false)}>Cancel</button>
		<button onclick={addLink}>Add</button>
	{/snippet}
</Modal>

<style>
	.toggle {
		display: block;
		margin-bottom: 1rem;
	}
	.footer-section {
		margin-bottom: 1.5rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid var(--color-border, #ddd);
	}
	.section-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
	.footer-section ul {
		list-style: none;
		padding: 0;
		margin: 0.5rem 0;
	}
	.footer-section li {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.35rem 0;
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
