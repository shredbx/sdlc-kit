<script lang="ts">
	// Left icon rail (Canva pattern, §11.6 redline 2) — a vertical context switcher
	// that selects which contextual panel the editor shows. Order (IA refactor R1):
	// Document · Sources · Layers · Media · Text · Components · Map. The Settings gear
	// is DISSOLVED — its document-settings + export-defaults moved into the new
	// Document panel; its sources-config grew into the dedicated Sources panel. Each
	// icon swaps the contextual panel (EditorShell holds the active context). Icon
	// names verified against the core-ui Icon registry (IconRegistry.ts).
	//
	// Hover-expand (M-3, tightened D-3): icons alone are hard to guess, so the rail
	// expands on hover (or keyboard :focus-visible) to show each name beside its
	// icon, and collapses the INSTANT the pointer leaves. The expanding surface is
	// an absolutely-positioned OVERLAY above the panel — the outer .rail keeps its
	// 56px footprint, so the canvas never reflows. The visible label IS the
	// button's accessible name (no aria-label/title doubles).
	import { Icon } from '@sbx/core-ui/components/primitives';
	import type { EditorMode } from '@sbx/canvas-kit';

	export type RailContext =
		| 'document'
		| 'sources'
		| 'layers'
		| 'media'
		| 'text'
		| 'components'
		| 'map';

	interface RailItem {
		id: RailContext;
		icon: string;
		label: string;
	}

	interface Props {
		active?: RailContext;
		onselect?: (id: RailContext) => void;
		/** Editor experience (Decision #0298). 'design' (default) shows the full rail;
		 *  'watermark' shows EXACTLY Document · Layers · Media · Text (the live-data
		 *  Sources, Components, and Map items are hidden — branding logos/labels reach
		 *  the design through the Media/Text panels, a later slice). */
		mode?: EditorMode;
		/** Product-level rail hiding (task 2607-033) — items a CONSUMER suppresses on top
		 *  of the mode filter, without changing the mode's feature bundle. BR's Media Canvas
		 *  passes ['map','components'] to hide Map + Shapes while staying in full design mode.
		 *  Applied AFTER the mode filter; unknown ids are ignored. */
		hiddenRailItems?: RailContext[];
	}

	let { active = 'document', onselect, mode = 'design', hiddenRailItems }: Props = $props();

	// One flat group, top→bottom. Icons verified in IconRegistry: the registry has no
	// text-specific glyph, so Text uses `book-text`; Sources uses `database` (bound
	// data records); Document uses `file-text`. The Document panel now hosts BOTH the
	// current-doc properties AND (when a consumer mounts a list) the sibling-documents
	// switcher — so there is no separate 'documents' rail item.
	const ALL_ITEMS: RailItem[] = [
		{ id: 'document', icon: 'file-text', label: 'Document' },
		{ id: 'sources', icon: 'database', label: 'Sources' },
		{ id: 'layers', icon: 'layers', label: 'Layers' },
		{ id: 'media', icon: 'image', label: 'Media' },
		{ id: 'text', icon: 'book-text', label: 'Text' },
		{ id: 'components', icon: 'shapes', label: 'Components' },
		{ id: 'map', icon: 'map', label: 'Map' }
	];

	// Watermark mode keeps the four authoring contexts (a static overlay needs no live
	// sources / shape components / map surface); design mode keeps the full rail.
	const WATERMARK_ITEMS = new Set<RailContext>(['document', 'layers', 'media', 'text']);
	const items = $derived.by(() => {
		const byMode =
			mode === 'watermark' ? ALL_ITEMS.filter((i) => WATERMARK_ITEMS.has(i.id)) : ALL_ITEMS;
		if (!hiddenRailItems?.length) return byMode;
		const hidden = new Set(hiddenRailItems);
		return byMode.filter((i) => !hidden.has(i.id));
	});
</script>

<!-- The outer .rail is the layout spacer (56px, never changes); the inner surface
     is the overlay that widens. Hovering the widened surface still hovers .rail
     (child), so the expansion holds until the pointer truly leaves. -->
<div class="rail">
	<nav class="rail__surface" aria-label="Editor sections">
		{#each items as item (item.id)}
			<button
				class="rail__item"
				class:rail__item--active={active === item.id}
				type="button"
				aria-pressed={active === item.id}
				onclick={() => onselect?.(item.id)}
			>
				<span class="rail__icon"><Icon name={item.icon} size="md" /></span>
				<span class="rail__label">{item.label}</span>
			</button>
		{/each}
	</nav>
</div>

<style>
	.rail {
		position: relative;
		width: 56px;
		min-width: 56px;
	}

	/* The expanding overlay. Collapsed it is exactly the old rail (56px, flat);
	   expanded it widens over the panel with a soft elevation shadow. Opening keeps
	   a ~140ms hover-intent delay (mousing past doesn't flicker it open); closing is
	   IMMEDIATE — the moment the pointer leaves, the overlay gets out of the way
	   (D-3; a lingering overlay occludes the panel's left column). Keyboard expansion
	   uses :focus-visible, NOT :focus-within: a mouse click parks focus on the rail
	   item, and :focus-within would hold the rail open forever after it (the live
	   "won't collapse" bug) — Tab still expands for a11y. overflow:hidden crops the
	   surface to its current width mid-transition; at rest the expanded width fits
	   the longest label in full (nowrap, no ellipsis — labels are never truncated). */
	.rail__surface {
		position: absolute;
		inset-block: 0;
		left: 0;
		z-index: 20;
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-xs, 0.25rem);
		width: 56px;
		padding: var(--cv-space-sm, 0.5rem);
		overflow: hidden;
		/* Chrome theme contract (design pass A): --cv-chrome-* first, original light
		   chain as the fallback — see TopToolbar for the token set. */
		background: var(--cv-chrome-bg, var(--cv-color-neutral-100, #f5f5f5));
		border-right: 1px solid var(--cv-chrome-border, var(--cv-border-color, #e0e0e0));
		/* Collapse is INSTANT (no width transition on the resting state): the expanded
		   184px overlay sits ABOVE the adjacent contextual panel (z-index:20), so an
		   animated collapse left it briefly covering the panel's left column — the layer
		   rows were unclickable mid-fade ("can't select the layer"). Snapping shut the
		   instant the pointer leaves the rail keeps the overlay off the panel's hit area.
		   The EXPAND still animates (the rule below restores the transition). */
		transition: none;
	}

	.rail:hover .rail__surface,
	.rail:has(:focus-visible) .rail__surface {
		width: 184px;
		box-shadow: 4px 0 16px rgba(0, 0, 0, 0.08);
		/* Animate only the OPEN direction (with the hover-intent delay) — see above. */
		transition:
			width 0.18s ease 0.14s,
			box-shadow 0.18s ease 0.14s;
	}

	/* A row: fixed 40px icon box (identical to the collapsed-only rail) + the name.
	   Collapsed, the row is exactly the old 40×40 icon button; expanded, the hover/
	   active background stretches across the full row. */
	.rail__item {
		display: flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		width: 100%;
		height: 40px;
		padding: 0;
		border: none;
		border-radius: var(--cv-radius-md, 0.5rem);
		background: transparent;
		color: var(--cv-chrome-fg, var(--cv-color-neutral-500, #707070));
		cursor: pointer;
		transition:
			background 0.15s ease,
			color 0.15s ease;
	}

	.rail__icon {
		display: flex;
		align-items: center;
		justify-content: center;
		flex: 0 0 40px;
		width: 40px;
		height: 40px;
	}

	.rail__label {
		font-size: 0.8125rem;
		white-space: nowrap;
		opacity: 0;
		transform: translateX(-4px);
		transition:
			opacity 0.15s ease,
			transform 0.15s ease;
	}

	.rail:hover .rail__label,
	.rail:has(:focus-visible) .rail__label {
		opacity: 1;
		transform: translateX(0);
		transition-delay: 0.14s;
	}

	.rail__item:hover {
		background: var(--cv-chrome-hover-bg, var(--cv-color-neutral-200, #e8e8e8));
		color: var(--cv-chrome-fg-strong, var(--cv-color-neutral-800, #202020));
	}

	.rail__item--active {
		background: var(--cv-chrome-active-bg, var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08)));
		color: var(--cv-chrome-active-fg, var(--cv-color-primary, #333333));
	}

	.rail__item:focus-visible {
		outline: 2px solid var(--cv-chrome-active-fg, var(--cv-color-primary, #333333));
		outline-offset: -2px;
	}

	/* Reduced motion: the expansion still works, just without animation. */
	@media (prefers-reduced-motion: reduce) {
		.rail__surface,
		.rail__label {
			transition: none;
		}
	}
</style>
