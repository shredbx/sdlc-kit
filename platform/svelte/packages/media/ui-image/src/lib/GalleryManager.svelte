<script lang="ts">
	import type { ImageData } from '@sbx/core-ui/types';
	import type { Snippet } from 'svelte';
	import ImageCard from './ImageCard.svelte';
	import { dragStartPayload } from './dragPayload';
	import { extractDropSources, sourceToFile } from './pipeline';

	/** Optimistic-tile state — tiles whose id matches a key render a per-tile
	 *  overlay with spinner + label. Used by consumers that upload async to
	 *  give the user immediate feedback that work is happening on a specific
	 *  photo, rather than a single global banner that's easy to miss on mobile.
	 *  Status drives the overlay style; label is free text shown beneath the
	 *  spinner; progress is an optional 0-100 percentage. onretry, when set,
	 *  swaps the error overlay for a clickable retry icon. */
	export type TilePendingState =
		| { status: 'queued'; label?: string }
		| { status: 'processing'; label?: string; progress?: number }
		| { status: 'uploading'; label?: string; progress?: number }
		| { status: 'error'; label?: string; onretry?: () => void };

	interface GalleryManagerProps {
		images: ImageData[];
		/** Which gallery item is the property's cover. ID match wins; if the
		 * caller only knows the URL, pass it here too (legacy). */
		coverImageId?: string;
		canReorder?: boolean;
		canDelete?: boolean;
		canSetCover?: boolean;
		canSelect?: boolean;
		/** How the per-tile remove (canDelete) reads: 'delete' (default) = trash icon +
		 *  inline 2-click "Confirm?" (destructive). 'unlink' = a single-click ✕, no
		 *  confirm — for non-destructive removal (e.g. unlink from an album; the image
		 *  stays in the library). Fires the same `ondelete`. Default 'delete'. */
		removeMode?: 'delete' | 'unlink';
		/** When true, tiles are draggable and stamp `application/x-image-id`
		 *  (+ `text/plain`=the image id) on drag-start EVEN when canReorder=false,
		 *  so a non-reorderable LIBRARY pane can still be a cross-pane drag SOURCE.
		 *  When canReorder=true the internal-reorder payload (text/plain=index)
		 *  takes precedence — the two payloads never collide. Default false. */
		dragSource?: boolean;
		/** When set, drag-start stamps `text/plain` with this callback's result
		 *  INSTEAD of the index/id, so dropping a tile on any native text drop
		 *  target (e.g. a markdown textarea) inserts consumer-shaped text at the
		 *  caret — the browser performs the insertion. Reorder is unaffected
		 *  (it is driven by internal state, not text/plain) and
		 *  `application/x-image-id` is still stamped for cross-pane targets.
		 *  Default unset → existing payload behavior, byte-identical. */
		dragText?: (image: ImageData) => string;
		cardWidth?: string;
		/** When set, the grid lays tiles out as N EVEN columns (CSS grid,
		 *  repeat(N, 1fr)) that FILL the width — instead of the default left-aligned
		 *  flex-wrap of fixed-`cardWidth` tiles (which leaves a trailing gap). Use for
		 *  dense rails that want clean, aligned columns. Default undefined (flex-wrap).
		 *  When set, `cardWidth` is ignored (the grid cell sizes the tile). */
		columns?: number;
		aspectRatio?: '4:3' | '16:9' | '3:2' | '1:1';
		acceptFiles?: boolean;
		/** Per-tile pending state. Keyed by ImageData.id; matched tiles render
		 *  a spinner + status overlay. Use for optimistic upload UX. */
		pendingState?: Record<string, TilePendingState>;
		/** Image ids that render DIMMED (e.g. a source pane dimming images already
		 *  present in a linked target pane). Default []. */
		mutedIds?: string[];
		/** Override the empty-state copy. When acceptFiles, the empty grid is a
		 *  droppable zone (files + a cross-pane source image both land here); a
		 *  consumer (e.g. a target album pane) can supply its own wording. When
		 *  omitted the built-in copy is used. */
		emptyLabel?: string;
		onreorder?: (images: ImageData[]) => void;
		ondelete?: (image: ImageData) => void;
		ondeletemany?: (ids: string[]) => void;
		onsetcover?: (image: ImageData) => void;
		onpreview?: (image: ImageData, index: number) => void;
		/** External file drop. When dropped on a TILE, `index` is that tile's
		 *  position; the grid-level (empty space) drop passes no index → append.
		 *  Single-pane callers may ignore the 2nd arg. */
		onfilesdrop?: (files: File[], index?: number) => void;
		/** Cross-pane link — fired in handleDrop when the drop carries
		 *  `application/x-image-id`, there is NO internal reorder in progress, and
		 *  no files. `index` is the target tile position (undefined for a
		 *  grid-level drop → append). Lets a TARGET pane receive a linked image. */
		onassignfromsource?: (imageId: string, index?: number) => void;
		/** Optional — fired when the empty drop-zone is clicked. Consumer wires
		 *  this to a hidden <input type="file"> click() so users can pick from
		 *  disk without drag-and-drop. Drop-zone stays drag-friendly either way. */
		onpickfiles?: () => void;
		/** Optional — delegate the bulk-delete CONFIRM UI to the consumer so it can
		 *  render its own themed dialog (e.g. BR ConfirmDialog over AdminModal)
		 *  instead of the built-in modal. Receives the open state + selected count
		 *  + the confirm/cancel handlers; GalleryManager still owns the selection and
		 *  the actual ondeletemany call. When omitted, the built-in modal is used. */
		confirmDialog?: Snippet<[{ open: boolean; count: number; onconfirm: () => void; oncancel: () => void }]>;
		/** Optional per-tile overlay snippet — rendered inside EACH tile (which is
		 *  position:relative) so a consumer can stamp its own corner badge / status
		 *  marker (e.g. a watermarked shield) without forking the gallery engine.
		 *  Receives the tile's image + its current selection state; the consumer
		 *  positions its own absolute element. Default unset → nothing rendered, so
		 *  existing callers are unaffected. */
		cardOverlay?: Snippet<[{ image: ImageData; isSelected: boolean }]>;
		/** Optional — fired whenever the multi-select set changes, emitting the
		 *  current selected-id list. Lets a host mirror the selection into its own
		 *  state (e.g. a capability action bar). Fires once on mount with [] (harmless).
		 *  Default unset → never called, so existing callers are unaffected. */
		onselectionchange?: (ids: string[]) => void;
		class?: string;
	}

	let {
		images = [],
		coverImageId = '',
		canReorder = true,
		canDelete = true,
		canSetCover = true,
		canSelect = false,
		removeMode = 'delete',
		dragSource = false,
		dragText,
		cardWidth = '160px',
		columns,
		aspectRatio = '4:3',
		acceptFiles = false,
		pendingState = {},
		mutedIds = [],
		emptyLabel,
		onreorder,
		ondelete,
		ondeletemany,
		onsetcover,
		onpreview,
		onfilesdrop,
		onassignfromsource,
		onpickfiles,
		confirmDialog,
		cardOverlay,
		onselectionchange,
		class: className = ''
	}: GalleryManagerProps = $props();

	// Set of dimmed ids — derived so the lookup is O(1) per tile.
	const mutedSet = $derived(new Set(mutedIds));

	// Multi-select state — desktop modifier pattern per locked design call:
	//   plain click          → onpreview (lightbox)
	//   Shift-click          → range extend from anchor to current
	//   Cmd/Ctrl-click       → toggle individual
	// Tracks selection by image.id so reorder/delete don't drift the set.
	let selectedIds = $state<Set<string>>(new Set());
	let anchorIndex = $state<number | null>(null);
	let bulkConfirmOpen = $state(false);

	// Mirror selection changes out to a host (multi-select callers). Reads
	// selectedIds so it re-runs on every change; spreads to a plain array so the
	// consumer can't mutate the internal Set. Fires once on mount with [] (harmless).
	$effect(() => {
		onselectionchange?.([...selectedIds]);
	});

	function selectionCount() {
		return selectedIds.size;
	}

	function isSelected(id: string): boolean {
		return selectedIds.has(id);
	}

	// Exported so a parent (via bind:this) can clear the selection after it has
	// consumed it (e.g. a capability that dispatched an apply over the selection).
	// The onselectionchange $effect below re-emits the cleared set.
	export function clearSelection() {
		selectedIds = new Set();
		anchorIndex = null;
	}

	function toggleOne(id: string, index: number) {
		const next = new Set(selectedIds);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		selectedIds = next;
		anchorIndex = index;
	}

	function extendRange(toIndex: number) {
		if (anchorIndex === null) {
			anchorIndex = toIndex;
			selectedIds = new Set([images[toIndex].id]);
			return;
		}
		const start = Math.min(anchorIndex, toIndex);
		const end = Math.max(anchorIndex, toIndex);
		const next = new Set(selectedIds);
		for (let i = start; i <= end; i++) {
			next.add(images[i].id);
		}
		selectedIds = next;
	}

	function handleTileClick(e: MouseEvent, image: ImageData, index: number) {
		if (!canSelect) {
			onpreview?.(image, index);
			return;
		}
		if (e.shiftKey) {
			e.preventDefault();
			extendRange(index);
			return;
		}
		if (e.metaKey || e.ctrlKey) {
			e.preventDefault();
			toggleOne(image.id, index);
			return;
		}
		// Plain click while items are already selected → keep selection mental
		// model intact (don't blow it away to open lightbox). User can hit Esc
		// to clear, or click an unselected tile to open lightbox.
		if (selectedIds.size > 0 && selectedIds.has(image.id)) {
			toggleOne(image.id, index);
			return;
		}
		if (selectedIds.size > 0) {
			clearSelection();
		}
		onpreview?.(image, index);
	}

	function handleBulkDeleteClick() {
		if (selectedIds.size === 0) return;
		bulkConfirmOpen = true;
	}

	function handleBulkDeleteConfirm() {
		ondeletemany?.([...selectedIds]);
		clearSelection();
		bulkConfirmOpen = false;
	}

	function handleBulkDeleteCancel() {
		bulkConfirmOpen = false;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!canSelect) return;
		if (e.key === 'Escape' && selectedIds.size > 0) {
			clearSelection();
		}
	}

	let dragIdx = $state<number | null>(null);
	// Positional insert index for the drop indicator (the accent line between tiles).
	// null while no tile is hovered. For a hovered tile it is `index` (left half → insert
	// before) or `index + 1` (right half → insert after); `images.length` means append.
	// Drives a SINGLE affordance for both cross-pane links and internal reorder — the
	// previous whole-tile `is-drop-target` border is retired in favour of this line.
	let dropInsertIndex = $state<number | null>(null);
	let fileDragActive = $state(false);

	function handleDragStart(e: DragEvent, index: number, image: ImageData) {
		// Reorderable → internal reorder drag (tracked via dragIdx; text/plain =
		// index unless the consumer overrides it with dragText). Non-reorderable
		// but dragSource → cross-pane SOURCE drag (dragIdx stays null so a target
		// pane's handleDrop treats it as a link, never a reorder). Neither → no
		// drag. The flavor set lives in dragStartPayload (unit-tested contract);
		// `application/x-image-id` is always stamped — cover slots and cross-pane
		// targets read it to re-designate without re-uploading.
		const payload = dragStartPayload({ canReorder, dragSource, dragText, image, index });
		if (!payload || !e.dataTransfer) return;
		if (canReorder) dragIdx = index;
		e.dataTransfer.effectAllowed = payload.effectAllowed;
		for (const [format, data] of payload.flavors) {
			e.dataTransfer.setData(format, data);
		}
	}

	function handleDragOver(e: DragEvent, index: number) {
		e.preventDefault();
		// Positional insert indicator — applies to BOTH cross-pane links (dragIdx === null,
		// a library tile dragged over the target) AND internal reorder (dragIdx !== null).
		// The library pane (canReorder=false, no onassignfromsource) bails: drops there are
		// uploads, not positional inserts, so no line is shown.
		if (!canReorder && !onassignfromsource) return;
		const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
		const isRightHalf = e.clientX - rect.left > rect.width / 2;
		const insert = isRightHalf ? index + 1 : index;
		// Reorder no-op: hovering on/adjacent to the dragged tile itself → nothing to show.
		if (dragIdx !== null && (insert === dragIdx || insert === dragIdx + 1)) {
			dropInsertIndex = null;
			return;
		}
		dropInsertIndex = insert;
	}

	// Resolve a drop into Files — real files pass through; dropped image URLs /
	// data-URIs (cross-tab drags) are fetched/decoded to Files so they queue for
	// upload exactly like a file. CORS-blocked remote URLs that can't be fetched
	// client-side are skipped here (the picker's server-import fallback is the
	// path for those); the consumer's per-tile upload UX surfaces any failure.
	async function resolveDropToFiles(e: DragEvent): Promise<File[]> {
		const sources = extractDropSources(e);
		if (sources.files.length > 0) return sources.files;
		const urlSources = [...sources.dataUris, ...sources.urls];
		const resolved: File[] = [];
		for (const src of urlSources) {
			try {
				resolved.push(await sourceToFile(src));
			} catch {
				// Skip unresolvable (e.g. CORS-blocked) sources — no silent throw.
			}
		}
		return resolved;
	}

	async function handleDrop(e: DragEvent, index: number) {
		e.preventDefault();
		// A tile owns its own drop — stop the event bubbling to the grid-level handler
		// (which would otherwise ALSO fire and append the same payload). The grid handler
		// only runs for drops on empty grid space.
		e.stopPropagation();
		const at = dropInsertIndex ?? index;

		// Cross-pane link — MUST be resolved before the hasExternal guard below. A
		// SOURCE-pane tile drag stamps `application/x-image-id`; the browser ALSO
		// auto-attaches text/uri-list + text/html to the dragged <img>, so hasExternal
		// is TRUE for a pure cross-pane link. Without this order, resolveDropToFiles
		// would bail on the authoritative internal id (→ []), the branch would return
		// without stopPropagation, and the grid handler would append — the "always to
		// the end" bug. Internal reorder (dragIdx !== null, also stamps the id) skips
		// this branch and falls through to the reorder logic.
		const crossPaneId = e.dataTransfer?.getData('application/x-image-id');
		if (dragIdx === null && crossPaneId) {
			fileDragActive = false;
			dropInsertIndex = null;
			onassignfromsource?.(crossPaneId, at);
			return;
		}

		// External file / image-URL drop on a tile — treat as upload-to-gallery at the
		// hovered position. An internal reorder drag carries no real files/urls.
		const hasExternal =
			(e.dataTransfer?.files.length ?? 0) > 0 ||
			(e.dataTransfer?.types.includes('text/uri-list') ?? false) ||
			(e.dataTransfer?.types.includes('text/html') ?? false);
		if (hasExternal && dragIdx === null) {
			fileDragActive = false;
			dropInsertIndex = null;
			const resolved = await resolveDropToFiles(e);
			if (resolved.length > 0) onfilesdrop?.(resolved, at);
			return;
		}

		if (dragIdx === null || dragIdx === index) {
			dragIdx = null;
			dropInsertIndex = null;
			return;
		}

		// Internal reorder at the computed insert position. After splicing the source
		// out, an insert point that sat AFTER the source shifts down by one.
		const reordered = [...images];
		const [moved] = reordered.splice(dragIdx, 1);
		const target = at > dragIdx ? at - 1 : at;
		reordered.splice(target, 0, moved);

		dragIdx = null;
		dropInsertIndex = null;
		// The browser auto-attaches text/html to a dragged <img>, so the grid-level
		// dragover flagged this reorder as an "external" drag — clear the file-drop
		// affordance here (and in dragend) or the dashed outline sticks after every
		// reorder.
		fileDragActive = false;
		onreorder?.(reordered);
	}

	function handleDragEnd() {
		dragIdx = null;
		dropInsertIndex = null;
		fileDragActive = false;
	}

	// Grid-level handlers — catch file/URL drops on empty space + show affordance.
	// A drag is "external" when it carries OS files OR a URL payload (text/uri-list
	// or text/html) — the latter covers cross-tab image drags.
	function isExternalDrag(e: DragEvent): boolean {
		const types = e.dataTransfer?.types;
		if (!types) return false;
		return types.includes('Files') || types.includes('text/uri-list') || types.includes('text/html');
	}

	// A cross-pane source drag carries an image id but is NOT an internal reorder
	// (dragIdx === null) — the grid-level handler treats it as an append-link.
	function isCrossPaneSourceDrag(e: DragEvent): boolean {
		if (!onassignfromsource || dragIdx !== null) return false;
		return e.dataTransfer?.types.includes('application/x-image-id') ?? false;
	}

	function handleGridDragOver(e: DragEvent) {
		if (acceptFiles && isExternalDrag(e)) {
			e.preventDefault();
			fileDragActive = true;
			// Over empty grid space (not bubbled from a tile) → append indicator.
			if (e.target === e.currentTarget) dropInsertIndex = images.length;
			return;
		}
		// Allow dropping a source-pane image onto empty grid space → append-link.
		if (isCrossPaneSourceDrag(e)) {
			e.preventDefault();
			fileDragActive = true;
			if (e.target === e.currentTarget) dropInsertIndex = images.length;
		}
	}

	function handleGridDragLeave(e: DragEvent) {
		// Only clear when leaving the grid itself, not when crossing tile borders.
		if (e.currentTarget === e.target) {
			fileDragActive = false;
			dropInsertIndex = null;
		}
	}

	async function handleGridDrop(e: DragEvent) {
		// Cross-pane append-link — a source-pane image dropped on empty grid space.
		if (isCrossPaneSourceDrag(e)) {
			e.preventDefault();
			fileDragActive = false;
			dropInsertIndex = null;
			const imageId = e.dataTransfer?.getData('application/x-image-id');
			if (imageId) onassignfromsource?.(imageId, undefined);
			return;
		}
		if (!acceptFiles) return;
		if (!isExternalDrag(e)) return;
		e.preventDefault();
		fileDragActive = false;
		dropInsertIndex = null;
		const resolved = await resolveDropToFiles(e);
		if (resolved.length > 0) onfilesdrop?.(resolved, undefined);
	}

	function setCover(image: ImageData) {
		if (!canSetCover) return;
		onsetcover?.(image);
	}

	// Inline-confirm delete (per 2605-070 design call):
	//   1st click on trash → button morphs to "Confirm?" for 4s
	//   2nd click in window → commits delete
	//   click elsewhere / timeout → silent cancel
	const CONFIRM_WINDOW_MS = 4000;
	let confirmingDeleteId = $state<string | null>(null);
	let confirmTimer: ReturnType<typeof setTimeout> | null = null;

	function clearConfirmingDelete() {
		if (confirmTimer !== null) {
			clearTimeout(confirmTimer);
			confirmTimer = null;
		}
		confirmingDeleteId = null;
	}

	function handleDeleteClick(image: ImageData) {
		if (!canDelete) return;
		if (confirmingDeleteId === image.id) {
			clearConfirmingDelete();
			ondelete?.(image);
			return;
		}
		// Move confirmation focus to this tile; previously-confirming tile resets.
		clearConfirmingDelete();
		confirmingDeleteId = image.id;
		confirmTimer = setTimeout(() => {
			confirmingDeleteId = null;
			confirmTimer = null;
		}, CONFIRM_WINDOW_MS);
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if canSelect && selectionCount() > 0}
	<div class="gallery-manager__bulk-bar" role="toolbar" aria-label="Bulk image actions">
		<span class="gallery-manager__bulk-count">{selectionCount()} selected</span>
		<div class="gallery-manager__bulk-actions">
			{#if canDelete}
				<button type="button" class="gallery-manager__bulk-btn gallery-manager__bulk-btn--danger" onclick={handleBulkDeleteClick}>
					Delete {selectionCount()}
				</button>
			{/if}
			<button type="button" class="gallery-manager__bulk-btn" onclick={clearSelection}>Cancel</button>
		</div>
	</div>
{/if}

<div
	class="gallery-manager {className}"
	class:is-file-dragover={fileDragActive}
	ondragover={handleGridDragOver}
	ondragleave={handleGridDragLeave}
	ondrop={handleGridDrop}
	role="region"
	aria-label="Image gallery"
>
	{#if images.length === 0}
		{#if acceptFiles}
			<!-- Empty drop-zone is also clickable when a picker callback is wired —
			     mirrors the cover-slot empty-state behaviour: drag OR click. -->
			<!-- role + tabindex are both gated on `onpickfiles`; the compiler can't
			     correlate the two conditionals, but tabindex=0 only ever co-occurs
			     with role="button" and the keydown handler below, so it is safe. -->
			<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
			<div
				class="gallery-manager__empty gallery-manager__empty--droppable"
				class:gallery-manager__empty--clickable={onpickfiles !== undefined}
				role={onpickfiles ? 'button' : undefined}
				tabindex={onpickfiles ? 0 : undefined}
				onclick={() => onpickfiles?.()}
				onkeydown={(e) => {
					if (onpickfiles && (e.key === 'Enter' || e.key === ' ')) {
						e.preventDefault();
						onpickfiles();
					}
				}}
			>
				<svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
					<polyline points="17 8 12 3 7 8" />
					<line x1="12" y1="3" x2="12" y2="15" />
				</svg>
				<p>
					{#if emptyLabel}
						{emptyLabel}
					{:else if onpickfiles}
						Drop photos here or <strong>click to browse</strong>
					{:else}
						Drop photos here to add them to the gallery
					{/if}
				</p>
			</div>
		{:else}
			<p class="gallery-manager__empty">No images yet</p>
		{/if}
	{:else}
		<div
			class="gallery-manager__grid"
			class:gallery-manager__grid--even={columns != null}
			style:--gm-cols={columns ?? null}
		>
			{#each images as image, index (image.id)}
				<div
					class="gallery-manager__item"
					class:is-dragging={dragIdx === index}
					class:is-insert-before={dropInsertIndex === index}
					class:is-insert-after={dropInsertIndex === images.length && index === images.length - 1}
					class:is-cover={image.id === coverImageId}
					class:is-selected={isSelected(image.id)}
					class:is-muted={mutedSet.has(image.id)}
					draggable={canReorder || dragSource}
					ondragstart={(e) => handleDragStart(e, index, image)}
					ondragover={(e) => handleDragOver(e, index)}
					ondrop={(e) => handleDrop(e, index)}
					ondragend={handleDragEnd}
					onclick={(e) => handleTileClick(e, image, index)}
					onkeydown={(e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							handleTileClick(e as unknown as MouseEvent, image, index);
						}
					}}
					style:width={columns != null ? null : cardWidth}
					role="button"
					tabindex="0"
					aria-pressed={isSelected(image.id)}
				>
					<ImageCard {image} {aspectRatio} lazy={true} />

					<!-- Consumer-supplied per-tile overlay (e.g. a watermarked badge). The
					     tile is position:relative, so the consumer positions its own absolute
					     element; nothing renders when the snippet is unset. -->
					{#if cardOverlay}{@render cardOverlay({ image, isSelected: isSelected(image.id) })}{/if}

					<!-- Optimistic upload overlay — per-tile spinner / status. Set by
					     the consumer via pendingState[image.id]. processing /
					     uploading render a dimmed backdrop + spinner; error swaps to
					     a retry affordance when onretry is wired. pointer-events:
					     none on processing/uploading so the tile stays draggable
					     except for the retry button which re-enables clicks. -->
					{#if pendingState[image.id]}
						{@const ps = pendingState[image.id]}
						<div
							class="gallery-manager__pending"
							class:is-error={ps.status === 'error'}
							class:is-queued={ps.status === 'queued'}
						>
							{#if ps.status === 'error'}
								<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
									<circle cx="12" cy="12" r="10"/>
									<line x1="12" y1="8" x2="12" y2="12"/>
									<line x1="12" y1="16" x2="12.01" y2="16"/>
								</svg>
								<span class="gallery-manager__pending-label">{ps.label ?? 'Failed'}</span>
								{#if ps.onretry}
									<button
										type="button"
										class="gallery-manager__pending-retry"
										onclick={(e) => { e.stopPropagation(); ps.onretry?.(); }}
									>
										Retry
									</button>
								{/if}
							{:else if ps.status === 'queued'}
								<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
									<circle cx="12" cy="12" r="10"/>
									<polyline points="12 6 12 12 16 14"/>
								</svg>
								<span class="gallery-manager__pending-label">{ps.label ?? 'Queued…'}</span>
							{:else}
								<span class="gallery-manager__pending-spinner" aria-hidden="true"></span>
								<span class="gallery-manager__pending-label">{ps.label ?? (ps.status === 'processing' ? 'Processing…' : 'Uploading…')}</span>
								{#if typeof ps.progress === 'number'}
									<span class="gallery-manager__pending-progress">{ps.progress}%</span>
								{/if}
							{/if}
						</div>
					{/if}

					{#if isSelected(image.id)}
						<span class="gallery-manager__select-check" aria-hidden="true">
							<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round">
								<polyline points="20 6 9 17 4 12"/>
							</svg>
						</span>
					{/if}

					{#if canReorder}
						<span class="gallery-manager__handle" title="Drag to reorder or onto cover slot" aria-hidden="true">
							<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
								<line x1="4" y1="8" x2="20" y2="8"/>
								<line x1="4" y1="16" x2="20" y2="16"/>
							</svg>
						</span>
					{/if}

					{#if image.id === coverImageId}
						<span class="gallery-manager__badge">Cover</span>
					{/if}

					<div class="gallery-manager__actions">
						{#if canSetCover && image.id !== coverImageId}
							<button
								class="gallery-manager__btn"
								onclick={(e) => { e.stopPropagation(); setCover(image); }}
								title="Set as cover"
								type="button"
							>
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
									<path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
								</svg>
							</button>
						{/if}
						{#if canDelete && removeMode === 'unlink'}
							<button
								class="gallery-manager__btn"
								onclick={(e) => { e.stopPropagation(); ondelete?.(image); }}
								title="Remove"
								type="button"
							>
								<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
									<line x1="6" y1="6" x2="18" y2="18"/>
									<line x1="6" y1="18" x2="18" y2="6"/>
								</svg>
							</button>
						{:else if canDelete}
							{@const confirming = confirmingDeleteId === image.id}
							<button
								class="gallery-manager__btn gallery-manager__btn--danger"
								class:is-confirming={confirming}
								onclick={(e) => { e.stopPropagation(); handleDeleteClick(image); }}
								title={confirming ? 'Click again to confirm' : 'Delete'}
								type="button"
							>
								{#if confirming}
									<span class="gallery-manager__confirm-label">Confirm?</span>
								{:else}
									<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
										<path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
										<path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
									</svg>
								{/if}
							</button>
						{/if}
					</div>

					{#if canReorder}
						<span class="gallery-manager__order">{index + 1}</span>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>

{#if confirmDialog}
	{@render confirmDialog({ open: bulkConfirmOpen, count: selectionCount(), onconfirm: handleBulkDeleteConfirm, oncancel: handleBulkDeleteCancel })}
{:else if bulkConfirmOpen}
	<div class="gallery-manager__modal-overlay" onclick={handleBulkDeleteCancel} role="presentation">
		<div
			class="gallery-manager__modal"
			role="dialog"
			aria-modal="true"
			aria-labelledby="bulk-delete-title"
			tabindex="-1"
			onclick={(e) => e.stopPropagation()}
			onkeydown={(e) => { if (e.key === 'Escape') handleBulkDeleteCancel(); }}
		>
			<h3 class="gallery-manager__modal-title" id="bulk-delete-title">
				Delete {selectionCount()} image{selectionCount() === 1 ? '' : 's'}?
			</h3>
			<p class="gallery-manager__modal-text">
				This permanently deletes the selected image{selectionCount() === 1 ? '' : 's'} from this property — removed from every album and the cover, and the original file reclaimed when no other property uses it. This cannot be undone.
			</p>
			<div class="gallery-manager__modal-actions">
				<button type="button" class="gallery-manager__bulk-btn" onclick={handleBulkDeleteCancel}>Cancel</button>
				<button type="button" class="gallery-manager__bulk-btn gallery-manager__bulk-btn--danger" onclick={handleBulkDeleteConfirm}>
					Delete {selectionCount()}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.gallery-manager {
		position: relative;
		border-radius: var(--radius-md, 8px);
		transition: outline-color 0.15s, background 0.15s;
		outline: 2px dashed transparent;
		outline-offset: 4px;
	}

	.gallery-manager.is-file-dragover {
		outline-color: var(--color-accent, var(--br-color-primary, #6366f1));
		background: color-mix(in srgb, var(--color-accent, var(--br-color-primary, #6366f1)) 6%, transparent);
	}

	.gallery-manager__grid {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	/* Even-column mode (opt-in via `columns`) — tiles fill N equal columns with no
	   trailing gap, unlike the default fixed-width flex-wrap. */
	.gallery-manager__grid--even {
		display: grid;
		grid-template-columns: repeat(var(--gm-cols), minmax(0, 1fr));
	}

	.gallery-manager__item {
		position: relative;
		border-radius: var(--radius-md, 8px);
		overflow: hidden;
		border: 2px solid transparent;
		transition: border-color 0.15s, opacity 0.15s, transform 0.15s;
		cursor: grab;
	}

	.gallery-manager__item:active {
		cursor: grabbing;
	}

	.gallery-manager__item.is-dragging {
		opacity: 0.4;
	}

	/* Positional drop indicator — a vertical accent line at the insert point (left edge
	   of the tile the drop will precede, or the right edge of the last tile when appending).
	   A pseudo-element on the position:relative tile, so it never touches layout/border and
	   never conflicts with the selected/dragging/cover tile states. The standard "drop here"
	   gap affordance, shared by cross-pane links and internal reorder. */
	.gallery-manager__item.is-insert-before::before,
	.gallery-manager__item.is-insert-after::after {
		content: '';
		position: absolute;
		top: 0;
		bottom: 0;
		width: 3px;
		border-radius: 2px;
		background: var(--color-accent, var(--br-color-primary, #6366f1));
		box-shadow: 0 0 8px color-mix(in srgb, var(--color-accent, var(--br-color-primary, #6366f1)) 55%, transparent);
		z-index: 6;
		pointer-events: none;
	}
	.gallery-manager__item.is-insert-before::before {
		left: -0.5rem;
	}
	.gallery-manager__item.is-insert-after::after {
		right: -0.5rem;
	}

	.gallery-manager__item.is-cover {
		border-color: var(--color-cover, var(--color-success, #22c55e));
	}

	.gallery-manager__item.is-selected {
		border-color: var(--color-accent, var(--br-color-primary, #6366f1));
		box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent, var(--br-color-primary, #6366f1)) 35%, transparent);
	}

	/* Dimmed — e.g. a source-pane image already present in a linked target pane.
	   Stays draggable + clickable; the dimming is a status cue, not a disable. */
	.gallery-manager__item.is-muted {
		opacity: 0.45;
	}

	.gallery-manager__item:focus-visible {
		outline: 2px solid var(--color-accent, var(--br-color-primary, #6366f1));
		outline-offset: 2px;
	}

	.gallery-manager__select-check {
		position: absolute;
		top: 0.5rem;
		right: 0.5rem;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 50%;
		background: var(--color-accent, var(--br-color-primary, #6366f1));
		color: #fff;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
		pointer-events: none;
	}

	/* Bulk-action toolbar — appears above the grid only when 1+ items selected. */
	.gallery-manager__bulk-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--br-space-md, 0.75rem);
		padding: 0.5rem 0.75rem;
		margin-bottom: 0.75rem;
		background: var(--color-accent, var(--br-color-primary, #6366f1));
		color: #fff;
		border-radius: var(--radius-md, 8px);
		font-size: 0.875rem;
	}

	.gallery-manager__bulk-count {
		font-weight: 600;
	}

	.gallery-manager__bulk-actions {
		display: flex;
		gap: 0.5rem;
	}

	.gallery-manager__bulk-btn {
		padding: 0.4rem 0.9rem;
		background: rgba(255, 255, 255, 0.18);
		color: #fff;
		border: 1px solid rgba(255, 255, 255, 0.3);
		border-radius: var(--radius-sm, 4px);
		font-size: 0.8125rem;
		font-weight: 500;
		cursor: pointer;
	}

	.gallery-manager__bulk-btn:hover {
		background: rgba(255, 255, 255, 0.28);
	}

	.gallery-manager__bulk-btn--danger {
		background: var(--color-error, #ef4444);
		border-color: var(--color-error, #ef4444);
	}

	.gallery-manager__bulk-btn--danger:hover {
		background: color-mix(in srgb, var(--color-error, #ef4444) 85%, black);
	}

	/* Bulk confirm modal — body-portaled visual; uses fixed positioning. */
	.gallery-manager__modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 300;
	}

	.gallery-manager__modal {
		background: var(--color-surface, var(--br-color-surface, #ffffff));
		color: var(--color-text, var(--br-color-text, #111));
		border-radius: var(--radius-lg, 12px);
		padding: 1.5rem;
		max-width: 420px;
		width: 90%;
		box-shadow: var(--br-shadow-lg, 0 10px 25px rgba(0,0,0,0.15));
	}

	.gallery-manager__modal-title {
		font-family: var(--br-font-heading, inherit);
		font-size: 1.125rem;
		font-weight: 700;
		margin: 0 0 0.5rem 0;
	}

	.gallery-manager__modal-text {
		font-size: 0.875rem;
		color: var(--br-color-neutral-600, #666);
		line-height: 1.5;
		margin: 0 0 1.25rem 0;
	}

	.gallery-manager__modal-actions {
		display: flex;
		justify-content: flex-end;
		gap: 0.5rem;
	}

	.gallery-manager__modal .gallery-manager__bulk-btn {
		background: transparent;
		color: var(--br-color-neutral-700, #444);
		border-color: var(--br-border-color, #ddd);
	}

	.gallery-manager__modal .gallery-manager__bulk-btn--danger {
		background: var(--color-error, var(--br-color-error, #ef4444));
		color: #fff;
		border-color: var(--color-error, var(--br-color-error, #ef4444));
	}

	.gallery-manager__badge {
		position: absolute;
		top: 0.375rem;
		left: 0.375rem;
		padding: 0.125rem 0.375rem;
		font-size: 0.625rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		background: var(--color-cover, var(--color-success, #22c55e));
		color: #fff;
		border-radius: var(--radius-sm, 4px);
	}

	.gallery-manager__actions {
		position: absolute;
		top: 0.375rem;
		right: 0.375rem;
		display: flex;
		gap: 0.25rem;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.gallery-manager__item:hover .gallery-manager__actions {
		opacity: 1;
	}

	.gallery-manager__btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border-radius: 4px;
		border: none;
		background: rgba(0, 0, 0, 0.6);
		color: #fff;
		cursor: pointer;
		transition: background 0.15s;
	}

	.gallery-manager__btn:hover {
		background: rgba(0, 0, 0, 0.85);
	}

	.gallery-manager__btn--danger:hover {
		background: var(--color-error, #ef4444);
	}

	.gallery-manager__btn.is-confirming {
		width: auto;
		padding: 0 0.6rem;
		background: var(--color-error, #ef4444);
		font-weight: 600;
	}

	.gallery-manager__confirm-label {
		font-size: 0.6875rem;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #fff;
		white-space: nowrap;
	}

	.gallery-manager__order {
		position: absolute;
		bottom: 0.375rem;
		left: 0.375rem;
		width: 20px;
		height: 20px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.625rem;
		font-weight: 700;
		background: rgba(0, 0, 0, 0.6);
		color: #fff;
		border-radius: 50%;
	}

	.gallery-manager__empty {
		color: var(--color-text-muted, #888);
		font-size: 0.875rem;
		text-align: center;
		padding: 2rem;
	}

	.gallery-manager__empty--droppable {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 3rem 2rem;
		border: 2px dashed var(--color-border, var(--br-border-color, #ccc));
		border-radius: var(--radius-md, 8px);
		color: var(--color-text-muted, #888);
	}

	.gallery-manager__empty--droppable p {
		margin: 0;
		font-size: 0.875rem;
	}

	.gallery-manager__empty--clickable {
		cursor: pointer;
		transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;
	}
	.gallery-manager__empty--clickable:hover,
	.gallery-manager__empty--clickable:focus-visible {
		border-color: var(--color-primary, var(--br-color-primary, #0d4f4f));
		color: var(--color-primary, var(--br-color-primary, #0d4f4f));
		background: color-mix(in srgb, var(--color-primary, var(--br-color-primary, #0d4f4f)) 4%, transparent);
		outline: none;
	}

	/* Per-tile pending overlay — sits over the ImageCard while processing /
	   uploading is in flight. Dimmed backdrop + centered spinner + label.
	   Pointer-events relaxed on the error variant so the Retry button works. */
	.gallery-manager__pending {
		position: absolute;
		inset: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 0.4rem;
		background: rgba(0, 0, 0, 0.62);
		color: #fff;
		font-size: 0.75rem;
		font-weight: 500;
		text-align: center;
		padding: 0.5rem;
		pointer-events: none;
		z-index: 4;
	}
	.gallery-manager__pending.is-error {
		background: rgba(192, 57, 43, 0.78);
		pointer-events: auto;
	}
	.gallery-manager__pending.is-queued {
		background: rgba(0, 0, 0, 0.45);
		color: rgba(255, 255, 255, 0.85);
	}
	.gallery-manager__pending-spinner {
		width: 22px;
		height: 22px;
		border: 2px solid rgba(255, 255, 255, 0.35);
		border-top-color: #fff;
		border-radius: 50%;
		animation: gm-spin 0.9s linear infinite;
	}
	@keyframes gm-spin { to { transform: rotate(360deg); } }
	.gallery-manager__pending-label {
		font-size: 0.7rem;
		line-height: 1.2;
		max-width: 100%;
		word-break: break-word;
	}
	.gallery-manager__pending-progress {
		font-variant-numeric: tabular-nums;
		font-size: 0.7rem;
		padding: 1px 6px;
		background: rgba(255, 255, 255, 0.18);
		border-radius: 999px;
	}
	.gallery-manager__pending-retry {
		margin-top: 0.25rem;
		padding: 0.15rem 0.6rem;
		font-size: 0.7rem;
		font-weight: 600;
		color: #c0392b;
		background: #fff;
		border: none;
		border-radius: 3px;
		cursor: pointer;
	}
	.gallery-manager__pending-retry:hover {
		background: rgba(255, 255, 255, 0.92);
	}

	.gallery-manager__handle {
		position: absolute;
		top: 0.375rem;
		left: 0.375rem;
		display: flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: 4px;
		background: rgba(0, 0, 0, 0.55);
		color: #fff;
		opacity: 0;
		pointer-events: none;
		transition: opacity 0.15s;
	}

	.gallery-manager__item:hover .gallery-manager__handle {
		opacity: 1;
	}
</style>
