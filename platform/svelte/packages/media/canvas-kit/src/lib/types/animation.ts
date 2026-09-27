// Animation model — Media Canvas extension (D2, Lottie-style property tracks).
// A Layer stays static (its base values); animation is additive: zero or more
// tracks keyed by property path, each a list of keyframes with an easing curve.
// `resolveLayerAtTime(layer, t)` collapses tracks into concrete values at time t.

/**
 * Easing curve applied from a keyframe to the next.
 * Named presets, or an explicit cubic-bezier `[x1, y1, x2, y2]` (CSS semantics,
 * with implicit P0 = (0,0) and P3 = (1,1)).
 */
export type Easing = 'linear' | 'easeIn' | 'easeOut' | 'easeInOut' | [number, number, number, number];

/** A single control point on a property track. */
export interface Keyframe {
	/** Time in ms, relative to the page/frame start. */
	time: number;
	/** Numeric values interpolate; string values step (hold the previous keyframe). */
	value: number | string;
	/** Curve from THIS keyframe to the next. */
	easing: Easing;
}

/** One animated property of a layer. */
export interface AnimationTrack {
	/** Property path on the layer: 'x' | 'y' | 'opacity' | 'rotation' | 'scale' | 'font_size' | … */
	property: string;
	/** Keyframes, expected sorted by time (the resolver tolerates unsorted input). */
	keyframes: Keyframe[];
}
