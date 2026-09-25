// Animation authoring reducers + helpers — slice-5a (D19 / design §12.9).
// A user "animated property" = a 2-keyframe AnimationTrack (start value + end
// value over [startMs .. startMs+durMs] with one easing curve). These reducers
// build/edit/remove those tracks; `resolveLayerAtTime` (resolve.ts) already
// interpolates them — NO engine change. Pure + framework-agnostic: each returns a
// NEW EditorState via the tested `updateLayer` reducer (immutability + unknown-id
// no-op inherited for free). One track per property per layer (add replaces).

import type { EditorState } from './store.js';
import { updateLayer } from './store.js';
import type { Document } from './types/document.js';
import type { Layer } from './types/layer.js';
import type { AnimationTrack, Easing } from './types/animation.js';

/** The animatable, numeric/continuous layer properties (design §12.6). The four
 *  crop edges (2026-06-07) animate as plain scalars — wipe/reveal effects. */
export type AnimatableProperty =
	| 'x'
	| 'y'
	| 'width'
	| 'height'
	| 'opacity'
	| 'rotation'
	| 'scale'
	| 'crop_top'
	| 'crop_right'
	| 'crop_bottom'
	| 'crop_left';

/** The authoring shape of one property animation (what the popup collects). */
export interface AnimationSpec {
	property: AnimatableProperty;
	/** Start value (the resolver holds this before startMs). */
	from: number;
	/** End value (held after startMs+durMs). */
	to: number;
	/** Segment start time in ms (relative to the page). */
	startMs: number;
	/** Segment length in ms. */
	durMs: number;
	/** Curve applied across the segment. */
	easing: Easing;
}

/**
 * Add (or REPLACE) the animation for a single property on a layer (§12.4). Builds
 * a 2-keyframe track [start..start+dur]; any existing track for the same property
 * is replaced (one track per property). No-op for an unknown layer id.
 */
export function addAnimation(state: EditorState, layerId: string, spec: AnimationSpec): EditorState {
	const layer = getLayer(state.document, layerId);
	if (!layer) return state;
	const others = (layer.animations ?? []).filter((t) => t.property !== spec.property);
	const animations = [...others, specToTrack(spec)];
	return updateLayer(state, layerId, { animations });
}

/**
 * Patch the existing animation for a property (§12.5 grabber-drag / popup edits).
 * Reads the current 2-keyframe spec, merges the patch, rebuilds the track. No-op
 * when the layer or the property's track is absent.
 */
export function updateAnimation(
	state: EditorState,
	layerId: string,
	property: AnimatableProperty,
	patch: Partial<Omit<AnimationSpec, 'property'>>
): EditorState {
	const layer = getLayer(state.document, layerId);
	const current = layer?.animations?.find((t) => t.property === property);
	if (!layer || !current) return state;
	const merged: AnimationSpec = { ...trackToSpec(current, property), ...patch };
	const animations = layer.animations!.map((t) => (t.property === property ? specToTrack(merged) : t));
	return updateLayer(state, layerId, { animations });
}

/**
 * Remove a property's animation (§12.5 delete). When it was the layer's last
 * track, `animations` is cleared to undefined (a static layer again). No-op for an
 * unknown layer / absent track.
 */
export function removeAnimation(state: EditorState, layerId: string, property: AnimatableProperty): EditorState {
	const layer = getLayer(state.document, layerId);
	if (!layer?.animations?.some((t) => t.property === property)) return state;
	const remaining = layer.animations.filter((t) => t.property !== property);
	return updateLayer(state, layerId, { animations: remaining.length ? remaining : undefined });
}

// --- authoring helpers -----------------------------------------------------

/** Snap a time (ms) to the timeline grid (default 0.5s, §12.3). Never negative. */
export function snapMs(ms: number, step = 500): number {
	if (!(step > 0)) return Math.max(0, ms);
	return Math.max(0, Math.round(ms / step) * step);
}

/** UI easing preset name → cubic-bézier (CSS semantics), per §12.7. */
export type EasingPresetName = 'in' | 'out' | 'in-out' | 'linear';

/** Resolve a popup easing preset to a concrete `Easing` value the keyframes store. */
export function easingPreset(name: EasingPresetName): Easing {
	switch (name) {
		case 'in':
			return [0.42, 0, 1, 1];
		case 'out':
			return [0, 0, 0.58, 1];
		case 'in-out':
			return [0.42, 0, 0.58, 1];
		case 'linear':
		default:
			return 'linear';
	}
}

/**
 * Reverse of `easingPreset` — map a stored `Easing` back to its preset name so the
 * info popover (5b) can show which curve button is active. Handles both the named
 * keywords and our preset bézier tuples; an unrecognised custom bézier falls back
 * to 'linear'.
 */
export function easingToPresetName(easing: Easing): EasingPresetName {
	if (easing === 'easeIn') return 'in';
	if (easing === 'easeOut') return 'out';
	if (easing === 'easeInOut') return 'in-out';
	if (easing === 'linear') return 'linear';
	const [a, b, c, d] = easing;
	if (a === 0.42 && b === 0 && c === 1 && d === 1) return 'in';
	if (a === 0 && b === 0 && c === 0.58 && d === 1) return 'out';
	if (a === 0.42 && b === 0 && c === 0.58 && d === 1) return 'in-out';
	return 'linear';
}

// --- internal --------------------------------------------------------------

function specToTrack(spec: AnimationSpec): AnimationTrack {
	return {
		property: spec.property,
		keyframes: [
			{ time: spec.startMs, value: spec.from, easing: spec.easing },
			{ time: spec.startMs + spec.durMs, value: spec.to, easing: spec.easing }
		]
	};
}

/** Derive the authoring spec from a 2-keyframe track (first = from, last = to). */
function trackToSpec(track: AnimationTrack, property: AnimatableProperty): AnimationSpec {
	const kfs = [...track.keyframes].sort((a, b) => a.time - b.time);
	const first = kfs[0];
	const last = kfs[kfs.length - 1];
	return {
		property,
		from: Number(first.value),
		to: Number(last.value),
		startMs: first.time,
		durMs: last.time - first.time,
		easing: first.easing
	};
}

function getLayer(doc: Document, layerId: string): Layer | undefined {
	for (const page of doc.pages) {
		const layer = page.layers.find((l) => l.id === layerId);
		if (layer) return layer;
	}
	return undefined;
}
