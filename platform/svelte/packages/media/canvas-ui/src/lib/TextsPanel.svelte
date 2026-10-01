<script lang="ts">
	// Text context panel (§11.6, redline 2) — the text side of sources. Sections
	// (in order):
	//   Text             grid of text presets (Heading · Subheading · Body · Text box) —
	//                    uses the SHARED ComponentGrid/ComponentCard; click inserts at
	//                    the page centre, drag places at the drop point (D-2)
	//   Droppable fields bindable field-chips of the Active source (slice 4a + D-1) —
	//                    clicking a chip binds the selected text element's content to
	//                    that field; dragging it onto the canvas places a new bound
	//                    text layer (or rebinds the text layer it lands on) — chips
	//                    are never inert, so they are never disabled
	//   Readable content read-only longtext blocks (Copy + Insert-as-text-block — 4c)
	// element = layer.
	import Panel from './Panel.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import EmptyState from './EmptyState.svelte';
	import ComponentGrid from './ComponentGrid.svelte';
	import ComponentCard from './ComponentCard.svelte';
	import { Icon } from '@sbx/core-ui/components/primitives';
	import type { FieldDescriptor } from '@sbx/canvas-kit';
	import { presetsForSection } from './palette.js';
	import { FIELD_BIND_MIME, startPresetDrag, type FieldBindDrag } from './dnd.js';

	interface Props {
		/** Insert a text preset — a card click ARMS the text tool (placed where you click/
		 *  draw on the canvas); the shell owns the placement mode. */
		oninsert?: (presetId: string) => void;
		/** The armed placement tool's preset id — the matching text card reads as selected. */
		activeTool?: string | null;
		/** Per-source field groups (PS-1) — one entry per attached source that declares
		 *  ≥1 text field. Replaces the old single `fields`/`sourceLabel`/`sourceAlias`
		 *  trio. Each group's alias is used for chip drag/bind so a drop places a layer
		 *  already bound to THAT source (no post-drop re-pointing). */
		fieldGroups?: { alias: string; label: string; title: string; fields: FieldDescriptor[] }[];
		/** Field groups of the BUILT-IN sources (#0286 — e.g. Branding's "Company
		 *  name"). Each renders as its own "From {label}" chip group; chips bind with
		 *  the group's alias. */
		builtinGroups?: { alias: string; label: string; fields: FieldDescriptor[] }[];
		/** True when a text element is selected (chips bind to its content). */
		canBind?: boolean;
		/** The selected text element's content binding (source alias + token) — the
		 *  matching chip reads as active and is scrolled into view. */
		boundAlias?: string;
		boundToken?: string;
		/** Bind the selected text element's content to a field — `alias` is set for
		 *  all group chips so a click always targets the correct source. */
		onbindfield?: (field: FieldDescriptor, alias?: string) => void;
	}

	let {
		oninsert,
		activeTool,
		fieldGroups = [],
		builtinGroups = [],
		canBind = false,
		boundAlias,
		boundToken,
		onbindfield
	}: Props = $props();

	/** The chip for `field` in the group bound to `alias` is the one the selected
	 *  layer's content links to. Every chip group now carries an explicit alias
	 *  (from fieldGroups or builtinGroups), so this comparison is always exact. */
	function isBoundChip(field: FieldDescriptor, alias: string): boolean {
		return !!boundToken && boundToken === field.token && boundAlias === alias;
	}

	// Reveal the bound chip whenever the selection's binding changes — and on mount,
	// so the Inspector's ⌖ reveal lands with the linked chip in view (E-2).
	let fieldsEl = $state<HTMLDivElement | null>(null);
	$effect(() => {
		void boundToken;
		void boundAlias;
		fieldsEl?.querySelector('.chip--active')?.scrollIntoView({ block: 'nearest' });
	});

	const textPresets = presetsForSection('text');

	/** Begin a chip drag (D-1) — the whole FieldDescriptor travels (CanvasStage binds
	 *  without a registry lookup); same copy-drag grammar as the image thumbnails. */
	function onChipDragStart(event: DragEvent, field: FieldDescriptor, alias?: string): void {
		if (!event.dataTransfer) return;
		const payload: FieldBindDrag = { field, alias };
		event.dataTransfer.setData(FIELD_BIND_MIME, JSON.stringify(payload));
		event.dataTransfer.effectAllowed = 'copy';
	}

	// Hover tooltip mirrors the Media/Map card guidance — chips stay drag-enabled with
	// no text element selected (a drop places a NEW bound text layer).
	const chipTip = $derived(
		canBind
			? 'Click to bind selected text · drag onto the canvas to place'
			: 'Drag onto the canvas to place · select a text element to bind by click'
	);
</script>

{#snippet chips(chipFields: FieldDescriptor[], alias: string)}
	<div class="chips">
		{#each chipFields as field (field.token)}
			{@const active = isBoundChip(field, alias)}
			<button
				type="button"
				class="chip"
				class:chip--active={active}
				draggable={true}
				aria-pressed={active}
				title={`${field.label} — ${chipTip}`}
				onclick={() => canBind && onbindfield?.(field, alias)}
				ondragstart={(e) => onChipDragStart(e, field, alias)}
			>
				<Icon name="tag" size="xs" />
				<span>{field.label}</span>
			</button>
		{/each}
	</div>
{/snippet}

<Panel title="Text" searchPlaceholder="Search text…">
	<CollapsibleSection title="Text">
		<ComponentGrid cols={2}>
			{#each textPresets as preset (preset.id)}
				<ComponentCard
					label={preset.label}
					icon={preset.icon}
					selected={activeTool === preset.id}
					onclick={() => oninsert?.(preset.id)}
					ondragstart={(e) => startPresetDrag(e, preset.id)}
				/>
			{/each}
		</ComponentGrid>
	</CollapsibleSection>

	<CollapsibleSection title="Droppable fields">
		{#if fieldGroups.length === 0 && builtinGroups.length === 0}
			<EmptyState text="Attach a source in Settings to bind fields." />
		{:else}
			<div bind:this={fieldsEl}>
				{#each fieldGroups as group (group.alias)}
					<CollapsibleSection title={`${group.label} · ${group.title}`}>
						{@render chips(group.fields, group.alias)}
					</CollapsibleSection>
				{/each}
				{#each builtinGroups as group (group.alias)}
					<p class="fields__from">From {group.label}</p>
					{@render chips(group.fields, group.alias)}
				{/each}
				{#if !canBind}
					<p class="fields__hint">Drag a chip onto the canvas — or select a text element to bind by click.</p>
				{/if}
			</div>
		{/if}
	</CollapsibleSection>

	<CollapsibleSection title="Readable content">
		<EmptyState text="Attach a source in Settings to read its content." />
	</CollapsibleSection>
</Panel>

<style>
	.fields__from {
		margin: 0 0 var(--cv-space-sm, 0.5rem);
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	/* Stacked groups (active source + built-ins) breathe between each other. */
	.chips + .fields__from {
		margin-top: var(--cv-space-md, 0.75rem);
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: var(--cv-space-xs, 0.25rem);
	}

	/* Always grabbable (drag works with nothing selected) — same affordance as the
	   Media/Map cards; click-bind still works whenever a text element is selected. */
	.chip {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.25rem 0.5rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: 999px;
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-neutral-700, #383838);
		font-size: 0.75rem;
		cursor: grab;
	}

	.chip:active {
		cursor: grabbing;
	}

	.chip:hover {
		border-color: var(--cv-color-primary, #333333);
		color: var(--cv-color-primary, #333333);
	}

	/* The chip the selected layer's content is bound to (E-2) — mirrors the
	   Media/Map bound-card highlight. */
	.chip--active {
		border-color: var(--cv-color-primary, #333333);
		color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.fields__hint {
		margin: var(--cv-space-sm, 0.5rem) 0 0;
		font-size: 0.75rem;
		color: var(--cv-color-neutral-500, #707070);
	}
</style>
