<script lang="ts">
	// Collapsible bottom timeline (D10) — transport (5a) + rich track editor (5b).
	// Collapsed by default: a thin strip with the Animate toggle (the top chrome bar
	// is gone, D17). Expanded: a working transport (Play/Pause → editor.togglePlay,
	// time readout, Loop, a scrubber bound to editor.seek, a Page-duration Select), a
	// 0.5s ruler, and ONE <TimelineLane> per non-system layer (G1). Each lane renders
	// a master row (composite span + ticks, gray when un-animated) plus per-property
	// sub-tracks with ◀▶ grabbers (5b). Single-click a bar selects it (state held
	// here); double-click opens the info/edit popover; clicking an empty master track
	// adds an animation at the clicked time — both routed up to EditorShell, which
	// mounts AddAnimationDialog + AnimationInfoPopover. A read-only playhead marks
	// editor.time. `expanded` is owned by EditorShell.
	import { Icon, Select } from '@sbx/core-ui/components/primitives';
	import type { Layer, Page, AnimatableProperty } from '@sbx/canvas-kit';
	import type { CanvasEditor } from './editor-state.svelte.js';
	import TimelineLane from './TimelineLane.svelte';

	interface Props {
		page: Page;
		/** The reactive editor — transport, scrub, page-duration all route here. */
		editor: CanvasEditor;
		expanded?: boolean;
		ontoggle?: () => void;
		/** Click an empty master track → add an animation on `layerId` at `startMs`. */
		onaddanimation?: (layerId: string, startMs: number) => void;
		/** Double-click a bar → open the info/edit popover for that layer's property. */
		oneditanimation?: (layerId: string, property: AnimatableProperty) => void;
	}

	let { page, editor, expanded = false, ontoggle, onaddanimation, oneditanimation }: Props = $props();

	// The selected bar — { layerId, property } | null — owned here (5b). Drives the
	// grabber affordance on exactly one bar across all lanes.
	let selectedBar = $state<{ layerId: string; property: AnimatableProperty } | null>(null);

	function selectedFor(layerId: string): { property: AnimatableProperty } | null {
		return selectedBar?.layerId === layerId ? { property: selectedBar.property } : null;
	}

	function handleSelectBar(layerId: string, property: AnimatableProperty | null): void {
		selectedBar = property ? { layerId, property } : null;
	}

	const durationMs = $derived(editor.pageDurationMs);
	const currentLabel = $derived(formatTime(editor.time));
	const durationLabel = $derived(formatTime(durationMs));

	/** mm:ss.t — minutes:seconds with a tenths place (timeline scale). */
	function formatTime(ms: number): string {
		const totalSeconds = ms / 1000;
		const m = Math.floor(totalSeconds / 60);
		const s = Math.floor(totalSeconds % 60);
		const t = Math.floor((totalSeconds * 10) % 10);
		return `${m}:${String(s).padStart(2, '0')}.${t}`;
	}

	// Page-duration choices (§12.3). Value = ms (string for the Select).
	const durationOptions = [
		{ value: '3000', label: '3s' },
		{ value: '4000', label: '4s' },
		{ value: '5000', label: '5s' },
		{ value: '8000', label: '8s' },
		{ value: '10000', label: '10s' },
		{ value: '15000', label: '15s' }
	];
	const durationValue = $derived(String(durationMs));

	function onDurationChange(value: string): void {
		const ms = Number(value);
		if (Number.isFinite(ms)) editor.setPageDuration(ms);
	}

	function onScrub(value: string): void {
		const ms = Number(value);
		if (Number.isFinite(ms)) editor.seek(ms);
	}

	// Lanes: exclude system layers (G1), front-most first (matches the Layers list).
	const lanes = $derived<Layer[]>([...page.layers].filter((l) => !l.system).reverse());

	// 0.5s ruler marks across the page duration (§12.3). Whole seconds are labelled;
	// half-second marks are minor ticks.
	const rulerMarks = $derived.by(() => {
		const steps = Math.max(1, Math.round(durationMs / 500));
		const marks: { pct: number; label: string | null }[] = [];
		for (let i = 0; i <= steps; i++) {
			const ms = i * 500;
			const isSecond = ms % 1000 === 0;
			marks.push({
				pct: durationMs > 0 ? (ms / durationMs) * 100 : 0,
				label: isSecond ? `${ms / 1000}.0s` : null
			});
		}
		return marks;
	});

	// Playhead position as a % of the page duration (read-only indicator).
	const playheadPct = $derived(durationMs > 0 ? (editor.time / durationMs) * 100 : 0);
</script>

<section class="timeline" class:timeline--expanded={expanded}>
	{#if !expanded}
		<div class="timeline__collapsed">
			<button class="timeline__handle" type="button" aria-label="Expand timeline" onclick={() => ontoggle?.()}>
				<Icon name="chevron-up" size="sm" />Timeline
			</button>
			<span class="timeline__hint">{durationLabel}</span>
			<!-- Animate toggle moved here from the removed top bar (D17). -->
			<button class="timeline__animate" type="button" onclick={() => ontoggle?.()}>
				<Icon name="play" size="sm" />Animate
			</button>
		</div>
	{:else}
		<header class="transport">
			<button class="transport__btn" type="button" aria-label="Collapse timeline" onclick={() => ontoggle?.()}>
				<Icon name="chevron-down" size="sm" />
			</button>
			<button
				class="transport__btn"
				type="button"
				aria-label={editor.playing ? 'Pause' : 'Play'}
				aria-pressed={editor.playing}
				onclick={() => editor.togglePlay()}
			>
				<Icon name={editor.playing ? 'pause' : 'play'} size="sm" />
			</button>
			<span class="transport__time">{currentLabel} / {durationLabel}</span>
			<button
				class="transport__btn"
				class:transport__btn--active={editor.loop}
				type="button"
				aria-label="Loop playback"
				aria-pressed={editor.loop}
				onclick={() => (editor.loop = !editor.loop)}
			>
				<Icon name="refresh-cw" size="sm" />
			</button>

			<!-- svelte-ignore a11y_no_redundant_roles -->
			<input
				class="transport__scrubber"
				type="range"
				min="0"
				max={durationMs}
				step="500"
				value={editor.time}
				aria-label="Scrub playhead"
				oninput={(e) => onScrub((e.currentTarget as HTMLInputElement).value)}
			/>

			<label class="transport__duration">
				<span class="transport__duration-label">Page duration</span>
				<Select
					name="page-duration"
					size="sm"
					value={durationValue}
					options={durationOptions}
					onchange={onDurationChange}
				/>
			</label>
		</header>

		<div class="tracks">
			<div class="tracks__ruler">
				<span class="tracks__ruler-label">LAYERS</span>
				<div class="tracks__marks">
					{#each rulerMarks as mark, i (i)}
						<span class="tracks__mark" class:tracks__mark--minor={!mark.label} style="left: {mark.pct}%;">
							{#if mark.label}<span class="tracks__mark-label">{mark.label}</span>{/if}
						</span>
					{/each}
				</div>
			</div>
			<ul class="tracks__lanes">
				{#each lanes as layer (layer.id)}
					<TimelineLane
						{layer}
						{durationMs}
						{editor}
						selected={selectedFor(layer.id)}
						onselectbar={(property) => handleSelectBar(layer.id, property)}
						oneditbar={(property) => oneditanimation?.(layer.id, property)}
						onaddtrack={(startMs) => onaddanimation?.(layer.id, startMs)}
					/>
				{/each}
			</ul>
			<!-- Read-only playhead over the lanes (the scrubber drives seek, 5a). -->
			<div class="tracks__playhead" style="left: calc(160px + (100% - 160px - 2rem) * {playheadPct / 100} + 1rem);" aria-hidden="true"></div>
		</div>
	{/if}
</section>

<style>
	.timeline {
		background: var(--cv-color-surface, #fff);
		border-top: 1px solid var(--cv-border-color, #e0e0e0);
	}

	.timeline--expanded {
		display: flex;
		flex-direction: column;
		height: 220px;
	}

	.timeline__collapsed {
		display: flex;
		align-items: center;
		gap: var(--cv-space-md, 1rem);
		width: 100%;
		height: 36px;
		padding: 0 var(--cv-space-md, 1rem);
		color: var(--cv-color-neutral-600, #505050);
	}

	.timeline__handle {
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		border: none;
		background: transparent;
		color: inherit;
		font-size: 0.8125rem;
		font-weight: 600;
		cursor: pointer;
	}

	.timeline__hint {
		font-size: 0.75rem;
		color: var(--cv-color-neutral-400, #a0a0a0);
		font-variant-numeric: tabular-nums;
	}

	/* Animate toggle right-aligned (hard rule: CTAs right). */
	.timeline__animate {
		margin-left: auto;
		display: inline-flex;
		align-items: center;
		gap: 0.375rem;
		padding: 0.25rem 0.625rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-primary, #333333);
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
	}

	.timeline__animate:hover {
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.transport {
		display: flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		height: 44px;
		padding: 0 var(--cv-space-md, 1rem);
		border-bottom: 1px solid var(--cv-border-color, #e0e0e0);
	}

	.transport__btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-600, #505050);
		cursor: pointer;
	}

	.transport__btn:hover {
		background: var(--cv-color-neutral-100, #f5f5f5);
	}

	.transport__btn--active {
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
		color: var(--cv-color-primary, #333333);
	}

	.transport__time {
		font-size: 0.8125rem;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		color: var(--cv-color-neutral-800, #202020);
	}

	.transport__scrubber {
		flex: 1;
		min-width: 0;
		height: 4px;
		margin: 0 var(--cv-space-sm, 0.5rem);
		accent-color: var(--cv-color-primary, #333333);
		cursor: pointer;
	}

	.transport__duration {
		display: inline-flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		/* Never let the flex row squeeze this control: its label is nowrap, so any
		   shrink lands entirely on the Select and clips the value ("5⊘"). */
		flex-shrink: 0;
	}

	/* Floor the Select width so the value + chevron always render in full. */
	.transport__duration :global(select) {
		min-width: 4.5rem;
	}

	.transport__duration-label {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
		white-space: nowrap;
	}

	.tracks {
		position: relative;
		flex: 1;
		display: flex;
		flex-direction: column;
		overflow-y: auto;
	}

	.tracks__ruler {
		display: flex;
		align-items: center;
		height: 28px;
		border-bottom: 1px solid var(--cv-border-color, #e0e0e0);
	}

	.tracks__ruler-label {
		width: 160px;
		min-width: 160px;
		padding-left: var(--cv-space-md, 1rem);
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	.tracks__marks {
		position: relative;
		flex: 1;
		height: 100%;
		margin: 0 var(--cv-space-md, 1rem);
	}

	.tracks__mark {
		position: absolute;
		top: 0;
		bottom: 0;
		display: flex;
		align-items: flex-end;
		width: 1px;
		background: var(--cv-color-neutral-200, #e8e8e8);
		transform: translateX(-0.5px);
	}

	.tracks__mark--minor {
		top: 50%;
		background: var(--cv-color-neutral-100, #f5f5f5);
	}

	.tracks__mark-label {
		position: absolute;
		bottom: 0.25rem;
		left: 0.25rem;
		font-size: 0.6875rem;
		color: var(--cv-color-neutral-400, #a0a0a0);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	.tracks__lanes {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.tracks__playhead {
		position: absolute;
		top: 28px; /* below the ruler */
		bottom: 0;
		width: 2px;
		/* Brand red as the conventional playhead indicator — NOT semantically the
		   danger palette, so use the literal token to avoid theme coupling. */
		background: #e5392b;
		pointer-events: none;
	}
</style>
