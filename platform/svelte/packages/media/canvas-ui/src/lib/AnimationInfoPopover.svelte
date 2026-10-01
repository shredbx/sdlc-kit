<script lang="ts">
	// Animation info / edit popover (slice-5b, design §12.5 double-click). A core-ui
	// Modal (mirrors AddAnimationDialog — consistent + robust, sidesteps popover
	// positioning bugs) that reads + LIVE-edits the selected property's 2-keyframe
	// track: From/To (display units), Start (s) + Duration (s), Easing presets (with
	// an EasingCurvePreview), and Delete. Every field writes through
	// editor.updateAnimation immediately, so "Done" just closes. Times are edited in
	// SECONDS here; the model stores ms (§12.7 — times editable in the popup OR by
	// dragging grabbers, both writing the same track). CTA right-aligned (hard rule);
	// Delete uses brand red. SSR-safe (no window/document access). Number inputs bind
	// to NUMBERS — never call string methods on the bound values.
	import { untrack } from 'svelte';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { Button, Field, Input } from '@sbx/core-ui/components/primitives';
	import {
		easingPreset,
		easingToPresetName,
		type AnimatableProperty,
		type EasingPresetName
	} from '@sbx/canvas-kit';
	import type { CanvasEditor } from './editor-state.svelte.js';
	import { animatableMeta, toDisplay, toModel } from './palette.js';
	import EasingCurvePreview from './EasingCurvePreview.svelte';

	interface Props {
		/** Whether the popover is open. */
		open: boolean;
		/** The reactive editor — reads the track + calls updateAnimation/removeAnimation. */
		editor: CanvasEditor;
		/** Layer that owns the animation (the selected bar's layer). */
		layerId: string | null;
		/** Property whose track is being edited (the selected bar). */
		property: AnimatableProperty | null;
		onclose?: () => void;
	}

	let { open, editor, layerId, property, onclose }: Props = $props();

	const easingPresets: { name: EasingPresetName; label: string }[] = [
		{ name: 'in', label: 'In' },
		{ name: 'out', label: 'Out' },
		{ name: 'in-out', label: 'In-out' },
		{ name: 'linear', label: 'Linear' }
	];

	// The layer + its track for `property`, resolved live so edits reflect at once.
	const layer = $derived(layerId ? editor.layers.find((l) => l.id === layerId) : undefined);
	const track = $derived(
		property ? layer?.animations?.find((t) => t.property === property) : undefined
	);

	// The track's current spec (sort keyframes defensively; first = from, last = to).
	const current = $derived.by(() => {
		if (!track || track.keyframes.length < 2) return null;
		const kfs = [...track.keyframes].sort((a, b) => a.time - b.time);
		const first = kfs[0];
		const last = kfs[kfs.length - 1];
		return {
			from: Number(first.value),
			to: Number(last.value),
			startMs: first.time,
			endMs: last.time,
			easing: first.easing
		};
	});

	// Metadata for the active property (drives the unit suffix shown on From/To).
	const meta = $derived(property ? animatableMeta(property) : undefined);
	const unit = $derived(meta?.unit ?? '');
	const activeEasing = $derived<EasingPresetName>(current ? easingToPresetName(current.easing) : 'linear');

	// --- Working form state (display units), seeded on the closed→open transition.
	let fromStr = $state('');
	let toStr = $state('');
	let startSecStr = $state('');
	let durSecStr = $state('');
	let wasOpen = false;

	$effect(() => {
		// Seed once per open transition (untrack: a one-time seed, not a live mirror —
		// the editor is the source of truth once open; the inputs push edits to it).
		if (open && !wasOpen) {
			untrack(() => seedFromTrack());
		}
		wasOpen = open;
	});

	function seedFromTrack(): void {
		const c = current;
		const p = property;
		if (!c || !p) {
			fromStr = '';
			toStr = '';
			startSecStr = '';
			durSecStr = '';
			return;
		}
		fromStr = String(toDisplay(p, c.from));
		toStr = String(toDisplay(p, c.to));
		startSecStr = String(round2(c.startMs / 1000));
		durSecStr = String(round2((c.endMs - c.startMs) / 1000));
	}

	function round2(n: number): number {
		return Math.round(n * 100) / 100;
	}

	// Number inputs bind to numbers at runtime ('' / null when cleared) — guard with
	// Number.isFinite (NEVER call string methods on the bound values). A blank or NaN
	// field is simply not committed, so a half-typed value never corrupts the track.
	const isBlank = (v: unknown): boolean => v === '' || v === null || v === undefined;

	function commitFromTo(): void {
		if (!layerId || !property) return;
		const f = Number(fromStr);
		const t = Number(toStr);
		// From≠To so the popover never flattens the track to a no-op (5a guard parity).
		if (isBlank(fromStr) || isBlank(toStr) || !Number.isFinite(f) || !Number.isFinite(t) || f === t) {
			return;
		}
		editor.updateAnimation(layerId, property, {
			from: toModel(property, f),
			to: toModel(property, t)
		});
	}

	function commitStart(): void {
		if (!layerId || !property || !current) return;
		const s = Number(startSecStr);
		if (isBlank(startSecStr) || !Number.isFinite(s) || s < 0) return;
		// Keep the END fixed; clamp so start can't cross end (min 0.5s duration).
		const startMs = Math.min(Math.round(s * 1000), current.endMs - 500);
		const clamped = Math.max(0, startMs);
		editor.updateAnimation(layerId, property, {
			startMs: clamped,
			durMs: current.endMs - clamped
		});
	}

	function commitDuration(): void {
		if (!layerId || !property || !current) return;
		const d = Number(durSecStr);
		if (isBlank(durSecStr) || !Number.isFinite(d)) return;
		const durMs = Math.max(500, Math.round(d * 1000)); // min 0.5s
		editor.updateAnimation(layerId, property, { durMs });
	}

	function selectEasing(name: EasingPresetName): void {
		if (!layerId || !property) return;
		editor.updateAnimation(layerId, property, { easing: easingPreset(name) });
	}

	function remove(): void {
		if (!layerId || !property) return;
		editor.removeAnimation(layerId, property);
		onclose?.();
	}
</script>

<Modal {open} title="Edit animation" onclose={() => onclose?.()}>
	<div class="anim">
		{#if current && property}
			<p class="anim__subject">{meta?.label ?? property} animation</p>

			<div class="anim__row">
				<Field for="info-from" label={`From (${unit})`}>
					{#snippet children()}
						<Input
							id="info-from"
							name="info-from"
							type="number"
							inputmode="decimal"
							bind:value={fromStr}
							oninput={commitFromTo}
							aria-label="From value"
						/>
					{/snippet}
				</Field>
				<Field for="info-to" label={`To (${unit})`}>
					{#snippet children()}
						<Input
							id="info-to"
							name="info-to"
							type="number"
							inputmode="decimal"
							bind:value={toStr}
							oninput={commitFromTo}
							aria-label="To value"
						/>
					{/snippet}
				</Field>
			</div>

			<div class="anim__row">
				<Field for="info-start" label="Start (s)">
					{#snippet children()}
						<Input
							id="info-start"
							name="info-start"
							type="number"
							inputmode="decimal"
							min="0"
							step="0.5"
							bind:value={startSecStr}
							oninput={commitStart}
							aria-label="Start time in seconds"
						/>
					{/snippet}
				</Field>
				<Field for="info-duration" label="Duration (s)">
					{#snippet children()}
						<Input
							id="info-duration"
							name="info-duration"
							type="number"
							inputmode="decimal"
							min="0.5"
							step="0.5"
							bind:value={durSecStr}
							oninput={commitDuration}
							aria-label="Duration in seconds"
						/>
					{/snippet}
				</Field>
			</div>

			<fieldset class="anim__easing">
				<legend class="anim__legend">Easing</legend>
				<div class="anim__easing-grid">
					<div class="anim__easing-row">
						{#each easingPresets as preset (preset.name)}
							<button
								class="ease"
								class:ease--active={activeEasing === preset.name}
								type="button"
								aria-pressed={activeEasing === preset.name}
								onclick={() => selectEasing(preset.name)}
							>
								{preset.label}
							</button>
						{/each}
					</div>
					<EasingCurvePreview easing={activeEasing} />
				</div>
			</fieldset>

			<div class="anim__delete">
				<button class="delete" type="button" onclick={remove}>Delete animation</button>
			</div>
		{:else}
			<p class="anim__missing">This animation is no longer available.</p>
		{/if}
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => onclose?.()}>Cancel</Button>
		<Button variant="primary" size="sm" onclick={() => onclose?.()}>Done</Button>
	{/snippet}
</Modal>

<style>
	.anim {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-md, 1rem);
	}

	.anim__subject {
		margin: 0;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.anim__row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--cv-space-md, 1rem);
	}

	.anim__easing {
		margin: 0;
		padding: 0;
		border: none;
	}

	.anim__legend {
		padding: 0;
		margin-bottom: var(--cv-space-sm, 0.5rem);
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.anim__easing-grid {
		display: flex;
		align-items: center;
		gap: var(--cv-space-md, 1rem);
	}

	.anim__easing-row {
		flex: 1;
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: var(--cv-space-sm, 0.5rem);
	}

	.ease {
		padding: var(--cv-space-sm, 0.5rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-md, 0.5rem);
		background: var(--cv-color-neutral-50, #fafafa);
		color: var(--cv-color-neutral-700, #383838);
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
		transition: border-color 0.15s ease, background 0.15s ease, color 0.15s ease;
	}

	.ease:hover {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.ease--active {
		border-color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary, #333333);
		color: var(--cv-color-surface, #fff);
	}

	/* Destructive action, left-aligned (it's not a confirm CTA — the footer holds
	   the Cancel/Done CTAs, which stay right-aligned per the hard rule). */
	.anim__delete {
		display: flex;
	}

	.delete {
		display: inline-flex;
		align-items: center;
		padding: var(--cv-space-sm, 0.5rem) var(--cv-space-md, 1rem);
		border: 1px solid #e5392b;
		border-radius: var(--cv-radius-md, 0.5rem);
		background: transparent;
		color: #e5392b;
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.15s ease, color 0.15s ease;
	}

	.delete:hover {
		background: #e5392b;
		color: #fff;
	}

	.anim__missing {
		margin: 0;
		font-size: 0.875rem;
		color: var(--cv-color-neutral-600, #505050);
	}
</style>
