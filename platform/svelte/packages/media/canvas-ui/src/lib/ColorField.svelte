<script lang="ts">
	// ColorField — the Inspector's ONE reusable color control (Decision #0286
	// Phase A; design pass A 2026-06-07): a COMPACT closed face (current-color
	// chip + value + chevron, input-height) that opens a popover with the brand
	// swatch grid + the free-entry hex Input. Replaces the always-open swatch
	// strip that ate a third of the panel. Used by every color row (text COLOR ·
	// shape FILL · shape STROKE · background COLOR) — markup lives here once,
	// never copy-pasted. Brand-agnostic: swatches come from the consumer's
	// BrandKit; with no entries the popover offers the hex input alone. <Field>
	// wraps the Input as before (field-owns-error rule).
	import { Field, Input, Icon } from '@sbx/core-ui/components/primitives';
	import type { BrandColor } from '@sbx/canvas-kit';

	interface Props {
		id: string;
		name: string;
		/** Accessible label context (e.g. "Text color") — lands on the face, the
		 *  Input and the swatch group label. */
		'aria-label': string;
		value: string;
		/** Brand entries offered as swatches (BrandKit.colors). Empty = input only. */
		colors?: BrandColor[];
		/** Fires for BOTH a swatch pick and free-entry typing. */
		onpick?: (value: string) => void;
	}

	let { id, name, 'aria-label': ariaLabel, value, colors = [], onpick }: Props = $props();

	let open = $state(false);
	let rootEl = $state<HTMLDivElement | null>(null);
	let popEl = $state<HTMLDivElement | null>(null);

	// Reveal the popover when it opens near the panel's bottom edge — the Inspector
	// scrolls (overflow-y:auto), so an absolute popover can land below the fold;
	// 'nearest' never jumps when it is already visible.
	$effect(() => {
		popEl?.scrollIntoView({ block: 'nearest' });
	});

	/** Case-insensitive match against the current value — drives the selected ring. */
	function isSelected(candidate: string): boolean {
		return candidate.trim().toLowerCase() === (value ?? '').trim().toLowerCase();
	}

	/** The brand name for the current value (the face label beats a raw hex). */
	const faceLabel = $derived(
		colors.find((c) => isSelected(c.value))?.name ?? (value || '(unset)')
	);

	// Close on outside click / Escape while open (document-level, browser-only).
	$effect(() => {
		if (!open) return;
		function onDocPointer(e: PointerEvent): void {
			if (rootEl && !rootEl.contains(e.target as Node)) open = false;
		}
		function onDocKey(e: KeyboardEvent): void {
			if (e.key === 'Escape') open = false;
		}
		document.addEventListener('pointerdown', onDocPointer, true);
		document.addEventListener('keydown', onDocKey, true);
		return () => {
			document.removeEventListener('pointerdown', onDocPointer, true);
			document.removeEventListener('keydown', onDocKey, true);
		};
	});
</script>

<div class="color-field" bind:this={rootEl}>
	<button
		type="button"
		class="face"
		aria-label={ariaLabel}
		aria-haspopup="true"
		aria-expanded={open}
		title={value}
		onclick={() => (open = !open)}
	>
		<span class="face__chip" style:background-color={value || '#ffffff'}></span>
		<span class="face__value">{faceLabel}</span>
		<Icon name="chevron-down" size="xs" />
	</button>

	{#if open}
		<div class="pop" bind:this={popEl}>
			{#if colors.length > 0}
				<span class="pop__label">Brand</span>
				<div class="pop__swatches" role="group" aria-label="{ariaLabel} brand swatches">
					{#each colors as color (color.name)}
						<button
							type="button"
							class="swatch"
							class:swatch--selected={isSelected(color.value)}
							style:background-color={color.value}
							aria-label={color.name}
							aria-pressed={isSelected(color.value)}
							title={color.name}
							onclick={() => {
								onpick?.(color.value);
								open = false;
							}}
						></button>
					{/each}
				</div>
			{/if}
			<span class="pop__label">Hex</span>
			<Field for={id}>
				{#snippet children()}
					<Input {id} {name} size="sm" {value} oninput={(v) => onpick?.(v)} aria-label={ariaLabel} />
				{/snippet}
			</Field>
		</div>
	{/if}
</div>

<style>
	/* Anchor for the popover. */
	.color-field {
		position: relative;
	}

	/* Closed face — input-height (28px) chip + value + chevron. */
	.face {
		display: flex;
		align-items: center;
		gap: var(--cv-space-xs, 0.25rem);
		width: 100%;
		height: 1.75rem;
		padding: 0 0.375rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		font: inherit;
		font-size: 0.75rem;
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
	}

	.face:hover {
		border-color: var(--cv-color-neutral-400, #a0a0a0);
	}

	.face:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 1px;
	}

	.face__chip {
		flex-shrink: 0;
		width: 1rem;
		height: 1rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: 0.25rem;
	}

	/* The value/name — single line; the full value is in the title tooltip. */
	.face__value {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
		text-align: left;
	}

	/* Popover — anchored under the face, above the panel content. */
	.pop {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		z-index: 10;
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-xs, 0.25rem);
		padding: var(--cv-space-sm, 0.5rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
	}

	.pop__label {
		font-size: 0.625rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	/* Swatch grid — wraps onto more rows when the kit is large (never clipped). */
	.pop__swatches {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}

	.swatch {
		width: 20px;
		height: 20px;
		padding: 0;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		cursor: pointer;
	}

	.swatch:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 1px;
	}

	/* Selected ring — a surface gap + accent ring reads on light AND dark swatches. */
	.swatch--selected {
		box-shadow:
			0 0 0 1px var(--cv-color-surface, #fff),
			0 0 0 3px var(--cv-color-primary, #333333);
	}
</style>
