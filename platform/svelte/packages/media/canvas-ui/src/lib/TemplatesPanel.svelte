<script lang="ts">
	// Templates SECTION (#4) — embedded in the Document panel. Lists the consumer's REAL
	// templates (kind='template') and live-renders each one's first page into its card via
	// the shared export renderer (TemplatePreview) — no fake gradients, no baked snapshot.
	//   ★ Favourites   the favourited subset (session-local ★ toggle), shown first
	//   All templates  every template
	// Apply: the consumer creates a NEW document from the template and opens it
	// (non-destructive "start from template"); over a non-empty doc we confirm first.
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { Button, Icon } from '@sbx/core-ui/components/primitives';
	import type { FormatterRegistry } from '@sbx/canvas-kit';
	import type { TemplateGallery, TemplateSummary } from './template-gallery.js';
	import ComponentGrid from './ComponentGrid.svelte';
	import ComponentCard from './ComponentCard.svelte';
	import EmptyState from './EmptyState.svelte';
	import PanelSearch from './PanelSearch.svelte';
	import TemplatePreview from './TemplatePreview.svelte';

	interface Props {
		/** Real templates + lazy doc loader + apply handler (consumer-owned). Omit → empty. */
		gallery?: TemplateGallery;
		/** Image proxy + formatters forwarded to each live preview (same as the stage). */
		resolveImageSrc?: (src: string) => string;
		formatters?: FormatterRegistry;
		/** Whether the current document already has user content (drives the apply confirm:
		 *  empty → apply directly; non-empty → warn first). */
		documentHasContent?: boolean;
	}

	let { gallery, resolveImageSrc = (s) => s, formatters = {}, documentHasContent = true }: Props =
		$props();

	const templates = $derived(gallery?.templates ?? []);

	// Session-local favourites (★ toggle). Persistence is a clean follow-up — the panel
	// stays generic (no localStorage key baked into the package).
	let favouriteIds = $state<Set<string>>(new Set());

	// Scoped search — filters by title, case-insensitive. Empty query → everything.
	let query = $state('');
	const matches = $derived.by(() => {
		const q = query.trim().toLowerCase();
		return q ? templates.filter((t) => t.title.toLowerCase().includes(q)) : templates;
	});
	const favourites = $derived(matches.filter((t) => favouriteIds.has(t.id)));

	let confirmTemplate = $state<TemplateSummary | null>(null);

	function toggleFavourite(t: TemplateSummary): void {
		const next = new Set(favouriteIds);
		next.has(t.id) ? next.delete(t.id) : next.add(t.id);
		favouriteIds = next;
	}

	function requestApply(t: TemplateSummary): void {
		if (documentHasContent) confirmTemplate = t;
		else apply(t);
	}

	function apply(t: TemplateSummary): void {
		confirmTemplate = null;
		gallery?.onApply(t.id);
	}

	const confirmName = $derived(confirmTemplate?.title ?? '');
</script>

{#snippet templateCard(t: TemplateSummary)}
	<ComponentCard label={t.title} onclick={() => requestApply(t)}>
		{#snippet thumb()}
			<span class="thumb">
				{#if gallery}
					<TemplatePreview id={t.id} load={gallery.loadDoc} {resolveImageSrc} {formatters} />
				{/if}
			</span>
		{/snippet}
		{#snippet overlay()}
			<button
				class="ovl-btn"
				type="button"
				aria-label={`Apply ${t.title}`}
				title="Apply"
				onclick={(e) => {
					e.stopPropagation();
					requestApply(t);
				}}
			>
				<Icon name="check" size="sm" />
			</button>
			<button
				class="ovl-btn"
				class:ovl-btn--on={favouriteIds.has(t.id)}
				type="button"
				aria-pressed={favouriteIds.has(t.id)}
				aria-label={favouriteIds.has(t.id) ? `Unfavourite ${t.title}` : `Favourite ${t.title}`}
				title={favouriteIds.has(t.id) ? 'Unfavourite' : 'Favourite'}
				onclick={(e) => {
					e.stopPropagation();
					toggleFavourite(t);
				}}
			>
				<Icon name="star" size="sm" />
			</button>
		{/snippet}
	</ComponentCard>
{/snippet}

<div class="templates">
	<div class="templates__search">
		<PanelSearch bind:value={query} placeholder="Search templates…" label="Search templates" />
	</div>

	{#if templates.length === 0}
		<EmptyState
			text="No templates yet. Save a document as a template (Document settings → Kind) and it appears here."
		/>
	{:else if matches.length === 0}
		<EmptyState text="No templates match “{query}”." />
	{:else}
		{#if favourites.length > 0}
			<section class="block">
				<h3 class="block__title"><Icon name="star" size="sm" />Favourites</h3>
				<ComponentGrid cols={2}>
					{#each favourites as t (t.id)}
						{@render templateCard(t)}
					{/each}
				</ComponentGrid>
			</section>
		{/if}

		<section class="block">
			<h3 class="block__title">All templates</h3>
			<ComponentGrid cols={2}>
				{#each matches as t (t.id)}
					{@render templateCard(t)}
				{/each}
			</ComponentGrid>
		</section>
	{/if}
</div>

<!-- Apply over a non-empty document → confirm (a new doc is created FROM the template). -->
<Modal open={confirmTemplate !== null} title="Use this template?" onclose={() => (confirmTemplate = null)}>
	<p class="confirm">
		Start a new document from “{confirmName}”? Your current document stays untouched.
	</p>
	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => (confirmTemplate = null)}>Cancel</Button>
		<Button variant="primary" size="sm" onclick={() => confirmTemplate && apply(confirmTemplate)}>
			Use template
		</Button>
	{/snippet}
</Modal>

<style>
	.templates {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-md, 1rem);
	}

	.templates__search {
		display: flex;
	}

	.block {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.block__title {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		margin: 0;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	/* Live-preview thumbnail — fills the ComponentCard --thumb media slot. */
	.thumb {
		position: relative;
		display: block;
		width: 100%;
		height: 100%;
	}

	/* Hover overlay buttons — dark pills like the property-gallery set-cover icon. */
	.ovl-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.5rem;
		height: 1.5rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: rgba(16, 16, 8, 0.6);
		color: #fff;
		cursor: pointer;
		transition: background 0.15s ease;
	}

	.ovl-btn:hover {
		background: rgba(16, 16, 8, 0.85);
	}

	.ovl-btn--on {
		background: var(--cv-color-accent-dark, #555555);
	}

	.confirm {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.5;
		color: var(--cv-color-neutral-700, #383838);
	}
</style>
