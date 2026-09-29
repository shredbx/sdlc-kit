<script lang="ts">
	import { TabbedPageShell } from '@sbx/core-ui/components/layouts';
	import type { PageListItem } from './types';

	let { pages, onOpen }: {
		pages: PageListItem[];
		onOpen: (slug: string) => void;
	} = $props();

	// Creating a page by slug is a real, working mechanism (the cms kit's
	// Repository.UpsertBySlug create-on-first-save semantics) — kept as the "new
	// page" affordance, no longer the only way to reach one now that the list is real.
	let newSlug = $state('');

	function createNew() {
		const s = newSlug.trim();
		if (s) onOpen(s);
	}
</script>

<TabbedPageShell title="Pages">
	{#if pages.length === 0}
		<p>No pages yet — create one below.</p>
	{:else}
		<table>
			<thead>
				<tr>
					<th>Slug</th>
					<th>Title</th>
					<th>Status</th>
					<th>Updated</th>
				</tr>
			</thead>
			<tbody>
				{#each pages as page (page.slug)}
					<tr onclick={() => onOpen(page.slug)}>
						<td>{page.slug}</td>
						<td>{page.title ?? ''}</td>
						<td>
							<span class="badge" class:published={page.published}>
								{page.published ? 'Published' : 'Draft'}
							</span>
						</td>
						<td>{new Date(page.updated_at).toLocaleString()}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}

	<form class="new-page" onsubmit={(e) => (e.preventDefault(), createNew())}>
		<label>
			New page slug
			<input bind:value={newSlug} placeholder="about" required pattern="[a-z0-9]+(-[a-z0-9]+)*" />
		</label>
		<button type="submit">Create</button>
	</form>
</TabbedPageShell>

<style>
	table {
		width: 100%;
		border-collapse: collapse;
		margin-bottom: 1.5rem;
	}
	th,
	td {
		text-align: left;
		padding: 0.5rem 0.75rem;
		border-bottom: 1px solid var(--color-border, #ddd);
	}
	tbody tr {
		cursor: pointer;
	}
	tbody tr:hover {
		background: rgba(127, 127, 127, 0.08);
	}
	.badge {
		display: inline-block;
		padding: 0.15rem 0.5rem;
		border-radius: 999px;
		font-size: 0.8rem;
		background: rgba(127, 127, 127, 0.15);
		color: var(--color-text-muted, #666);
	}
	.badge.published {
		background: rgba(34, 197, 94, 0.15);
		color: var(--color-status-success, #2d8b57);
	}
	form.new-page {
		display: flex;
		align-items: flex-end;
		gap: 0.75rem;
		max-width: 24rem;
	}
	form.new-page label {
		flex: 1;
	}
	form.new-page input {
		display: block;
		width: 100%;
		margin-top: 0.25rem;
	}
</style>
