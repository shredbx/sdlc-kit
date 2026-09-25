<script lang="ts">
	// Shared palette card (Fix 4, §11.6) — the ONE card every insertable palette
	// renders (Text · Shapes · Widgets · Map · Templates · Images). A centered
	// glyph (an Icon, or a thumbnail snippet for template/image variants) with a
	// label below, bordered, with a hover highlight. The card CSS lives ONCE here
	// (never-copy-paste-CSS) — panels compose this, they never re-author card CSS.
	// onclick inserts (page centre); ondragstart makes the card drag-to-place (D-2).
	// Label wraps — never clipped.
	import { Icon } from '@sbx/core-ui/components/primitives';

	interface Props {
		/** Card label (wraps; never truncated). */
		label: string;
		/** core-ui Icon name — used when no `thumb` snippet is given. */
		icon?: string;
		/** Optional thumbnail (template/image variants render a block instead of an Icon). */
		thumb?: import('svelte').Snippet;
		/** Optional overlay (template cards reveal Apply / favourite on hover). */
		overlay?: import('svelte').Snippet;
		/** Marks the card as the current selection (e.g. the applied template). */
		selected?: boolean;
		/** Disable the insert affordance (e.g. geo presets before a Map exists). */
		disabled?: boolean;
		/** Accessible name override — forwarded to the button as `aria-label`. Used for the
		 *  GATED state (a disabled button's parent `title` is hover-only and never announced;
		 *  an aria-label is read even on disabled controls in screen-reader browse mode). When
		 *  omitted the button's visible label is its accessible name. */
		ariaLabel?: string;
		/** Insert affordance (no-op when omitted). */
		onclick?: () => void;
		/** Drag affordance (D-2) — when given, the card is draggable (grab cursor) and
		 *  the panel fills the DataTransfer (e.g. a PresetDrag for drag-to-place). */
		ondragstart?: (event: DragEvent) => void;
	}

	let {
		label,
		icon,
		thumb,
		overlay,
		selected = false,
		disabled = false,
		ariaLabel,
		onclick,
		ondragstart
	}: Props = $props();
</script>

<div class="card-wrap">
	<button
		class="card"
		class:card--selected={selected}
		class:card--thumb={!!thumb}
		class:card--draggable={!!ondragstart}
		type="button"
		{disabled}
		title={label}
		aria-label={ariaLabel}
		draggable={ondragstart && !disabled ? true : undefined}
		onclick={() => onclick?.()}
		{ondragstart}
	>
		<span class="card__media" aria-hidden="true">
			{#if thumb}
				{@render thumb()}
			{:else if icon}
				<Icon name={icon} size="md" />
			{/if}
		</span>
		<span class="card__label">{label}</span>
	</button>

	{#if overlay}
		<div class="card__overlay">{@render overlay()}</div>
	{/if}
</div>

<style>
	/* Positioning context for the hover overlay (Apply / favourite). */
	.card-wrap {
		position: relative;
		display: flex;
	}

	.card {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--cv-space-xs, 0.25rem);
		min-width: 0;
		padding: var(--cv-space-sm, 0.5rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-neutral-50, #fafafa);
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
		transition: border-color 0.15s ease, background 0.15s ease;
	}

	.card:hover:not(:disabled) {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.card:disabled {
		cursor: not-allowed;
		opacity: 0.55;
	}

	/* Drag-enabled cards advertise it (same affordance as the field chips / map cards). */
	.card--draggable:not(:disabled) {
		cursor: grab;
	}

	.card--draggable:not(:disabled):active {
		cursor: grabbing;
	}

	.card--selected {
		border-color: var(--cv-color-primary, #333333);
		box-shadow: 0 0 0 1px var(--cv-color-primary, #333333);
	}

	.card__media {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 1.5rem;
		color: var(--cv-color-neutral-600, #505050);
	}

	/* Thumbnail cards (templates/images) give the media a taller, full-width slot. */
	.card--thumb .card__media {
		width: 100%;
		aspect-ratio: 4 / 5;
		min-height: 0;
		border-radius: var(--cv-radius-sm, 0.375rem);
		overflow: hidden;
	}

	.card__label {
		font-size: 0.6875rem;
		font-weight: 500;
		line-height: 1.3;
		text-align: center;
		/* Never clip — long labels wrap. */
		overflow-wrap: anywhere;
	}

	/* Hover-revealed overlay — mirrors the property-gallery set-cover affordance
	   (absolute top-right, hidden until the card is hovered/focused). */
	.card__overlay {
		position: absolute;
		top: 0.375rem;
		right: 0.375rem;
		display: flex;
		gap: 0.25rem;
		opacity: 0;
		transition: opacity 0.15s ease;
	}

	.card-wrap:hover .card__overlay,
	.card-wrap:focus-within .card__overlay {
		opacity: 1;
	}
</style>
