<script lang="ts">
	// Layers context panel (§11.6, redline 2) — its OWN rail item now (Document is
	// dissolved). Two chevron-collapsible sections:
	//   Layers   z-stack of placed elements (drag-reorder · select · 👁 · 🔒)
	//   Pages    frame navigator — thumbnail strip + ＋ add page (stubbed)
	// element = layer (D6) — this Layers list and the Timeline lanes read the SAME
	// layers[] array (single source of truth). Wired to the same editor callbacks
	// the old DocumentPanel used; deep interactions stay as today. (Template +
	// Document info moved to Settings → Document settings, §11.6.)
	import { Icon } from '@sbx/core-ui/components/primitives';
	import type { Layer, Page } from '@sbx/canvas-kit';
	import Panel from './Panel.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import { layerIcon } from './palette.js';

	interface Props {
		layers: Layer[];
		pages: Page[];
		selectedLayerId: string | null;
		/** id of the active page (for thumbnail-strip highlight). */
		activePageId?: string;
		onselect?: (id: string) => void;
		onreorder?: (fromIndex: number, toIndex: number) => void;
		ontogglevisibility?: (id: string) => void;
		ontogglelock?: (id: string) => void;
	}

	let {
		layers,
		pages,
		selectedLayerId,
		activePageId,
		onselect,
		onreorder,
		ontogglevisibility,
		ontogglelock
	}: Props = $props();

	// Layers shown front-most first (top of stack = top of list); the underlying
	// array is back→front, so the display index maps back via (len - 1 - i).
	const displayLayers = $derived([...layers].map((l, i) => ({ layer: l, index: i })).reverse());

	// Reveal the active row whenever the selection changes — and on mount, so the
	// Inspector's "Show in Layers" (D-4) lands with the row in view even when the
	// list scrolls. 'nearest' never jumps when the row is already visible.
	let listEl = $state<HTMLUListElement | null>(null);
	$effect(() => {
		void selectedLayerId;
		listEl?.querySelector('.layers__item--active')?.scrollIntoView({ block: 'nearest' });
	});

	// Drag-reorder — INSERT-BETWEEN, not drop-on-row. The old code passed the TARGET
	// row's array index straight to the splice-insert reducer, which is directional
	// (drag-down lands after / drag-up lands before) and can never reach the extreme
	// ends. Instead we read the pointer's half of the hovered row to decide before/
	// after, map that to an array INSERT-BEFORE index (display is front-first, so the
	// array is reversed), then convert to the reducer's post-removal `to` — so every
	// slot, including the very front and just-above-background, is reachable.
	let dragFromIndex = $state<number | null>(null);
	// The live insertion marker, in DISPLAY space: a gap is drawn before display row
	// `index` (top edge) or after it (bottom edge). null = no active drag/marker.
	let dropMarker = $state<{ index: number; pos: 'before' | 'after' } | null>(null);

	function handleDragStart(event: DragEvent, index: number): void {
		dragFromIndex = index;
		if (event.dataTransfer) {
			event.dataTransfer.effectAllowed = 'move';
			event.dataTransfer.setData('text/plain', String(index));
		}
	}

	/** Which half of the hovered row the pointer is in → insert before (top) or after (bottom). */
	function handleDragOver(event: DragEvent, displayIndex: number): void {
		event.preventDefault();
		if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
		const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
		dropMarker = { index: displayIndex, pos: event.clientY < rect.top + rect.height / 2 ? 'before' : 'after' };
	}

	function handleDragEnd(): void {
		dragFromIndex = null;
		dropMarker = null;
	}

	function handleDrop(event: DragEvent): void {
		event.preventDefault();
		const from = dragFromIndex;
		const marker = dropMarker;
		dragFromIndex = null;
		dropMarker = null;
		if (from === null || !marker) return;

		// Map the DISPLAY-space marker → an array INSERT-BEFORE index. Display is front-
		// first (array reversed), so "before" a display row = a HIGHER array index (more
		// front) and "after" = that row's own array index.
		const targetArrayIndex = displayLayers[marker.index]?.index ?? 0;
		let insertBefore = marker.pos === 'before' ? targetArrayIndex + 1 : targetArrayIndex;
		// The system background is pinned to the bottom (array index 0) — never drop below it.
		if (layers[0]?.system) insertBefore = Math.max(1, insertBefore);

		// Convert insert-before (original array) → the reducer's `to` (index in the array
		// AFTER `from` is spliced out). No-op when the position is unchanged.
		const to = insertBefore > from ? insertBefore - 1 : insertBefore;
		if (to !== from) onreorder?.(from, to);
	}
</script>

<Panel title="Layers" searchPlaceholder="Search layers…">
	<CollapsibleSection title="Layers" count={layers.length}>
		<ul class="layers" bind:this={listEl}>
			{#each displayLayers as entry, di (entry.layer.id)}
				{@const layer = entry.layer}
				<li
					class="layers__item"
					class:layers__item--active={layer.id === selectedLayerId}
					class:layers__item--system={layer.system}
					class:layers__item--drop-before={dropMarker?.index === di && dropMarker.pos === 'before'}
					class:layers__item--drop-after={dropMarker?.index === di && dropMarker.pos === 'after'}
					draggable={!layer.system}
					ondragstart={(e) => handleDragStart(e, entry.index)}
					ondragover={(e) => handleDragOver(e, di)}
					ondrop={handleDrop}
					ondragend={handleDragEnd}
				>
					{#if !layer.system}
						<span class="layers__grip" aria-hidden="true"><Icon name="grip-vertical" size="sm" /></span>
					{:else}
						<span class="layers__grip layers__grip--fixed" aria-hidden="true"></span>
					{/if}
					<button class="layers__select" type="button" onclick={() => onselect?.(layer.id)}>
						<span class="layers__type"><Icon name={layerIcon(layer.type)} size="sm" /></span>
						<span class="layers__name">{layer.name}</span>
					</button>
					<button
						class="layers__toggle"
						type="button"
						aria-label={layer.visible === false ? 'Show layer' : 'Hide layer'}
						onclick={() => ontogglevisibility?.(layer.id)}
					>
						<Icon name={layer.visible === false ? 'eye-off' : 'eye'} size="sm" />
					</button>
					{#if !layer.system}
						<!-- System layers (background) are immovable regardless — an
						     unlockable lock would be a dead affordance (the eye stays:
						     hiding the background is a real feature). -->
						<button
							class="layers__toggle"
							type="button"
							aria-label={layer.locked ? 'Unlock layer' : 'Lock layer'}
							onclick={() => ontogglelock?.(layer.id)}
						>
							<Icon name={layer.locked ? 'lock' : 'unlock'} size="sm" />
						</button>
					{:else}
						<span class="layers__toggle layers__toggle--fixed" aria-hidden="true"></span>
					{/if}
				</li>
			{/each}
		</ul>
	</CollapsibleSection>

	<CollapsibleSection title="Pages" count={pages.length}>
		<div class="pages">
			{#each pages as page, i (page.id)}
				<button
					class="page-card"
					class:page-card--active={page.id === (activePageId ?? pages[0]?.id)}
					type="button"
					aria-label={`Page ${i + 1}`}
				>
					<span class="page-card__thumb" aria-hidden="true"></span>
					<span class="page-card__num">{i + 1}</span>
				</button>
			{/each}
			<!-- Add page — stubbed this cut (single-page mock; multi-page lands later). -->
			<button class="page-card page-card--add" type="button" aria-label="Add page" disabled>
				<Icon name="plus" size="md" />
			</button>
		</div>
	</CollapsibleSection>
</Panel>

<style>
	/* Layers z-stack */
	.layers {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.layers__item {
		display: flex;
		align-items: center;
		gap: var(--cv-space-xs, 0.25rem);
		padding: 0.375rem 0.25rem;
		border-radius: var(--cv-radius-sm, 0.375rem);
		color: var(--cv-color-neutral-700, #383838);
	}

	.layers__item:hover {
		background: var(--cv-color-neutral-100, #f5f5f5);
	}

	.layers__item--active {
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	/* Drag-reorder insertion marker — a crisp line on the edge the drop will land on,
	   so the target slot (incl. the very top / just-above-background) is unambiguous. */
	.layers__item--drop-before {
		box-shadow: inset 0 2px 0 0 var(--cv-color-primary, #333333);
	}

	.layers__item--drop-after {
		box-shadow: inset 0 -2px 0 0 var(--cv-color-primary, #333333);
	}

	.layers__item--system {
		color: var(--cv-color-neutral-500, #707070);
	}

	.layers__grip {
		display: flex;
		align-items: center;
		color: var(--cv-color-neutral-400, #a0a0a0);
		cursor: grab;
	}

	.layers__grip--fixed {
		width: 1rem;
		cursor: default;
	}

	/* Alignment placeholder where the lock toggle is omitted (system rows). */
	.layers__toggle--fixed {
		cursor: default;
	}

	.layers__select {
		flex: 1;
		display: flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		min-width: 0;
		border: none;
		background: transparent;
		color: inherit;
		font: inherit;
		text-align: left;
		cursor: pointer;
	}

	.layers__type {
		display: flex;
		color: var(--cv-color-neutral-500, #707070);
	}

	.layers__name {
		font-size: 0.8125rem;
		font-weight: 500;
	}

	.layers__toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.layers__toggle:hover {
		background: var(--cv-color-neutral-200, #e8e8e8);
		color: var(--cv-color-neutral-800, #202020);
	}

	/* Pages thumbnail strip */
	.pages {
		display: flex;
		flex-wrap: wrap;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.page-card {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 64px;
		height: 80px;
		padding: 0;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-neutral-50, #fafafa);
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.page-card:hover:not(:disabled) {
		border-color: var(--cv-color-primary, #333333);
	}

	.page-card--active {
		border-color: var(--cv-color-primary, #333333);
		box-shadow: 0 0 0 1px var(--cv-color-primary, #333333);
	}

	.page-card__thumb {
		position: absolute;
		inset: 4px;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		border: 1px solid var(--cv-color-neutral-100, #f5f5f5);
	}

	.page-card__num {
		position: relative;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--cv-color-neutral-700, #383838);
	}

	.page-card--add {
		border-style: dashed;
		background: transparent;
	}

	.page-card--add:disabled {
		cursor: not-allowed;
		opacity: 0.6;
	}
</style>
