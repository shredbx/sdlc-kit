<script lang="ts">
	// Add-animation dialog (slice-5a, design §12.4) — a FUNCTIONAL core-ui Modal
	// (mirrors ResizeDialog's open/close/footer/CTA pattern) that authors one
	// property's 2-keyframe animation on the selected layer. Fields: Property
	// (animatable; pre-selected + locked when opened from an Inspector row, else a
	// dropdown of not-yet-animated props), From/To (display units — opacity/scale
	// edit as %), Duration (1/2/3s/Custom), Easing (In/Out/In-out/Linear). On Add it
	// converts display→model, builds an AnimationSpec, calls editor.addAnimation, and
	// closes. CTA right-aligned (hard rule). SSR-safe (no window/document access).
	import { untrack } from 'svelte';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import { Button, Field, Input, Select } from '@sbx/core-ui/components/primitives';
	import {
		easingPreset,
		type AnimationSpec,
		type AnimatableProperty,
		type EasingPresetName
	} from '@sbx/canvas-kit';
	import type { CanvasEditor } from './editor-state.svelte.js';
	import { ANIMATABLE_PROPS, animatableMeta, toDisplay, toModel } from './palette.js';

	interface Props {
		/** Whether the dialog is open. */
		open: boolean;
		/** The reactive editor — the dialog reads base values + calls addAnimation. */
		editor: CanvasEditor;
		/** Layer the animation is authored on (the selected layer). */
		layerId: string | null;
		/** Pre-selected property — locks the dropdown when opened from an Inspector ≣. */
		property?: AnimatableProperty;
		/** Segment start time in ms (§12.4 — Inspector shortcut → 0). */
		startMs?: number;
		onclose?: () => void;
	}

	let { open, editor, layerId, property, startMs = 0, onclose }: Props = $props();

	// The layer being authored (resolved live so base values are current).
	const layer = $derived(layerId ? editor.layers.find((l) => l.id === layerId) : undefined);

	// Property dropdown rows. When opened from an Inspector row the property is
	// fixed (locked); otherwise list only the layer's not-yet-animated props.
	const propertyOptions = $derived.by(() => {
		if (property) {
			const meta = animatableMeta(property);
			return meta ? [{ value: meta.key, label: `${meta.label} (${meta.unit})` }] : [];
		}
		return ANIMATABLE_PROPS.filter((p) => !editor.isAnimated(layer, p.key)).map((p) => ({
			value: p.key,
			label: `${p.label} (${p.unit})`
		}));
	});

	// --- Working form state (seeded on the closed→open transition) -------------
	let propKey = $state<AnimatableProperty>('opacity');
	let fromStr = $state('');
	let toStr = $state('');
	let durationChoice = $state('1'); // '1' | '2' | '3' | 'custom'
	let customSecondsStr = $state('1');
	let easing = $state<EasingPresetName>('linear');
	let wasOpen = false;

	const durationOptions = [
		{ value: '1', label: '1s' },
		{ value: '2', label: '2s' },
		{ value: '3', label: '3s' },
		{ value: 'custom', label: 'Custom' }
	];

	const easingPresets: { name: EasingPresetName; label: string }[] = [
		{ name: 'in', label: 'In' },
		{ name: 'out', label: 'Out' },
		{ name: 'in-out', label: 'In-out' },
		{ name: 'linear', label: 'Linear' }
	];

	// Metadata for the active property (drives unit suffix + display scaling).
	const meta = $derived(animatableMeta(propKey));
	const unit = $derived(meta?.unit ?? '');

	$effect(() => {
		// Seed once per open transition (untrack: a one-time seed, not a live mirror).
		if (open && !wasOpen) {
			untrack(() => {
				const initial = property ?? propertyOptions[0]?.value ?? 'opacity';
				propKey = initial as AnimatableProperty;
				seedFromBase(propKey);
				durationChoice = '1';
				customSecondsStr = '1';
				easing = 'linear';
			});
		}
		wasOpen = open;
	});

	// "From" = the layer's current value (in display units); "To" mirrors it so the
	// user only changes the target. Re-seed when the chosen property changes.
	function seedFromBase(key: AnimatableProperty): void {
		const base = layer ? editor.baseValueFor(layer, key) : 0;
		const display = String(toDisplay(key, base));
		fromStr = display;
		toStr = display;
	}

	function onPropertyChange(value: string): void {
		propKey = value as AnimatableProperty;
		seedFromBase(propKey);
	}

	// Duration in ms from the choice (Custom reveals a seconds input).
	const durMs = $derived.by(() => {
		if (durationChoice === 'custom') {
			const s = Number(customSecondsStr);
			return Number.isFinite(s) && s > 0 ? Math.round(s * 1000) : 0;
		}
		return Number(durationChoice) * 1000;
	});

	const fromNum = $derived(Number(fromStr));
	const toNum = $derived(Number(toStr));
	// The From/To inputs are type="number", so bind:value yields a number at runtime
	// (or '' / null when empty) — NEVER call string methods on them. A blank field
	// coerces to 0 via Number(); guard it explicitly so a cleared field can't silently
	// animate from 0, and require From≠To so Add never builds a flat (no-op) track.
	const isBlank = (v: unknown): boolean => v === '' || v === null || v === undefined;
	const valid = $derived(
		!!layerId &&
			!!propKey &&
			!isBlank(fromStr) &&
			!isBlank(toStr) &&
			Number.isFinite(fromNum) &&
			Number.isFinite(toNum) &&
			fromNum !== toNum &&
			durMs > 0
	);

	function add(): void {
		if (!valid || !layerId) return;
		const spec: AnimationSpec = {
			property: propKey,
			from: toModel(propKey, fromNum),
			to: toModel(propKey, toNum),
			startMs,
			durMs,
			easing: easingPreset(easing)
		};
		editor.addAnimation(layerId, spec);
		onclose?.();
	}
</script>

<Modal {open} title="Add animation" onclose={() => onclose?.()}>
	<div class="anim">
		<Field for="anim-property" label="Property">
			{#snippet children()}
				<Select
					id="anim-property"
					name="anim-property"
					value={propKey}
					options={propertyOptions}
					disabled={!!property}
					onchange={onPropertyChange}
				/>
			{/snippet}
		</Field>

		<div class="anim__row">
			<Field for="anim-from" label={`From (${unit})`}>
				{#snippet children()}
					<Input
						id="anim-from"
						name="anim-from"
						type="number"
						inputmode="decimal"
						bind:value={fromStr}
						aria-label="From value"
					/>
				{/snippet}
			</Field>
			<Field for="anim-to" label={`To (${unit})`}>
				{#snippet children()}
					<Input
						id="anim-to"
						name="anim-to"
						type="number"
						inputmode="decimal"
						bind:value={toStr}
						aria-label="To value"
					/>
				{/snippet}
			</Field>
		</div>

		<div class="anim__row">
			<Field for="anim-duration" label="Duration">
				{#snippet children()}
					<Select
						id="anim-duration"
						name="anim-duration"
						bind:value={durationChoice}
						options={durationOptions}
					/>
				{/snippet}
			</Field>
			{#if durationChoice === 'custom'}
				<Field for="anim-custom" label="Seconds">
					{#snippet children()}
						<Input
							id="anim-custom"
							name="anim-custom"
							type="number"
							inputmode="decimal"
							min="0"
							step="0.5"
							bind:value={customSecondsStr}
							aria-label="Custom duration in seconds"
						/>
					{/snippet}
				</Field>
			{/if}
		</div>

		<fieldset class="anim__easing">
			<legend class="anim__legend">Easing</legend>
			<div class="anim__easing-row">
				{#each easingPresets as preset (preset.name)}
					<button
						class="ease"
						class:ease--active={easing === preset.name}
						type="button"
						aria-pressed={easing === preset.name}
						onclick={() => (easing = preset.name)}
					>
						{preset.label}
					</button>
				{/each}
			</div>
		</fieldset>
	</div>

	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => onclose?.()}>Cancel</Button>
		<Button variant="primary" size="sm" disabled={!valid} onclick={add}>Add</Button>
	{/snippet}
</Modal>

<style>
	.anim {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-md, 1rem);
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

	.anim__easing-row {
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
</style>
