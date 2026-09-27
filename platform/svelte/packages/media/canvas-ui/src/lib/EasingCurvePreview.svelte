<script lang="ts">
	// Tiny SVG cubic-bézier preview (slice-5b, §12.5 "small curve preview"). Draws
	// the active easing curve from bottom-left (0,H) to top-right (W,0) so the popover
	// shows, at a glance, the shape of the selected preset. Pure presentational — no
	// editor/kit-state dependency, only the kit's `easingPreset` tuple lookup. SSR-safe
	// (SVG renders identically on the server; no DOM/window access).
	import { easingPreset, type EasingPresetName } from '@sbx/canvas-kit';

	interface Props {
		/** Which preset curve to draw. */
		easing: EasingPresetName;
		/** Box size in px (square-ish viewport). */
		size?: { w: number; h: number };
	}

	let { easing, size = { w: 64, h: 44 } }: Props = $props();

	const W = $derived(size.w);
	const H = $derived(size.h);

	// The SVG path. y is flipped (SVG y grows downward) so progress=1 sits at the top.
	// 'linear' has no bézier tuple — draw the straight diagonal explicitly.
	const path = $derived.by(() => {
		const e = easingPreset(easing);
		if (!Array.isArray(e)) return `M 0 ${H} L ${W} 0`; // linear
		const [x1, y1, x2, y2] = e;
		const c1x = x1 * W;
		const c1y = H - y1 * H;
		const c2x = x2 * W;
		const c2y = H - y2 * H;
		return `M 0 ${H} C ${c1x} ${c1y} ${c2x} ${c2y} ${W} 0`;
	});
</script>

<svg
	class="curve"
	width={W}
	height={H}
	viewBox={`0 0 ${W} ${H}`}
	role="img"
	aria-label={`${easing} easing curve`}
>
	<!-- Reference diagonal (linear baseline) so the eased curve's bow reads clearly. -->
	<line class="curve__baseline" x1="0" y1={H} x2={W} y2="0" />
	<path class="curve__path" d={path} fill="none" />
</svg>

<style>
	.curve {
		display: block;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-neutral-50, #fafafa);
	}

	.curve__baseline {
		stroke: var(--cv-color-neutral-200, #e8e8e8);
		stroke-width: 1;
		stroke-dasharray: 3 3;
	}

	.curve__path {
		stroke: var(--cv-color-primary, #333333);
		stroke-width: 2;
		stroke-linecap: round;
	}
</style>
