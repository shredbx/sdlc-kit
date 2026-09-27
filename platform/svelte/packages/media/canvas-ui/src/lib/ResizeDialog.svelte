<script lang="ts">
	// Resize dialog (Fix 3, §11.6) — a FUNCTIONAL core-ui Modal that mutates the
	// active page's artboard. Two presentations on the SAME Modal/Apply contract:
	//   PRESET GRID (default) — Canva-style preset sizes/ratios that fill W/H, custom
	//     numeric Width/Height, a ☑ Lock proportions checkbox (editing one dimension
	//     scales the other by the current aspect ratio). The Media Canvas path.
	//   PHOTO SIZING (Slice S1, when `ratios` is supplied) — an iPhone-Photos-style
	//     control: an Orientation segment (Landscape · Portrait · Square) + ratio chips
	//     (+ Free) that size the document so the LONGEST edge hits `maxDimension`. The
	//     watermark overlay path. Same Apply (right-aligned) + Cancel; CanvasStage
	//     re-fits automatically. SSR-safe (no window/document access).
	import { untrack } from 'svelte';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { Button, Field, Input, Icon } from '@sbx/core-ui/components/primitives';
	import SegmentedControl from './SegmentedControl.svelte';

	/** A ratio offered in photo-sizing mode — small ints (e.g. {w:16,h:9} = 16:9). */
	interface RatioOption {
		id: string;
		label: string;
		w: number;
		h: number;
	}

	interface Props {
		open: boolean;
		/** Current artboard width (seeds the inputs each time the dialog opens). */
		width: number;
		/** Current artboard height. */
		height: number;
		/** Photo-sizing mode (Slice S1): when supplied, the dialog renders the
		 *  Orientation + ratio-chip control INSTEAD of the preset grid. First entry is
		 *  the default selection. Omit → the classic preset grid (Media Canvas). */
		ratios?: RatioOption[];
		/** Longest-edge cap for photo-sizing mode (watermark passes 2048). Every
		 *  computed/entered dimension is clamped to 1..maxDimension. */
		maxDimension?: number;
		onclose?: () => void;
		/** Apply the new dimensions to the active page (real mutation). */
		onapply?: (width: number, height: number) => void;
	}

	let { open, width, height, ratios, maxDimension = 2048, onclose, onapply }: Props = $props();

	// Photo-sizing mode is active iff the consumer supplied a ratio set.
	const photoMode = $derived(!!ratios && ratios.length > 0);

	interface SizePreset {
		id: string;
		label: string;
		ratio: string;
		w: number;
		h: number;
	}

	// Standard social/print presets (don't reinvent — Canva's common set).
	const presets: SizePreset[] = [
		{ id: 'square', label: 'Square', ratio: '1:1', w: 1080, h: 1080 },
		{ id: 'portrait', label: 'Portrait', ratio: '4:5', w: 1080, h: 1350 },
		{ id: 'story', label: 'Story', ratio: '9:16', w: 1080, h: 1920 },
		{ id: 'landscape', label: 'Landscape', ratio: '16:9', w: 1920, h: 1080 },
		{ id: 'wide', label: 'Wide', ratio: '21:9', w: 2520, h: 1080 },
		{ id: 'a4', label: 'A4', ratio: '√2', w: 2480, h: 3508 }
	];

	// Working values as strings (Input is string-valued); seeded ONCE per open
	// transition from the incoming dims so re-opening reflects the live artboard.
	let wStr = $state('');
	let hStr = $state('');
	let lockProportions = $state(true);
	// The aspect ratio used when Lock proportions scales the other dimension.
	let lockedRatio = $state(1);
	let wasOpen = false;

	// ── Photo-sizing state (Slice S1) ────────────────────────────────────────
	// Orientation is the long/short-edge mapping; 'free' is the ratio id when the
	// dimensions are independent (the orientation control hides). Seeded on open
	// by inferring the best ratio + orientation from the incoming dims.
	type Orientation = 'landscape' | 'portrait' | 'square';
	const FREE = 'free';
	let orientation = $state<Orientation>('landscape');
	let ratioId = $state<string>(FREE);

	const ORIENTATION_SEGMENTS = [
		{ value: 'landscape', label: 'Landscape' },
		{ value: 'portrait', label: 'Portrait' },
		{ value: 'square', label: 'Square' }
	];

	const activeRatio = $derived(ratios?.find((r) => r.id === ratioId) ?? null);
	const isSquareRatio = $derived(!!activeRatio && activeRatio.w === activeRatio.h);
	// Square ratio (or no ratio that allows portrait/landscape) forces the
	// orientation control off — there's nothing to swap.
	const orientationDisabled = $derived(ratioId === FREE || isSquareRatio);

	function clampDim(n: number): number {
		if (!Number.isFinite(n)) return 1;
		return Math.min(maxDimension, Math.max(1, Math.round(n)));
	}

	/** Size the document for ratio R + orientation O so the LONGEST edge = maxDimension.
	 *  Landscape → w=max, h=max·Rh/Rw; Portrait swaps; Square → w=h=max. */
	function dimsFor(r: RatioOption, o: Orientation): { w: number; h: number } {
		if (r.w === r.h || o === 'square') return { w: maxDimension, h: maxDimension };
		// Normalize so the ratio's larger leg is the long edge, then map by orientation.
		const long = Math.max(r.w, r.h);
		const short = Math.min(r.w, r.h);
		const shortEdge = clampDim((maxDimension * short) / long);
		return o === 'portrait'
			? { w: shortEdge, h: maxDimension }
			: { w: maxDimension, h: shortEdge };
	}

	/** Apply the active ratio + orientation to the inputs (photo mode). */
	function syncFromRatio(): void {
		if (!activeRatio) return;
		const { w, h } = dimsFor(activeRatio, orientation);
		wStr = String(w);
		hStr = String(h);
	}

	/** Pick the orientation + ratio that best matches the incoming dims (within ~1px),
	 *  else Free. Used to seed the controls so re-opening reflects the live artboard. */
	function inferSelection(w: number, h: number): void {
		orientation = w > h ? 'landscape' : h > w ? 'portrait' : 'square';
		if (!ratios) {
			ratioId = FREE;
			return;
		}
		const long = Math.max(w, h);
		const short = Math.min(w, h);
		const target = long > 0 ? short / long : 1;
		let best: RatioOption | null = null;
		let bestErr = Infinity;
		for (const r of ratios) {
			const rTarget = Math.min(r.w, r.h) / Math.max(r.w, r.h);
			// Tolerance in the SHORT/LONG fraction that maps to ~1px at maxDimension.
			const err = Math.abs(rTarget - target);
			if (err < bestErr) {
				bestErr = err;
				best = r;
			}
		}
		ratioId = best && bestErr * maxDimension <= 1 ? best.id : FREE;
	}

	$effect(() => {
		// Seed inputs on the closed→open transition (untrack: a one-time seed, not
		// a live mirror of the props while the user edits).
		if (open && !wasOpen) {
			untrack(() => {
				wStr = String(width);
				hStr = String(height);
				lockedRatio = height > 0 ? width / height : 1;
				// Photo mode also infers the orientation/ratio chips from the seed dims.
				if (ratios && ratios.length > 0) inferSelection(width, height);
			});
		}
		wasOpen = open;
	});

	const wNum = $derived(Number(wStr));
	const hNum = $derived(Number(hStr));
	const valid = $derived(Number.isFinite(wNum) && Number.isFinite(hNum) && wNum > 0 && hNum > 0);

	function selectPreset(p: SizePreset): void {
		wStr = String(p.w);
		hStr = String(p.h);
		lockedRatio = p.w / p.h;
	}

	// ── Preset-grid (lock-proportions) handlers — unchanged ───────────────────
	function handleWidthInput(): void {
		if (lockProportions && Number(wStr) > 0 && lockedRatio > 0) {
			hStr = String(Math.round(Number(wStr) / lockedRatio));
		}
	}

	function handleHeightInput(): void {
		if (lockProportions && Number(hStr) > 0 && lockedRatio > 0) {
			wStr = String(Math.round(Number(hStr) * lockedRatio));
		}
	}

	function toggleLock(): void {
		lockProportions = !lockProportions;
		// Re-capture the ratio from the current values when locking back on.
		if (lockProportions && Number(hStr) > 0) lockedRatio = Number(wStr) / Number(hStr);
	}

	// ── Photo-sizing handlers (Slice S1) ──────────────────────────────────────
	function selectOrientation(o: Orientation): void {
		orientation = o;
		syncFromRatio(); // swaps the W↔H mapping for the active ratio
	}

	function selectRatio(id: string): void {
		ratioId = id;
		if (id === FREE) return; // Free: keep the current independent W/H
		// A square ratio forces orientation to square; else keep the chosen orientation.
		const r = ratios?.find((x) => x.id === id);
		if (r && r.w === r.h) orientation = 'square';
		else if (orientation === 'square') orientation = 'landscape';
		syncFromRatio();
	}

	// In photo mode with a ratio locked, editing one dimension recomputes the other
	// per the active ratio+orientation; Free leaves them independent. Both clamp.
	function photoWidthInput(): void {
		let w = clampDim(Number(wStr));
		wStr = String(w);
		if (ratioId !== FREE && activeRatio && !isSquareRatio) {
			const long = Math.max(activeRatio.w, activeRatio.h);
			const short = Math.min(activeRatio.w, activeRatio.h);
			// Width is the long edge in landscape, the short edge in portrait.
			const h = orientation === 'portrait' ? (w * long) / short : (w * short) / long;
			hStr = String(clampDim(h));
		} else if (isSquareRatio) {
			hStr = String(w);
		}
	}

	function photoHeightInput(): void {
		let h = clampDim(Number(hStr));
		hStr = String(h);
		if (ratioId !== FREE && activeRatio && !isSquareRatio) {
			const long = Math.max(activeRatio.w, activeRatio.h);
			const short = Math.min(activeRatio.w, activeRatio.h);
			// Height is the short edge in landscape, the long edge in portrait.
			const w = orientation === 'portrait' ? (h * short) / long : (h * long) / short;
			wStr = String(clampDim(w));
		} else if (isSquareRatio) {
			wStr = String(h);
		}
	}

	function apply(): void {
		if (!valid) return;
		// Trust boundary to editor.resizePage: in photo mode re-assert the cap so no edit
		// path can ever emit a dimension > maxDimension, independent of which handler ran.
		const w = photoMode ? clampDim(wNum) : Math.round(wNum);
		const h = photoMode ? clampDim(hNum) : Math.round(hNum);
		onapply?.(w, h);
		onclose?.();
	}
</script>

<Modal {open} title="Resize artboard" onclose={() => onclose?.()}>
	<div class="resize">
		{#if photoMode}
			<!-- iPhone-Photos-style sizing (Slice S1): Orientation + ratio chips drive the
			     dimensions so the longest edge hits maxDimension. -->
			<fieldset class="resize__presets">
				<legend class="resize__legend">Orientation</legend>
				<SegmentedControl
					aria-label="Orientation"
					value={orientation}
					options={ORIENTATION_SEGMENTS}
					disabled={orientationDisabled}
					onchange={(o) => selectOrientation(o as Orientation)}
				/>
			</fieldset>

			<fieldset class="resize__presets">
				<legend class="resize__legend">Aspect ratio</legend>
				<div class="resize__chips" role="radiogroup" aria-label="Aspect ratio">
					{#each ratios ?? [] as r (r.id)}
						<button
							class="chip"
							class:chip--active={ratioId === r.id}
							type="button"
							role="radio"
							aria-checked={ratioId === r.id}
							onclick={() => selectRatio(r.id)}
						>
							{r.label}
						</button>
					{/each}
					<button
						class="chip"
						class:chip--active={ratioId === FREE}
						type="button"
						role="radio"
						aria-checked={ratioId === FREE}
						onclick={() => selectRatio(FREE)}
					>
						Free
					</button>
				</div>
			</fieldset>

			<div class="resize__custom">
				<Field for="resize-width" label="Width (px)">
					{#snippet children(ids)}
						<Input
							id="resize-width"
							name="resize-width"
							type="number"
							inputmode="numeric"
							min="1"
							max={maxDimension}
							bind:value={wStr}
							oninput={photoWidthInput}
							aria-describedby={ids.describedBy}
						/>
					{/snippet}
				</Field>
				<Field for="resize-height" label="Height (px)">
					{#snippet children(ids)}
						<Input
							id="resize-height"
							name="resize-height"
							type="number"
							inputmode="numeric"
							min="1"
							max={maxDimension}
							bind:value={hStr}
							oninput={photoHeightInput}
							aria-describedby={ids.describedBy}
						/>
					{/snippet}
				</Field>
			</div>
			<p class="resize__hint">Longest edge caps at max {maxDimension} px.</p>
		{:else}
			<fieldset class="resize__presets">
				<legend class="resize__legend">Preset sizes</legend>
				<div class="resize__preset-grid">
					{#each presets as p (p.id)}
						{@const active = Number(wStr) === p.w && Number(hStr) === p.h}
						<button
							class="preset"
							class:preset--active={active}
							type="button"
							onclick={() => selectPreset(p)}
						>
							<span class="preset__label">{p.label}</span>
							<span class="preset__dims">{p.w} × {p.h}</span>
							<span class="preset__ratio">{p.ratio}</span>
						</button>
					{/each}
				</div>
			</fieldset>

			<div class="resize__custom">
				<Field for="resize-width" label="Width (px)">
					{#snippet children(ids)}
						<Input
							id="resize-width"
							name="resize-width"
							type="number"
							inputmode="numeric"
							min="1"
							bind:value={wStr}
							oninput={handleWidthInput}
							aria-describedby={ids.describedBy}
						/>
					{/snippet}
				</Field>
				<Field for="resize-height" label="Height (px)">
					{#snippet children(ids)}
						<Input
							id="resize-height"
							name="resize-height"
							type="number"
							inputmode="numeric"
							min="1"
							bind:value={hStr}
							oninput={handleHeightInput}
							aria-describedby={ids.describedBy}
						/>
					{/snippet}
				</Field>
			</div>

			<button class="lock" type="button" aria-pressed={lockProportions} onclick={toggleLock}>
				<span class="lock__box" class:lock__box--on={lockProportions} aria-hidden="true">
					{#if lockProportions}<Icon name="check" size="sm" />{/if}
				</span>
				<span class="lock__label">Lock proportions</span>
			</button>
		{/if}
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => onclose?.()}>Cancel</Button>
		<Button variant="primary" size="sm" disabled={!valid} onclick={apply}>Apply</Button>
	{/snippet}
</Modal>

<style>
	.resize {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-md, 1rem);
	}

	.resize__presets {
		margin: 0;
		padding: 0;
		border: none;
	}

	.resize__legend {
		padding: 0;
		margin-bottom: var(--cv-space-sm, 0.5rem);
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.resize__preset-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: var(--cv-space-sm, 0.5rem);
	}

	/* Ratio chips (photo-sizing mode) — a wrapping row of selectable pills. */
	.resize__chips {
		display: flex;
		flex-wrap: wrap;
		gap: var(--cv-space-sm, 0.5rem);
	}

	.chip {
		padding: 0.3125rem 0.75rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: 999px;
		background: var(--cv-color-neutral-50, #fafafa);
		color: var(--cv-color-neutral-700, #383838);
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
		transition: border-color 0.15s ease, background 0.15s ease, color 0.15s ease;
	}

	.chip:hover:not(.chip--active) {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.chip--active {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary, #333333);
		color: var(--cv-color-surface, #fff);
	}

	.chip:focus-visible {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 2px;
	}

	/* Max-dimension hint under the W/H inputs (photo-sizing mode). */
	.resize__hint {
		margin: 0;
		font-size: 0.75rem;
		line-height: 1.4;
		color: var(--cv-color-neutral-500, #707070);
	}

	.preset {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.125rem;
		padding: var(--cv-space-sm, 0.5rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-neutral-50, #fafafa);
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
		text-align: left;
		transition: border-color 0.15s ease, background 0.15s ease;
	}

	.preset:hover {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.preset--active {
		border-color: var(--cv-color-primary, #333333);
		box-shadow: 0 0 0 1px var(--cv-color-primary, #333333);
	}

	.preset__label {
		font-size: 0.8125rem;
		font-weight: 600;
	}

	.preset__dims {
		font-size: 0.6875rem;
		font-variant-numeric: tabular-nums;
		color: var(--cv-color-neutral-600, #505050);
	}

	.preset__ratio {
		font-size: 0.625rem;
		font-weight: 600;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	.resize__custom {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--cv-space-md, 1rem);
	}

	.lock {
		display: inline-flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		align-self: flex-start;
		padding: 0;
		border: none;
		background: transparent;
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
	}

	.lock__box {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 1.125rem;
		height: 1.125rem;
		border: 1px solid var(--cv-color-neutral-400, #a0a0a0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-surface, #fff);
	}

	.lock__box--on {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary, #333333);
	}

	.lock__label {
		font-size: 0.8125rem;
		font-weight: 500;
	}
</style>
