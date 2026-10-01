<script lang="ts">
	// One layer's timeline lane (slice-5b, design §12.3/§12.5). Renders a MASTER row
	// (layer name in the fixed head + a track area) and, for an animated layer, a
	// per-property SUB-TRACK row beneath it. The master shows the composite span
	// (earliest start → latest end across the layer's tracks) in teal plus thin
	// per-property ticks; gray rail when the layer isn't animated. Each sub-track bar
	// is positioned left=startMs/dur, width=durMs/dur within the track area.
	//
	// Interactions (§12.5):
	//  · single-click a bar → select it (◀ start / ▶ end grabbers appear)
	//  · drag start grabber → move start (snap 0.5s), END fixed, min 0.5s dur
	//  · drag end grabber → change end/duration (snap 0.5s), min 0.5s dur
	//  · drag bar body → translate the whole segment (snap), duration fixed, clamp [0,dur]
	//  · double-click a bar → open the info/edit popover
	//  · click the gray master track (empty) → add an animation at the clicked time
	//
	// Pointer math mirrors CanvasStage: read the track element's getBoundingClientRect
	// at pointerdown, ms = (clientX - rectLeft) / width * durationMs, then snapMs.
	// SSR-safe: all rect/pointer access is inside event handlers (browser-only).
	import { Icon } from '@sbx/core-ui/components/primitives';
	import { snapMs, type Layer, type AnimatableProperty, type AnimationTrack } from '@sbx/canvas-kit';
	import type { CanvasEditor } from './editor-state.svelte.js';
	import { layerIcon } from './palette.js';

	interface Props {
		layer: Layer;
		/** Page duration in ms — the lane's full width maps to this. */
		durationMs: number;
		editor: CanvasEditor;
		/** The currently selected bar's property (within THIS layer), or null. */
		selected: { property: AnimatableProperty } | null;
		/** Single-click a bar → select it; click the empty master track → null. */
		onselectbar: (property: AnimatableProperty | null) => void;
		/** Double-click a bar → open the info/edit popover. */
		oneditbar: (property: AnimatableProperty) => void;
		/** Click the empty gray master track → add an animation at the snapped time. */
		onaddtrack: (startMs: number) => void;
	}

	let { layer, durationMs, editor, selected, onselectbar, oneditbar, onaddtrack }: Props = $props();

	// One bar's geometry on a property sub-track, derived from its 2-keyframe track.
	interface Bar {
		property: AnimatableProperty;
		startMs: number;
		endMs: number;
		left: number; // %
		width: number; // %
	}

	const MIN_DUR = 500; // 0.5s minimum segment length (§12.5)

	function barOf(track: AnimationTrack): Bar | null {
		if (durationMs <= 0 || track.keyframes.length < 2) return null;
		const kfs = [...track.keyframes].sort((a, b) => a.time - b.time);
		const startMs = kfs[0].time;
		const endMs = kfs[kfs.length - 1].time;
		const clampedStart = Math.max(0, Math.min(startMs, durationMs));
		const clampedEnd = Math.max(clampedStart, Math.min(endMs, durationMs));
		return {
			property: track.property as AnimatableProperty,
			startMs,
			endMs,
			left: (clampedStart / durationMs) * 100,
			width: ((clampedEnd - clampedStart) / durationMs) * 100
		};
	}

	// Sub-track bars, sorted by start so the rows read top-to-bottom in time order.
	const bars = $derived.by<Bar[]>(() => {
		const tracks = layer.animations ?? [];
		return tracks
			.map(barOf)
			.filter((b): b is Bar => b !== null)
			.sort((a, b) => a.startMs - b.startMs || a.property.localeCompare(b.property));
	});

	const isAnimated = $derived(bars.length > 0);

	// Composite master span (earliest start → latest end) + per-property tick marks.
	const master = $derived.by(() => {
		if (!isAnimated || durationMs <= 0) return null;
		let start = Infinity;
		let end = -Infinity;
		for (const b of bars) {
			if (b.startMs < start) start = b.startMs;
			if (b.endMs > end) end = b.endMs;
		}
		const clampedStart = Math.max(0, Math.min(start, durationMs));
		const clampedEnd = Math.max(clampedStart, Math.min(end, durationMs));
		return {
			left: (clampedStart / durationMs) * 100,
			width: ((clampedEnd - clampedStart) / durationMs) * 100,
			ticks: bars.map((b) => (b.startMs / durationMs) * 100)
		};
	});

	// --- Pointer interaction (grabber + body drag) -------------------------
	type DragKind = 'start' | 'end' | 'body';
	let drag = $state<{
		kind: DragKind;
		property: AnimatableProperty;
		rectLeft: number;
		rectWidth: number;
		startMs: number;
		endMs: number;
		/** ms offset between the pointer's down-time and the bar start (body drag). */
		grabOffsetMs: number;
		/** True once the pointer moved past the click→drag threshold. */
		moved: boolean;
	} | null>(null);

	// Did the just-ended drag actually move? pointerup clears `drag` BEFORE the
	// synthetic click fires, so the click handler reads this persisted flag (and
	// resets it) to suppress a select after a real drag. A move-less press (pure
	// click) leaves it false → the click selects as normal.
	let lastDragMoved = false;

	/** Pointer clientX → snapped ms, using the track rect captured at pointerdown. */
	function pointerMs(clientX: number, rectLeft: number, rectWidth: number): number {
		if (rectWidth <= 0) return 0;
		const raw = ((clientX - rectLeft) / rectWidth) * durationMs;
		return snapMs(Math.min(Math.max(0, raw), durationMs));
	}

	/** Raw (un-snapped) ms at a clientX — for the body-drag grab offset. */
	function rawMs(clientX: number, rectLeft: number, rectWidth: number): number {
		if (rectWidth <= 0) return 0;
		return Math.min(Math.max(0, ((clientX - rectLeft) / rectWidth) * durationMs), durationMs);
	}

	/** Begin a grabber / body drag. `el` is the track container we measure against. */
	function startDrag(event: PointerEvent, kind: DragKind, bar: Bar, trackEl: HTMLElement): void {
		event.preventDefault();
		event.stopPropagation();
		lastDragMoved = false; // fresh interaction — clear any stale suppression flag
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		const rect = trackEl.getBoundingClientRect();
		drag = {
			kind,
			property: bar.property,
			rectLeft: rect.left,
			rectWidth: rect.width,
			startMs: bar.startMs,
			endMs: bar.endMs,
			grabOffsetMs: rawMs(event.clientX, rect.left, rect.width) - bar.startMs,
			moved: false
		};
	}

	function onDragMove(event: PointerEvent): void {
		if (!drag) return;
		drag.moved = true;
		const ms = pointerMs(event.clientX, drag.rectLeft, drag.rectWidth);
		const id = layer.id;
		if (drag.kind === 'start') {
			// Move start, keep END fixed; enforce min 0.5s (don't cross the end).
			const startMs = Math.min(ms, drag.endMs - MIN_DUR);
			const clamped = Math.max(0, startMs);
			editor.updateAnimation(id, drag.property, { startMs: clamped, durMs: drag.endMs - clamped });
		} else if (drag.kind === 'end') {
			// Move end → change duration; keep START fixed; min 0.5s.
			const endMs = Math.max(ms, drag.startMs + MIN_DUR);
			const clampedEnd = Math.min(endMs, durationMs);
			editor.updateAnimation(id, drag.property, { durMs: Math.max(MIN_DUR, clampedEnd - drag.startMs) });
		} else {
			// Body drag: translate the whole segment, duration fixed, clamp to [0,dur].
			const dur = drag.endMs - drag.startMs;
			const rawStart = rawMs(event.clientX, drag.rectLeft, drag.rectWidth) - drag.grabOffsetMs;
			const snapped = snapMs(rawStart);
			const startMs = Math.min(Math.max(0, snapped), Math.max(0, durationMs - dur));
			editor.updateAnimation(id, drag.property, { startMs });
		}
	}

	function onDragEnd(event: PointerEvent): void {
		if (!drag) return;
		try {
			(event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
		} catch {
			// capture may already be released — ignore.
		}
		lastDragMoved = drag.moved; // remember across the pending synthetic click
		drag = null;
	}

	// --- Click / select / add ----------------------------------------------
	function onBarClick(event: MouseEvent, property: AnimatableProperty): void {
		event.stopPropagation();
		// A drag that moved should not also fire a select on pointerup→click.
		if (lastDragMoved) {
			lastDragMoved = false;
			return;
		}
		onselectbar(property);
	}

	function onBarDblClick(event: MouseEvent, property: AnimatableProperty): void {
		event.stopPropagation();
		oneditbar(property);
	}

	/** Click the empty master track → add at the clicked, snapped time. */
	function onMasterTrackClick(event: MouseEvent): void {
		const el = event.currentTarget as HTMLElement;
		const rect = el.getBoundingClientRect();
		const ms = pointerMs(event.clientX, rect.left, rect.width);
		onselectbar(null);
		onaddtrack(ms);
	}
</script>

<li class="lane">
	<!-- MASTER ROW (always present) -->
	<div class="lane__row lane__row--master">
		<span class="lane__head">
			<Icon name={layerIcon(layer.type)} size="sm" />
			<span class="lane__name">{layer.name}</span>
		</span>
		<!-- Clicking the empty master track adds an animation at the clicked time. The
		     teal composite span + ticks are non-interactive (pointer-events:none) so
		     they don't swallow the add-click; the per-property sub-track bars below
		     are where selection/drag/double-click happen. -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<span class="lane__track lane__track--master" onclick={onMasterTrackClick}>
			{#if master}
				<span class="master-span" style="left: {master.left}%; width: {master.width}%;" aria-hidden="true"></span>
				{#each master.ticks as tickPct, i (i)}
					<span class="master-tick" style="left: {tickPct}%;" aria-hidden="true"></span>
				{/each}
			{:else}
				<span class="lane__rail" aria-hidden="true"></span>
			{/if}
		</span>
	</div>

	<!-- SUB-TRACK ROWS (animated layers only), one per property. -->
	{#each bars as bar (bar.property)}
		{@const isSel = selected?.property === bar.property}
		<div class="lane__row lane__row--sub">
			<span class="lane__head lane__head--sub">
				<span class="lane__subname">{bar.property}</span>
			</span>
			<span class="lane__track lane__track--sub">
				<span class="lane__rail" aria-hidden="true"></span>
				<!-- The bar body: single-click selects, double-click edits, drag moves. -->
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<!-- svelte-ignore a11y_click_events_have_key_events -->
				<span
					class="bar"
					class:bar--selected={isSel}
					style="left: {bar.left}%; width: {bar.width}%;"
					onpointerdown={(e) => {
						const el = (e.currentTarget as HTMLElement).parentElement;
						if (el) startDrag(e, 'body', bar, el);
					}}
					onpointermove={onDragMove}
					onpointerup={onDragEnd}
					onpointercancel={onDragEnd}
					onclick={(e) => onBarClick(e, bar.property)}
					ondblclick={(e) => onBarDblClick(e, bar.property)}
				>
					{#if isSel}
						<!-- ◀ start / ▶ end grabbers (only on the selected bar). -->
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<span
							class="grabber grabber--start"
							aria-label="Drag start time"
							onpointerdown={(e) => {
								const el = (e.currentTarget as HTMLElement).closest('.lane__track');
								if (el instanceof HTMLElement) startDrag(e, 'start', bar, el);
							}}
							onpointermove={onDragMove}
							onpointerup={onDragEnd}
							onpointercancel={onDragEnd}
							onclick={(e) => { e.stopPropagation(); lastDragMoved = false; }}
						></span>
						<!-- svelte-ignore a11y_no_static_element_interactions -->
						<span
							class="grabber grabber--end"
							aria-label="Drag end time"
							onpointerdown={(e) => {
								const el = (e.currentTarget as HTMLElement).closest('.lane__track');
								if (el instanceof HTMLElement) startDrag(e, 'end', bar, el);
							}}
							onpointermove={onDragMove}
							onpointerup={onDragEnd}
							onpointercancel={onDragEnd}
							onclick={(e) => { e.stopPropagation(); lastDragMoved = false; }}
						></span>
					{/if}
				</span>
			</span>
		</div>
	{/each}
</li>

<style>
	.lane {
		display: flex;
		flex-direction: column;
		border-bottom: 1px solid var(--cv-color-neutral-100, #f5f5f5);
	}

	.lane__row {
		display: flex;
		align-items: center;
		height: 32px;
	}

	.lane__row--sub {
		height: 28px;
	}

	.lane__head {
		display: flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		width: 160px;
		min-width: 160px;
		padding-left: var(--cv-space-md, 1rem);
		color: var(--cv-color-neutral-600, #505050);
	}

	.lane__head--sub {
		padding-left: calc(var(--cv-space-md, 1rem) + 1.5rem);
		color: var(--cv-color-neutral-500, #707070);
	}

	.lane__name {
		font-size: 0.8125rem;
		font-weight: 500;
	}

	/* A leading dot marks a property sub-row (matches the §12.3 "· opacity" mock). */
	.lane__subname {
		position: relative;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.lane__subname::before {
		content: '·';
		position: absolute;
		left: -0.75rem;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	.lane__track {
		position: relative;
		flex: 1;
		height: 100%;
		display: flex;
		align-items: center;
		padding: 0 var(--cv-space-md, 1rem);
	}

	.lane__track--master {
		cursor: copy; /* clicking empty space adds an animation here */
	}

	/* The gray rail spans the track interior (full timeline width). */
	.lane__rail {
		flex: 1;
		height: 6px;
		border-radius: 999px;
		background: var(--cv-color-neutral-200, #e8e8e8);
	}

	/* Master composite span — teal, non-interactive (clicks pass to the track). */
	.master-span {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		height: 8px;
		min-width: 4px;
		border-radius: 999px;
		background: var(--cv-color-primary, #333333);
		opacity: 0.35;
		pointer-events: none;
	}

	/* Thin per-property tick on the master row (one per sub-track start). */
	.master-tick {
		position: absolute;
		top: 50%;
		transform: translate(-0.5px, -50%);
		width: 2px;
		height: 14px;
		border-radius: 1px;
		background: var(--cv-color-primary, #333333);
		pointer-events: none;
	}

	/* Sub-track bar — the interactive segment. */
	.bar {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		height: 12px;
		min-width: 6px;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-primary, #333333);
		cursor: grab;
		touch-action: none;
	}

	.bar:active {
		cursor: grabbing;
	}

	.bar--selected {
		box-shadow: 0 0 0 2px var(--cv-color-surface, #fff), 0 0 0 3px var(--cv-color-primary, #333333);
	}

	/* ◀ ▶ grabbers at the bar edges (only on the selected bar). */
	.grabber {
		position: absolute;
		top: 50%;
		width: 10px;
		height: 18px;
		transform: translateY(-50%);
		border-radius: 3px;
		background: var(--cv-color-surface, #fff);
		border: 1.5px solid var(--cv-color-primary, #333333);
		touch-action: none;
	}

	.grabber--start {
		left: -5px;
		cursor: ew-resize;
	}

	.grabber--end {
		right: -5px;
		cursor: ew-resize;
	}
</style>
