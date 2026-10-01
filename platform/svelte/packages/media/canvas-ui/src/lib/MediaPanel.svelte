<script lang="ts">
	// Media panel (E3/E4 + G6b/G6c) — the image side of the attached sources. Media-type tabs
	// (Images live; Video/Audio declared-but-disabled, NO "coming soon") sit over the shared
	// <ImageSourcePicker> ([Sources ▾] filter + SourceImageGrid). Clicking a thumbnail links the
	// selected image layer's src to the positional token images.<group>.<index>.url on the
	// chosen source's alias (multi-source binding). The zero-sources case is panel-specific
	// (the "attach a source" CTA via EmptyState's shared actions row), so it lives here; the
	// selector+grid is delegated.
	import type { Snippet } from 'svelte';
	import { Button } from '@sbx/core-ui/components/primitives';
	import type { ImageSourceView } from './editor-state.svelte.js';
	import Panel from './Panel.svelte';
	import EmptyState from './EmptyState.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import SourceImageGrid from './SourceImageGrid.svelte';
	import ImageSourcePicker from './ImageSourcePicker.svelte';
	import ThumbImage from './ThumbImage.svelte';
	import { IMAGE_BIND_MIME, type ImageBindDrag } from './dnd.js';

	interface Props {
		/** Attached, image-capable sources (editor.imageSources()). */
		sources?: ImageSourceView[];
		/** The selected layer can receive an image link (i.e. it is an image layer). */
		canBind?: boolean;
		/** The selected layer's bound source alias — the live slot highlights only while that
		 *  source is the one being viewed. */
		boundAlias?: string;
		/** Token bound to the selected layer's src. */
		boundToken?: string;
		/** Link the selected layer's src to a positional token on the given source alias. */
		onbind?: (token: string, alias: string) => void;
		/** Place a NEW image layer (bound to the token) in the page centre — fired when a
		 *  thumbnail is clicked with no image layer selected (Canva-style instant drop). */
		oninsertimage?: (token: string, alias: string) => void;
		/** The page background's CURRENT image face (resolved src) — undefined while the
		 *  background is a plain color. Drives the Background section's thumb vs CTA. */
		backgroundSrc?: string;
		/** The background's bound source alias + token — highlights the current image
		 *  inside the inline change picker (same live-slot ring as the main grid). */
		bgBoundAlias?: string;
		bgBoundToken?: string;
		/** Set the PAGE BACKGROUND to a source image (the shell confirms when replacing
		 *  a user-set background — user directive 2026-06-06). */
		onsetbackground?: (token: string, alias: string) => void;
		/** Open the attach-source flow (when nothing is attached). */
		onattach?: () => void;
		/** Optional consumer-owned uploader rendered at the TOP of the Images view (e.g. BR's
		 *  "Upload watermark" ImagePicker → the watermark library). The kit stays agnostic: it
		 *  only hosts the slot; the consumer owns the upload + which pool it lands in. */
		uploadSlot?: Snippet;
	}

	let { sources = [], canBind = false, boundAlias, boundToken, onbind, oninsertimage, backgroundSrc, bgBoundAlias, bgBoundToken, onsetbackground, onattach, uploadSlot }: Props = $props();

	// Background section state — the inline picker open/closed + drag-over highlight.
	let pickingBg = $state(false);
	let bgDragOver = $state(false);

	/** A grid thumb dragged onto the Background section sets the background (same
	 *  G8 payload as the canvas drop matrix — shape-guarded the same way). */
	function onBgDrop(event: DragEvent): void {
		bgDragOver = false;
		const raw = event.dataTransfer?.getData(IMAGE_BIND_MIME);
		if (!raw) return;
		event.preventDefault();
		try {
			const payload = JSON.parse(raw) as ImageBindDrag;
			if (typeof payload.token === 'string' && typeof payload.alias === 'string') {
				onsetbackground?.(payload.token, payload.alias);
			}
		} catch {
			/* foreign payload — ignore */
		}
	}

	function onBgDragOver(event: DragEvent): void {
		if (!event.dataTransfer?.types.includes(IMAGE_BIND_MIME)) return;
		event.preventDefault();
		event.dataTransfer.dropEffect = 'copy';
		bgDragOver = true;
	}

	// Media shows PHOTOGRAPHIC imagery only — derived map renders (ImageCategory.role
	// 'map') live exclusively in the Map tab (E-1). Strip map groups from each source
	// view; the grid and its counts are category-driven, so totals self-correct. A
	// source left with no non-map groups still lists (its grid reads honestly empty).
	const mediaSources = $derived(
		sources.map((s) => ({ ...s, categories: s.categories.filter((c) => c.role !== 'map') }))
	);

	/** Total images across a source's (non-map) groups. */
	function totalImagesFor(s: ImageSourceView): number {
		return s.categories.reduce((n, c) => n + (s.images[c.id] ?? []).length, 0);
	}
	// PS-1 (2026-06-15): one block PER attached source (was a single [Sources ▾] dropdown) —
	// so you drag a thumbnail from the right source's block and the placed layer is bound to
	// THAT source, no post-drop re-pointing. Sources with no images are omitted entirely.
	const mediaBlocks = $derived(mediaSources.filter((s) => totalImagesFor(s) > 0));

	// Only Images is live; Video/Audio are declared so the IA is honest, but disabled with
	// no placeholder copy (never "coming soon").
	const MEDIA_TYPES = [
		{ id: 'images', label: 'Images', enabled: true },
		{ id: 'video', label: 'Video', enabled: false },
		{ id: 'audio', label: 'Audio', enabled: false }
	] as const;
	let activeType = $state<'images' | 'video' | 'audio'>('images');
</script>

<!-- Text-only CTA in EmptyState's shared right-aligned actions row (single chrome
     definition — see EmptyState). -->
{#snippet attachAction()}
	<Button variant="secondary" size="sm" onclick={() => onattach?.()}>Attach a source</Button>
{/snippet}

<Panel title="Media">
	<div class="mtabs" role="tablist" aria-label="Media type">
		{#each MEDIA_TYPES as t (t.id)}
			<button
				type="button"
				class="mtab"
				class:mtab--active={activeType === t.id}
				role="tab"
				aria-selected={activeType === t.id}
				disabled={!t.enabled}
				onclick={() => t.enabled && (activeType = t.id)}
			>
				{t.label}
			</button>
		{/each}
	</div>

	{#if activeType === 'images'}
		{#if uploadSlot}
			<!-- Consumer-owned uploader (BR: watermark library). Top of the Images view so a
			     freshly uploaded asset appears in the source blocks just below it. -->
			<div class="upload-sec">{@render uploadSlot()}</div>
		{/if}
		{#if sources.length === 0}
			<EmptyState
				text="Attach a source in Sources to browse and link its images."
				actions={onattach ? attachAction : undefined}
			/>
		{:else}
			{#if onsetbackground}
				<!-- Background image section (user directive 2026-06-06): the current
				     background image (click to change) or a set-CTA dropzone; grid thumbs
				     drag in (same G8 payload), or pick via the inline shared picker. -->
				<div
					class="bgsec"
					class:bgsec--over={bgDragOver}
					ondragover={onBgDragOver}
					ondragleave={() => (bgDragOver = false)}
					ondrop={onBgDrop}
					role="group"
					aria-label="Page background image"
				>
					<span class="bgsec__label">Background</span>
					{#if pickingBg}
						<ImageSourcePicker
							sources={mediaSources}
							canBind={true}
							boundAlias={bgBoundAlias}
							boundToken={bgBoundToken}
							onbind={(token, alias) => {
								onsetbackground?.(token, alias);
								pickingBg = false;
							}}
						/>
						<span class="bgsec__actions">
							<button type="button" class="bgsec__btn" onclick={() => (pickingBg = false)}>Cancel</button>
						</span>
					{:else if backgroundSrc}
						<button
							type="button"
							class="bgsec__thumb"
							title="Change the background image"
							onclick={() => (pickingBg = true)}
						>
							<ThumbImage src={backgroundSrc} label="Page background" />
						</button>
					{:else}
						<button type="button" class="bgsec__cta" onclick={() => (pickingBg = true)}>
							Set background image — click or drag an image here
						</button>
					{/if}
				</div>
			{/if}
			<!-- PS-1: one block per attached source — drag a thumbnail from the right source's
			     block to place a layer bound to THAT source (G8). The Inspector image-bind popup
			     keeps the compact [Sources ▾] picker (ImageSourcePicker); the rail shows blocks. -->
			{#if mediaBlocks.length === 0}
				<EmptyState text="None of the attached sources has images yet. Add images to a source and they’ll appear here automatically." />
			{:else}
				{#each mediaBlocks as src (src.alias)}
					<CollapsibleSection title={`${src.label} · ${src.title}`} count={totalImagesFor(src)}>
						<SourceImageGrid
							categories={src.categories}
							images={src.images}
							{canBind}
							boundToken={src.alias === boundAlias ? boundToken : undefined}
							draggable
							alias={src.alias}
							onbind={(token) => onbind?.(token, src.alias)}
							oninsert={(token) => oninsertimage?.(token, src.alias)}
						/>
					</CollapsibleSection>
				{/each}
			{/if}
		{/if}
	{/if}
</Panel>

<style>
	/* Consumer-owned uploader slot — spacing only; the consumer styles its own control. */
	.upload-sec {
		margin-bottom: var(--cv-space-md, 0.75rem);
	}

	/* Media-type tabs — segmented row; disabled types read as inert, not "coming soon". */
	.mtabs {
		display: flex;
		gap: 0.25rem;
		margin-bottom: var(--cv-space-md, 1rem);
	}

	.mtab {
		flex: 1;
		padding: 0.375rem 0.5rem;
		font: inherit;
		font-size: 0.8125rem;
		color: var(--cv-color-neutral-500, #707070);
		background: var(--cv-color-neutral-100, #f5f5f5);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		cursor: pointer;
	}

	.mtab--active {
		color: #fff;
		background: var(--cv-color-primary, #333333);
		border-color: var(--cv-color-primary, #333333);
	}

	.mtab:disabled {
		opacity: 0.45;
		cursor: not-allowed;
	}

	/* Background image section — sits above the grid; doubles as a drop target. */
	.bgsec {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
		margin-bottom: var(--cv-space-md, 1rem);
		padding: var(--cv-space-sm, 0.5rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
	}

	.bgsec--over {
		border-color: var(--cv-color-primary, #333333);
		border-style: dashed;
	}

	.bgsec__label {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.04em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	/* The current background image — the whole thumb is the change affordance. */
	.bgsec__thumb {
		display: block;
		width: 100%;
		aspect-ratio: 16 / 10;
		padding: 0;
		overflow: hidden;
		background: var(--cv-color-neutral-100, #f5f5f5);
		border: 2px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		cursor: pointer;
	}

	.bgsec__thumb:hover {
		border-color: var(--cv-color-primary, #333333);
	}

	/* Set-CTA — the whole zone is clickable AND a drop target (dashed = droppable). */
	.bgsec__cta {
		padding: var(--cv-space-md, 1rem) var(--cv-space-sm, 0.5rem);
		font: inherit;
		font-size: 0.75rem;
		color: var(--cv-color-neutral-500, #707070);
		background: transparent;
		border: 1px dashed var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		cursor: pointer;
	}

	.bgsec__cta:hover {
		color: var(--cv-color-primary, #333333);
		border-color: var(--cv-color-primary, #333333);
	}

	/* Picker actions — right-aligned (hard rule). */
	.bgsec__actions {
		display: flex;
		justify-content: flex-end;
	}

	.bgsec__btn {
		border: none;
		background: transparent;
		padding: 0;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--cv-color-primary, #333333);
		cursor: pointer;
	}

	.bgsec__btn:hover {
		text-decoration: underline;
	}
</style>
