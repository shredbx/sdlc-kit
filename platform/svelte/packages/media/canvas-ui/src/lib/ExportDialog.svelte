<script lang="ts">
	// Export dialog (IA refactor R4) — replaces the direct Export action. A core-ui Modal
	// (mirrors ResizeDialog) with two sections:
	//   Image (functional) — PNG / JPEG / PDF, a page-scope choice that appears ONLY for a
	//     multipage document, and an "exports the current frame" note when the doc is animated.
	//   Video (disabled placeholder) — gated on animated mode; the video export ENGINE is a
	//     later slice, so the option stays disabled. For a static document a factual helper
	//     points at the precondition (turn on Animated). No "coming soon" copy.
	// The chosen format + page scope are returned via onexport; the shell owns the rasterize.
	// Cancel / Export are right-aligned (CTA rule).
	import { untrack } from 'svelte';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { Button } from '@sbx/core-ui/components/primitives';
	import type { ExportFormat } from './download.js';

	interface Props {
		open: boolean;
		/** Seeds the format radios each time the dialog opens. */
		format?: ExportFormat;
		/** Page count — the page-scope control shows ONLY for a multipage document. */
		pageCount?: number;
		/** Animated mode — shows the current-frame note + is the precondition for video. */
		animated?: boolean;
		/** Active artboard size — when both are present, a read-only ratio/orientation/size
		 *  summary renders at the top of the dialog (validation aid; Slice S1). */
		width?: number;
		height?: number;
		/** True while a rasterize is in flight (disables Export). */
		exporting?: boolean;
		onclose?: () => void;
		/** Run the export with the chosen image format + page scope. */
		onexport?: (format: ExportFormat, scope: 'current' | 'all') => void;
	}

	let { open, format = 'png', pageCount = 1, animated = false, width, height, exporting = false, onclose, onexport }: Props =
		$props();

	// Size summary (Slice S1) — reduce w:h to lowest terms and name the common ratios,
	// else show the reduced "a:b". Orientation by which edge is longer. Only when both
	// dimensions are present + positive.
	function gcd(a: number, b: number): number {
		return b === 0 ? a : gcd(b, a % b);
	}
	const COMMON_RATIOS: Record<string, string> = {
		'16:9': '16:9',
		'3:2': '3:2',
		'4:3': '4:3',
		'1:1': '1:1'
	};
	const hasSize = $derived(
		typeof width === 'number' && typeof height === 'number' && width > 0 && height > 0
	);
	const ratioLabel = $derived.by(() => {
		if (!hasSize) return '';
		const w = Math.round(width as number);
		const h = Math.round(height as number);
		const g = gcd(w, h) || 1;
		const key = `${w / g}:${h / g}`;
		return COMMON_RATIOS[key] ?? key;
	});
	const orientationLabel = $derived(
		!hasSize ? '' : (width as number) > (height as number) ? 'Landscape' : (height as number) > (width as number) ? 'Portrait' : 'Square'
	);

	const IMAGE_FORMATS: { id: ExportFormat; label: string; hint: string }[] = [
		{ id: 'png', label: 'PNG', hint: 'Lossless · transparency' },
		{ id: 'jpeg', label: 'JPEG', hint: 'Smaller · no transparency' },
		{ id: 'pdf', label: 'PDF', hint: 'Print-ready page' }
	];

	// Seed once from the prop (untrack = explicit one-shot, also silences
	// state_referenced_locally); the open-effect below re-seeds on each open.
	let fmt = $state<ExportFormat>(untrack(() => format));
	let scope = $state<'current' | 'all'>('current');

	// Re-seed on each open (open is the only tracked dep → this fires once per open
	// transition): the format from the prop (untrack = one-shot, not a live mirror — the
	// in-dialog choice owns it while open) and the page scope back to 'current' so a prior
	// 'all' selection doesn't persist into the next open.
	$effect(() => {
		if (open) {
			fmt = untrack(() => format);
			scope = 'current';
		}
	});

	const multipage = $derived(pageCount > 1);
</script>

<Modal {open} title="Export" size="md" onclose={() => onclose?.()}>
	<div class="export">
		{#if hasSize}
			<section class="export__section">
				<h3 class="export__heading">Size</h3>
				<p class="export__note">{ratioLabel} · {orientationLabel} · {Math.round(width as number)} × {Math.round(height as number)} px</p>
			</section>
		{/if}

		<section class="export__section">
			<h3 class="export__heading">Image</h3>
			<div class="export__formats" role="radiogroup" aria-label="Image format">
				{#each IMAGE_FORMATS as f (f.id)}
					<label class="fmt" class:fmt--on={fmt === f.id}>
						<input
							class="fmt__radio"
							type="radio"
							name="export-format"
							value={f.id}
							checked={fmt === f.id}
							onchange={() => (fmt = f.id)}
						/>
						<span class="fmt__label">{f.label}</span>
						<span class="fmt__hint">{f.hint}</span>
					</label>
				{/each}
			</div>

			{#if multipage}
				<fieldset class="export__scope">
					<legend class="export__sub">Pages</legend>
					<label class="scope">
						<input type="radio" name="export-scope" checked={scope === 'current'} onchange={() => (scope = 'current')} />
						Current page
					</label>
					<label class="scope">
						<input type="radio" name="export-scope" checked={scope === 'all'} onchange={() => (scope = 'all')} />
						All pages
					</label>
				</fieldset>
			{/if}

			{#if animated}
				<p class="export__note">Animated document — exports the current frame at the playhead.</p>
			{/if}
		</section>

		<section class="export__section">
			<h3 class="export__heading">Video</h3>
			<label class="fmt fmt--disabled">
				<input class="fmt__radio" type="radio" disabled />
				<span class="fmt__label">MP4</span>
				<span class="fmt__hint">Animated documents</span>
			</label>
			{#if !animated}
				<p class="export__note export__note--muted">Turn on Animated mode to export video.</p>
			{/if}
		</section>
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => onclose?.()}>Cancel</Button>
		<Button variant="primary" size="sm" disabled={exporting} onclick={() => onexport?.(fmt, scope)}>
			{exporting ? 'Exporting…' : 'Export'}
		</Button>
	{/snippet}
</Modal>

<style>
	.export {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-lg, 1.5rem);
	}

	.export__section {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.export__heading {
		margin: 0;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.export__formats {
		display: flex;
		gap: var(--cv-space-sm, 0.5rem);
	}

	/* Selectable format card; the radio itself is visually hidden (the card is the control). */
	.fmt {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 0.125rem;
		padding: var(--cv-space-sm, 0.5rem) var(--cv-space-md, 1rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		cursor: pointer;
		transition: border-color 0.15s ease, background 0.15s ease;
	}

	.fmt--on {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	/* The radio is visually hidden, so surface keyboard focus on the card itself
	   (WCAG 2.4.7 Focus Visible). */
	.fmt:has(.fmt__radio:focus-visible) {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 2px;
	}

	.fmt--disabled {
		cursor: not-allowed;
		opacity: 0.55;
	}

	.fmt__radio {
		position: absolute;
		width: 1px;
		height: 1px;
		opacity: 0;
		pointer-events: none;
	}

	.fmt__label {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--cv-color-neutral-800, #202020);
	}

	.fmt__hint {
		font-size: 0.6875rem;
		color: var(--cv-color-neutral-500, #707070);
	}

	.export__scope {
		display: flex;
		gap: var(--cv-space-md, 1rem);
		margin: 0;
		padding: 0;
		border: none;
	}

	.export__sub {
		padding: 0;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.scope {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		font-size: 0.8125rem;
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
	}

	.export__note {
		margin: 0;
		font-size: 0.75rem;
		line-height: 1.4;
		color: var(--cv-color-neutral-600, #505050);
	}

	.export__note--muted {
		color: var(--cv-color-neutral-500, #707070);
	}
</style>
