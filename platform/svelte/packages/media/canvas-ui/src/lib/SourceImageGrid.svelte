<script lang="ts">
	// Shared image-group grid (G5) — the category control + thumbnail grid extracted from
	// MediaPanel so the SAME content backs both the left-rail Media panel AND the Inspector
	// image-bind popup (G6). A thumbnail click emits the composed token
	// images.<group>.<index>.url; binding stays the caller's job.
	//
	// ONE behaviour (no count-based switching): an All / Categories toggle.
	//   • All        → every image in a single flat grid (no section headers).
	//   • Categories → one expandable section per image group (Cover, Gallery, …).
	//
	// Action guidance lives on the thumbnail as a hover tooltip (no persistent hint line).
	import type { ImageCategory, ResolvedImage } from '@sbx/canvas-kit';
	import { pickImageVariant } from '@sbx/canvas-kit';
	import { Icon } from '@sbx/core-ui/components/primitives';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import ThumbImage from './ThumbImage.svelte';
	import { IMAGE_BIND_MIME, type ImageBindDrag } from './dnd.js';

	interface Props {
		/** Image groups the source offers (provider.imageCategories()). */
		categories?: ImageCategory[];
		/** Resolved images keyed by group id ({ cover, gallery, … }). */
		images?: Record<string, ResolvedImage[]>;
		/** A bind target is selected (otherwise thumbnails are inert + the tooltip explains). */
		canBind?: boolean;
		/** Token bound to the current target — highlights the live slot. */
		boundToken?: string;
		/** Thumbnails are draggable onto the canvas (G8). Needs `alias` to compose the payload.
		 *  Independent of canBind: a drag can create a NEW image layer with no layer selected. */
		draggable?: boolean;
		/** The source alias these images belong to — carried in the drag payload (G8). */
		alias?: string;
		/** A thumbnail was chosen WHILE an image layer is selected — caller REPLACES that
		 *  layer's src with the composed `images.<group>.<index>.url`. */
		onbind?: (token: string) => void;
		/** A thumbnail was chosen with NOTHING (re)bindable selected — caller PLACES a new
		 *  image layer in the page centre (Canva-style instant drop). Independent of
		 *  canBind: when present, a thumbnail click is never inert. */
		oninsert?: (token: string) => void;
	}

	let { categories = [], images = {}, canBind = false, boundToken, draggable = false, alias, onbind, oninsert }: Props =
		$props();

	/** Begin a drag (G8) — carry the composed token + alias so CanvasStage can bind on drop. */
	function onThumbDragStart(event: DragEvent, token: string): void {
		if (!draggable || !alias || !event.dataTransfer) return;
		const payload: ImageBindDrag = { token, alias };
		event.dataTransfer.setData(IMAGE_BIND_MIME, JSON.stringify(payload));
		event.dataTransfer.effectAllowed = 'copy';
	}

	let activeView = $state<'all' | 'categories'>('all');

	// Reveal the bound slot whenever the binding changes — and on mount, so opening
	// the panel/popup (or the Inspector's ⌖ reveal) lands with the linked thumbnail
	// in view (E-2). 'nearest' never jumps when it's already visible.
	let rootEl = $state<HTMLDivElement | null>(null);
	$effect(() => {
		void boundToken;
		rootEl?.querySelector('.thumb--active')?.scrollIntoView({ block: 'nearest' });
	});

	function countFor(groupId: string): number {
		return (images[groupId] ?? []).length;
	}
	function labelFor(groupId: string): string {
		return categories.find((c) => c.id === groupId)?.label ?? groupId;
	}
	const totalCount = $derived(categories.reduce((n, c) => n + countFor(c.id), 0));

	function itemsFor(groupIds: string[]): { group: string; index: number; image: ResolvedImage }[] {
		const out: { group: string; index: number; image: ResolvedImage }[] = [];
		for (const groupId of groupIds) {
			(images[groupId] ?? []).forEach((image, index) => out.push({ group: groupId, index, image }));
		}
		return out;
	}
	function tokenFor(group: string, index: number): string {
		return `images.${group}.${index}.url`;
	}
	const actionTip = $derived(
		canBind
			? draggable
				? 'Click to replace the selected image · drag onto the canvas to place'
				: 'Click to replace the selected image'
			: oninsert
				? draggable
					? 'Click to place · drag onto an image to replace it'
					: 'Click to place'
				: draggable
					? 'Drag onto the canvas to place · select an image layer to replace'
					: 'Select an image layer to link'
	);
</script>

{#snippet grid(items: { group: string; index: number; image: ResolvedImage }[])}
	<ul class="grid">
		{#each items as item (item.group + ':' + item.index)}
			{@const token = tokenFor(item.group, item.index)}
			{@const active = boundToken === token}
			<li>
				<button
					type="button"
					class="thumb"
					class:thumb--active={active}
					class:thumb--draggable={draggable}
					disabled={!canBind && !draggable && !oninsert}
					draggable={draggable}
					aria-pressed={active}
					title={`${labelFor(item.group)} ${item.index + 1} — ${actionTip}`}
					onclick={() => (canBind ? onbind?.(token) : oninsert?.(token))}
					ondragstart={(e) => onThumbDragStart(e, token)}
				>
					<ThumbImage src={pickImageVariant(item.image, 'thumbnail')} label={`${labelFor(item.group)} ${item.index + 1}`} />
					{#if active}
						<span class="thumb__badge" aria-hidden="true"><Icon name="check" size="xs" /></span>
					{/if}
				</button>
			</li>
		{/each}
	</ul>
{/snippet}

<!-- Single root so the reveal effect scopes its query to THIS grid instance
     (the Media panel and the Inspector popup can render simultaneously). -->
<div bind:this={rootEl}>
<div class="toggle" role="tablist" aria-label="Image view">
	<button
		type="button"
		class="toggle__btn"
		class:toggle__btn--active={activeView === 'all'}
		role="tab"
		aria-selected={activeView === 'all'}
		onclick={() => (activeView = 'all')}
	>
		All<span class="toggle__count">{totalCount}</span>
	</button>
	<button
		type="button"
		class="toggle__btn"
		class:toggle__btn--active={activeView === 'categories'}
		role="tab"
		aria-selected={activeView === 'categories'}
		onclick={() => (activeView = 'categories')}
	>
		Categories
	</button>
</div>

{#if activeView === 'all'}
	{@render grid(itemsFor(categories.map((c) => c.id)))}
{:else}
	<div class="sections">
		{#each categories as c (c.id)}
			<CollapsibleSection title={c.label} count={countFor(c.id)}>
				{@render grid(itemsFor([c.id]))}
			</CollapsibleSection>
		{/each}
	</div>
{/if}
</div>

<style>
	.toggle {
		display: flex;
		gap: 0.375rem;
		margin-bottom: var(--cv-space-sm, 0.5rem);
	}

	.toggle__btn {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.625rem;
		font: inherit;
		font-size: 0.8125rem;
		color: var(--cv-color-neutral-700, #383838);
		background: transparent;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: 999px;
		cursor: pointer;
	}

	.toggle__btn--active {
		color: #fff;
		background: var(--cv-color-primary, #333333);
		border-color: var(--cv-color-primary, #333333);
	}

	.toggle__count {
		font-size: 0.6875rem;
		opacity: 0.8;
	}

	.sections {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
		gap: 0.5rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.thumb {
		position: relative;
		display: block;
		width: 100%;
		aspect-ratio: 4 / 3;
		padding: 0;
		overflow: hidden;
		background: var(--cv-color-neutral-100, #f5f5f5);
		border: 2px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		cursor: pointer;
	}

	.thumb:disabled {
		cursor: default;
	}

	/* Draggable thumbs (G8) read as grabbable; the active drag shows the grabbing cursor. */
	.thumb--draggable {
		cursor: grab;
	}

	.thumb--draggable:active {
		cursor: grabbing;
	}

	.thumb--active {
		border-color: var(--cv-color-primary, #333333);
	}

	/* Keyboard focus ring — the thumbnails are tab-navigable buttons (WCAG 2.4.7); the
	   outline is distinct from the --active selected border. */
	.thumb:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 2px;
	}

	.thumb__badge {
		position: absolute;
		top: 0.25rem;
		right: 0.25rem;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.125rem;
		height: 1.125rem;
		color: #fff;
		background: var(--cv-color-primary, #333333);
		border-radius: 999px;
	}
</style>
