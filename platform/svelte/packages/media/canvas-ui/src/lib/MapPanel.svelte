<script lang="ts">
	// Map panel (T5 / M-2 rewrite) — the Map tab of the left rail. Two sections:
	//
	//   1. Components  — a ComponentGrid of 4 square cards: Map · Pin · Outline · Callout.
	//                    Map is always enabled (inserts/drags anywhere on the page).
	//                    Pin / Outline / Callout are DISABLED (R4-gate) until ≥1 map layer
	//                    exists on the active page (`hasMapSurface`). No "coming soon" copy —
	//                    the disabled state with a tooltip is the honest affordance.
	//
	//   2. Sources     — per-source map blocks (PS-1 / R2 filter). One CollapsibleSection per
	//                    attached source that offers ≥1 map-role image group AND has ≥1 card.
	//                    Sources with no location/region map imagery are omitted entirely.
	//                    Cards bind exactly like Media thumbnails (IMAGE_BIND_MIME, G8), so
	//                    CanvasStage's drop matrix applies unchanged.
	//
	// The Components section uses the SAME ComponentGrid/ComponentCard chrome as Shapes/Widgets
	// (zero new card/grid CSS — never-copy-paste-CSS). PRESET_MIME drag connects to the
	// existing onplacepreset drop path in CanvasStage.

	import { Button } from '@sbx/core-ui/components/primitives';
	import type { ResolvedImage } from '@sbx/canvas-kit';
	import { pickImageVariant } from '@sbx/canvas-kit';
	import type { ImageSourceView } from './editor-state.svelte.js';
	import Panel from './Panel.svelte';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import EmptyState from './EmptyState.svelte';
	import ComponentGrid from './ComponentGrid.svelte';
	import ComponentCard from './ComponentCard.svelte';
	import ThumbImage from './ThumbImage.svelte';
	import { presetsForSection } from './palette.js';
	import { IMAGE_BIND_MIME, PRESET_MIME, startPresetDrag, type ImageBindDrag, type PresetDrag } from './dnd.js';

	interface Props {
		/** Attached, image-capable sources (editor.imageSources()). */
		sources?: ImageSourceView[];
		/** The selected layer can receive an image link (i.e. it is an image layer). */
		canBind?: boolean;
		/** The selected layer's bound source alias — the live card highlights only
		 *  within that source's section. */
		boundAlias?: string;
		/** Token bound to the selected layer's src. */
		boundToken?: string;
		/** True when ≥1 map-type layer exists on the active page (R4-gate). Pin/Outline/
		 *  Callout cards are enabled only when this is true. */
		hasMapSurface?: boolean;
		/** The ARMED placement tool's preset id (or null). The matching card reads as
		 *  selected so it's clear which tool the next canvas click will drop. */
		activeTool?: string | null;
		/** Arm a map preset as the active placement tool (the ComponentCard click path):
		 *  the next canvas click drops it at the pointer. Toggles on re-click. */
		oninsert?: (presetId: string) => void;
		/** Link the selected layer's src to a positional token on the given source alias. */
		onbind?: (token: string, alias: string) => void;
		/** Set the PAGE BACKGROUND to this map (the shell confirms before applying). */
		onsetbackground?: (token: string, alias: string, label: string) => void;
		/** Open the attach-source flow. */
		onattach?: () => void;
	}

	let { sources = [], canBind = false, boundAlias, boundToken, hasMapSurface = false, activeTool, oninsert, onbind, onsetbackground, onattach }: Props = $props();

	/** One bindable map card: composed positional token + display label + image. */
	interface MapCard {
		token: string;
		label: string;
		image: ResolvedImage;
	}

	/** Flatten a source's role-marked map groups into cards (R2: only map-role groups). */
	function cardsFor(source: ImageSourceView): MapCard[] {
		const out: MapCard[] = [];
		for (const category of source.categories) {
			if (category.role !== 'map') continue;
			(source.images[category.id] ?? []).forEach((image, index) => {
				out.push({
					token: `images.${category.id}.${index}.url`,
					label: image.alt ?? `${category.label} ${index + 1}`,
					image
				});
			});
		}
		return out;
	}

	// R2: sources filtered to those with ≥1 map-role group AND ≥1 card. Sources with no
	// location/region map imagery are omitted entirely (no "no maps yet" rows per R2).
	const mapSources = $derived(
		sources.filter((s) => {
			if (!s.categories.some((c) => c.role === 'map')) return false;
			return cardsFor(s).length > 0;
		})
	);

	// Map Components grid presets (section:'map').
	const mapPresets = presetsForSection('map');

	// Reveal the bound card whenever the binding changes — and on mount, so the
	// Inspector's ⌖ reveal lands with the linked card in view (E-2).
	let sectionsEl = $state<HTMLDivElement | null>(null);
	$effect(() => {
		void boundToken;
		void boundAlias;
		sectionsEl?.querySelector('.card--active')?.scrollIntoView({ block: 'nearest' });
	});

	/** Begin an image-bind drag (G8) — same MIME + payload as the Media grid. */
	function onCardDragStart(event: DragEvent, token: string, alias: string): void {
		if (!event.dataTransfer) return;
		const payload: ImageBindDrag = { token, alias };
		event.dataTransfer.setData(IMAGE_BIND_MIME, JSON.stringify(payload));
		event.dataTransfer.effectAllowed = 'copy';
	}

	const actionTip = $derived(
		canBind
			? 'Click to link · drag onto the canvas to place'
			: 'Drag onto the canvas to place · select an image layer to link'
	);

	// R4-gate tooltip for disabled annotation cards.
	const gatedTip = 'Add a Map to the page first';
</script>

{#snippet attachAction()}
	<Button variant="secondary" size="sm" onclick={() => onattach?.()}>Attach a source</Button>
{/snippet}

<Panel title="Map" search={false}>
	<!-- 1. Components section — shared ComponentGrid/ComponentCard chrome (R1). -->
	<CollapsibleSection title="Components">
		<ComponentGrid cols={2}>
			{#each mapPresets as preset (preset.id)}
				{@const isMapCard = preset.baseType === 'map'}
				{@const disabled = !isMapCard && !hasMapSurface}
				<!-- Gated cards: the <span title> gives the "Add a Map first" tooltip on MOUSE hover;
				     `ariaLabel` forwards the same explanation to the disabled button so SCREEN READERS
				     announce it too (a disabled button's parent title is never announced — review MED-1). -->
				<span title={disabled ? gatedTip : undefined}>
					<ComponentCard
						label={preset.label}
						icon={preset.icon}
						{disabled}
						selected={activeTool === preset.id}
						ariaLabel={disabled ? `${preset.label} — ${gatedTip}` : undefined}
						onclick={!disabled ? () => oninsert?.(preset.id) : undefined}
						ondragstart={!disabled ? (e) => startPresetDrag(e, preset.id) : undefined}
					/>
				</span>
			{/each}
		</ComponentGrid>
	</CollapsibleSection>

	<!-- 2. Per-source map blocks (PS-1 / R2). -->
	{#if sources.length === 0}
		<EmptyState
			text="Attach a source in Sources to browse and place its maps."
			actions={onattach ? attachAction : undefined}
		/>
	{:else if mapSources.length === 0}
		<!-- EmptyState only when NO attached source offers maps at all (R2: empty-geo sources
		     just don't appear as blocks — this covers zero qualifying sources). -->
		<EmptyState
			text="None of the attached sources has location or region data. Set a location in the property editor and maps will appear here."
			actions={onattach ? attachAction : undefined}
		/>
	{:else}
		<div bind:this={sectionsEl}>
			{#each mapSources as source (source.alias)}
				{@const cards = cardsFor(source)}
				<CollapsibleSection title={`${source.label} · ${source.title}`} count={cards.length}>
					<ul class="cards">
						{#each cards as card (card.token)}
							{@const active = source.alias === boundAlias && card.token === boundToken}
							<li>
								<button
									type="button"
									class="card"
									class:card--active={active}
									draggable={true}
									aria-pressed={active}
									title={`${card.label} — ${actionTip}`}
									onclick={() => canBind && onbind?.(card.token, source.alias)}
									ondragstart={(e) => onCardDragStart(e, card.token, source.alias)}
								>
									<span class="card__media">
										<ThumbImage src={pickImageVariant(card.image, 'thumbnail')} label={card.label} />
									</span>
									<span class="card__label">{card.label}</span>
								</button>
								{#if onsetbackground}
									<span class="card__actions">
										<button
											type="button"
											class="card__bg"
											onclick={() => onsetbackground?.(card.token, source.alias, card.label)}
										>Set as background</button>
									</span>
								{/if}
							</li>
						{/each}
					</ul>
				</CollapsibleSection>
			{/each}
		</div>
	{/if}
</Panel>

<style>
	.cards {
		display: grid;
		grid-template-columns: 1fr;
		gap: var(--cv-space-sm, 0.5rem);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	/* A map card — full-width media frame + a visible label under it (labels wrap,
	   never clipped). Always grabbable (drag works with no layer selected). */
	.card {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		width: 100%;
		padding: 0;
		font: inherit;
		text-align: left;
		background: transparent;
		border: none;
		cursor: grab;
	}

	.card:active {
		cursor: grabbing;
	}

	.card__media {
		position: relative;
		display: block;
		width: 100%;
		aspect-ratio: 16 / 10;
		overflow: hidden;
		background: var(--cv-color-neutral-100, #f5f5f5);
		border: 2px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
	}

	/* Bound highlight — the card linked to the selected layer reads as active. */
	.card--active .card__media {
		border-color: var(--cv-color-primary, #333333);
	}

	/* Keyboard focus ring — cards are tab-navigable buttons (WCAG 2.4.7). */
	.card:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 2px;
	}

	.card__label {
		font-size: 0.75rem;
		line-height: 1.4;
		color: var(--cv-color-neutral-700, #383838);
	}

	/* Per-card background action — right-aligned (hard rule). */
	.card__actions {
		display: flex;
		justify-content: flex-end;
	}

	.card__bg {
		border: none;
		background: transparent;
		padding: 0;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--cv-color-primary, #333333);
		cursor: pointer;
	}

	.card__bg:hover {
		text-decoration: underline;
	}
</style>
