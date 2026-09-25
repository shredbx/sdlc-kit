<script lang="ts">
	/**
	 * ProgressRing — a compact circular completeness indicator with THREE states.
	 *
	 * Derives state from `value` (0..1, clamped; non-finite → 0):
	 *   • empty    (value === 0)      — a hollow track ring (nothing started)
	 *   • progress (0 < value < 1)    — a swept arc proportional to value
	 *   • complete (value >= 1)       — a SOLID filled disc + a check glyph
	 *
	 * Renders an accessible `role="progressbar"`; `aria-valuenow` is value×100
	 * (0..100), and `aria-label` is suffixed with the state ("complete" /
	 * "<n>% complete" / "not started") so the state is conveyed non-visually
	 * (state is never communicated by colour alone — WCAG 1.4.1).
	 *
	 * COLOURS ARE CONSUMER-SET via CSS custom properties — this primitive is
	 * platform-neutral and carries NO brand tokens. The consumer sets, on this
	 * element or an ancestor:
	 *   --progress-ring-track     (the faint base ring; default #e8e8e0)
	 *   --progress-ring-empty     (the empty-state ring; default currentColor)
	 *   --progress-ring-progress  (the arc; default currentColor)
	 *   --progress-ring-complete  (the filled disc; default currentColor)
	 *   --progress-ring-check     (the checkmark; default #fff)
	 *
	 * @example
	 *   <ProgressRing value={0.6} />              // 60% — arc
	 *   <ProgressRing value={1} size={22} />      // complete — disc + check
	 */

	interface Props {
		/** Completeness fraction 0..1. Clamped; non-finite → 0. */
		value: number;
		/** Diameter in px (default 16). */
		size?: number;
		/** Stroke width in px (default 2.5). */
		stroke?: number;
		/** Accessible label prefix (default "completeness"). State is appended. */
		label?: string;
		/** Extra class on the wrapper. */
		class?: string;
	}

	let { value, size = 16, stroke = 2.5, label = 'completeness', class: className = '' }: Props =
		$props();

	// Clamp to [0,1]; treat non-finite (NaN / Infinity) as 0.
	const fraction = $derived(Number.isFinite(value) ? Math.min(1, Math.max(0, value)) : 0);
	const percent = $derived(Math.round(fraction * 100));

	// Three states drive both the visual and the a11y suffix.
	const state = $derived<'empty' | 'progress' | 'complete'>(
		fraction >= 1 ? 'complete' : fraction <= 0 ? 'empty' : 'progress'
	);
	const stateText = $derived(
		state === 'complete' ? 'complete' : state === 'empty' ? 'not started' : `${percent}% complete`
	);

	// Geometry: a circle inset by half the stroke so the arc isn't clipped.
	const radius = $derived((size - stroke) / 2);
	const center = $derived(size / 2);
	const circumference = $derived(2 * Math.PI * radius);
	// Dash offset shrinks as the arc fills (dasharray = full circumference).
	const dashOffset = $derived(circumference * (1 - fraction));

	// Filled disc radius for the complete state — fill nearly the whole box so
	// the check sits on a solid coin.
	const discRadius = $derived(size / 2);
	// Checkmark polyline as fractions of the box, so it scales with `size`.
	const check = $derived(
		`M ${size * 0.3} ${size * 0.52} L ${size * 0.44} ${size * 0.66} L ${size * 0.71} ${size * 0.35}`
	);
	const checkStroke = $derived(Math.max(1.4, size * 0.11));
</script>

<span
	class="progress-ring progress-ring--{state} {className}"
	role="progressbar"
	aria-valuemin="0"
	aria-valuemax="100"
	aria-valuenow={percent}
	aria-label={`${label}: ${stateText}`}
	style="--_size: {size}px;"
>
	<svg width={size} height={size} viewBox="0 0 {size} {size}" aria-hidden="true">
		{#if state === 'complete'}
			<circle class="progress-ring__disc" cx={center} cy={center} r={discRadius} />
			<path
				class="progress-ring__check"
				d={check}
				fill="none"
				stroke-width={checkStroke}
				stroke-linecap="round"
				stroke-linejoin="round"
			/>
		{:else}
			<circle
				class="progress-ring__track"
				class:is-empty={state === 'empty'}
				cx={center}
				cy={center}
				r={radius}
				fill="none"
				stroke-width={stroke}
			/>
			{#if state === 'progress'}
				<circle
					class="progress-ring__arc"
					cx={center}
					cy={center}
					r={radius}
					fill="none"
					stroke-width={stroke}
					stroke-linecap="round"
					stroke-dasharray={circumference}
					stroke-dashoffset={dashOffset}
					transform="rotate(-90 {center} {center})"
				/>
			{/if}
		{/if}
	</svg>
</span>

<style>
	.progress-ring {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: var(--_size, 16px);
		height: var(--_size, 16px);
		flex-shrink: 0;
		line-height: 0;
	}

	/* Faint base ring (progress state); slightly stronger in the empty state so a
	   not-started section still reads as a defined ring, not a smudge. */
	.progress-ring__track {
		stroke: var(--progress-ring-track, #e8e8e0);
	}
	.progress-ring__track.is-empty {
		stroke: var(--progress-ring-empty, currentColor);
	}

	.progress-ring__arc {
		stroke: var(--progress-ring-progress, currentColor);
		transition: stroke-dashoffset 0.25s ease;
	}

	.progress-ring__disc {
		fill: var(--progress-ring-complete, currentColor);
	}
	.progress-ring__check {
		stroke: var(--progress-ring-check, #fff);
	}
</style>
