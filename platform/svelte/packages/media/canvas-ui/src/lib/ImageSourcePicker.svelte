<script lang="ts">
	// Shared image-source picker (G6c) — the [Sources ▾] filter + the SourceImageGrid, the
	// reusable middle that backs BOTH the left-rail Media panel AND the Inspector image-bind
	// popup. Extracted so the multi-source selection logic (and the Svelte-5 one-way Select
	// `value`+`onchange` — binding `undefined` to a Select with a fallback throws
	// props_invalid_value) lives in ONE place (never-copy-paste). The caller owns the
	// zero-sources case (the messaging/CTA differs: "attach a source" in the panel vs a bind
	// hint in the Inspector), so this component assumes at least one image source.
	import { Select } from '@sbx/core-ui/components/primitives';
	import type { ImageSourceView } from './editor-state.svelte.js';
	import EmptyState from './EmptyState.svelte';
	import SourceImageGrid from './SourceImageGrid.svelte';

	interface Props {
		/** Attached, image-capable sources (editor.imageSources()). */
		sources?: ImageSourceView[];
		/** A bind target is selected — thumbnails are clickable (else inert + tooltip). */
		canBind?: boolean;
		/** The target's bound source alias — its slot highlights only while that source is viewed. */
		boundAlias?: string;
		/** Token bound to the target's src. */
		boundToken?: string;
		/** Thumbnails are draggable onto the canvas (G8) — forwarded to the grid with the
		 *  selected source's alias so the drop payload is complete. */
		draggable?: boolean;
		/** Link the target to a positional token on the given source alias. */
		onbind?: (token: string, alias: string) => void;
	}

	let { sources = [], canBind = false, boundAlias, boundToken, draggable = false, onbind }: Props = $props();

	// Which source's images are shown. `pickedAlias` is the user's explicit pick (if any);
	// `selected` falls back to the bound source, else the first — always defined while sources
	// exist AND self-repairs if the picked source detaches. (One-way Select value + onchange,
	// NOT bind:value — binding `undefined` to a Select with a fallback throws props_invalid_value.)
	let pickedAlias = $state<string | undefined>(undefined);
	const selected = $derived(
		sources.find((s) => s.alias === pickedAlias) ??
			sources.find((s) => s.alias === boundAlias) ??
			sources[0]
	);
	const sourceOptions = $derived(sources.map((s) => ({ value: s.alias, label: `${s.label} · ${s.title}` })));
	const totalImages = $derived(
		selected ? selected.categories.reduce((n, c) => n + (selected.images[c.id] ?? []).length, 0) : 0
	);
	// Highlight the bound slot only while viewing the source the target is bound to.
	const viewBoundToken = $derived(selected && selected.alias === boundAlias ? boundToken : undefined);
</script>

{#if sources.length > 1}
	<div class="srcfilter">
		<Select
			value={selected?.alias}
			options={sourceOptions}
			label="Source"
			size="sm"
			onchange={(v) => (pickedAlias = v)}
		/>
	</div>
{/if}

{#if totalImages === 0}
	<EmptyState
		text={`This ${selected?.label ?? 'source'} has no images yet. Add images to it and they’ll appear here automatically.`}
	/>
{:else}
	<SourceImageGrid
		categories={selected?.categories ?? []}
		images={selected?.images ?? {}}
		{canBind}
		boundToken={viewBoundToken}
		{draggable}
		alias={selected?.alias}
		onbind={(token) => selected && onbind?.(token, selected.alias)}
	/>
{/if}

<style>
	/* [Sources ▾] filter — sits above the grid when more than one image source is attached. */
	.srcfilter {
		margin-bottom: var(--cv-space-sm, 0.5rem);
	}
</style>
