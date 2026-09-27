<script lang="ts">
	// Drag-to-adjust unit suffix for the Inspector's numeric rows (user directive
	// 2026-06-07) — the "px" badge doubles as a horizontal scrubber, the settled
	// AE/Figma/Blender convention: SINGLE axis, right = increase, left = decrease,
	// ⟷ cursor announces the directions. The number input stays the accessible
	// control; this is a redundant pointer-only affordance (aria-hidden). Shift
	// scrubs ×10, Alt ×0.1 (fine). The live value is clamped INSIDE the
	// accumulator, so dragging past a limit has no dead zone — reversing
	// direction responds immediately. Emits only on actual change (a click
	// without travel never dirties the document — same guard the
	// SegmentedControl re-click fix needed).
	interface Props {
		/** Suffix text — px · % · ° · /10. */
		unit: string;
		/** Current model/display value the gesture starts from. */
		value: number;
		/** Units added per pixel of horizontal travel (before modifiers). */
		step?: number;
		min?: number;
		max?: number;
		/** Live per-move emission (rounded to 2 decimals, clamped). */
		onscrub?: (next: number) => void;
	}

	let { unit, value, step = 1, min, max, onscrub }: Props = $props();

	let scrubbing = $state(false);
	// Gesture-local accumulator — deliberately NOT reactive: the prop round-trips
	// through the document patch while dragging, and re-reading it would fight
	// the clamp/rounding (feedback loop).
	let live = 0;
	let lastX = 0;
	let lastEmit = 0;

	function clamp(n: number): number {
		if (min !== undefined && n < min) return min;
		if (max !== undefined && n > max) return max;
		return n;
	}

	function down(e: PointerEvent): void {
		// Primary button only (CanvasStage idiom) — a right-click opens the context
		// menu on mousedown and its pointerup never reaches the span, which would
		// stick the global scrub state on.
		if (e.button !== 0) return;
		// preventDefault keeps focus/text-selection off the input underneath.
		e.preventDefault();
		(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
		live = value;
		lastEmit = value;
		lastX = e.clientX;
		scrubbing = true;
	}

	function move(e: PointerEvent): void {
		if (!scrubbing) return;
		const dx = e.clientX - lastX;
		lastX = e.clientX;
		const eff = step * (e.shiftKey ? 10 : e.altKey ? 0.1 : 1);
		live = clamp(live + dx * eff);
		const next = Math.round(live * 100) / 100; // kill float dust
		if (next !== lastEmit) {
			lastEmit = next;
			onscrub?.(next);
		}
	}

	function up(): void {
		scrubbing = false;
	}

	// During the drag the ⟷ cursor is forced GLOBALLY — pointer capture keeps the
	// events coming, but the rendered cursor still follows whatever the pointer is
	// over, so without this it would flicker to text/default crossing the canvas.
	// Effect cleanup also covers unmount mid-drag.
	$effect(() => {
		if (!scrubbing) return;
		document.body.classList.add('cv-scrubbing');
		return () => document.body.classList.remove('cv-scrubbing');
	});
</script>

<span
	class="scrub"
	class:scrub--active={scrubbing}
	aria-hidden="true"
	title="Drag to adjust · Shift ×10 · Alt ×0.1"
	onpointerdown={down}
	onpointermove={move}
	onpointerup={up}
	onpointercancel={up}
	onlostpointercapture={up}
>{unit}</span>

<style>
	.scrub {
		/* The Input `right` slot disables pointer events (clicks pass through to
		   the field) — re-enable them for the scrubber only. */
		pointer-events: auto;
		cursor: ew-resize;
		touch-action: none;
		user-select: none;
		-webkit-user-select: none;
		/* Idle face matches the Inspector's plain .unit suffix exactly. */
		font-size: 0.6875rem;
		color: var(--cv-color-neutral-400, #a0a0a0);
		padding: 1px 3px;
		margin-right: -3px; /* padding grows the hit area without shifting the glyph */
		border-radius: 3px;
		transition:
			color 120ms ease,
			background 120ms ease;
	}

	.scrub:hover {
		color: var(--cv-color-primary, #333333);
		background: var(--cv-color-neutral-100, #f5f5f5);
	}

	.scrub--active {
		color: var(--cv-color-surface, #fff);
		background: var(--cv-color-primary, #333333);
	}

	/* `cursor` doesn't cascade past elements that set their own (inputs are
	   cursor:text), hence the * + !important while a scrub is live. */
	:global(body.cv-scrubbing),
	:global(body.cv-scrubbing *) {
		cursor: ew-resize !important;
		user-select: none !important;
	}
</style>
