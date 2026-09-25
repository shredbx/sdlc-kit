// TDD — animation authoring reducers + helpers (slice-5a, design §12.9).
// Scenarios: MC-SC-09 (animate a property), MC-SC-10 (edit/remove an animation).
//
// CASE ENUMERATION
//   addAnimation
//     TC-AN-01 success  builds a 2-keyframe track (times + values) for a property
//     TC-AN-02 edge     replaces an existing track for the SAME property (one per property)
//     TC-AN-03 regress  preserves a DIFFERENT property's track
//     TC-AN-04 failure  no-op for an unknown layer id
//     TC-AN-05 regress  does not mutate the input state
//   updateAnimation
//     TC-AN-06 success  patches durMs (the end keyframe time moves)
//     TC-AN-07 success  patches the to-value
//     TC-AN-08 failure  no-op when the property is not animated
//     TC-AN-09 failure  no-op for an unknown layer id
//     TC-AN-19 success  patching startMs moves BOTH keyframes, preserving duration (5b grabber drag)
//   removeAnimation
//     TC-AN-10 success  removes the property's track
//     TC-AN-11 edge     clears `animations` to undefined when the last track is removed
//     TC-AN-12 regress  keeps the other tracks
//     TC-AN-13 failure  no-op when the track is absent
//   snapMs
//     TC-AN-14 success  snaps to the nearest 0.5s
//     TC-AN-15 edge     clamps negatives to 0
//     TC-AN-16 edge     honours a custom step
//   easingPreset
//     TC-AN-17 success  preset name → cubic-bézier tuple / 'linear'
//   integration
//     TC-AN-18 success  add → resolveLayerAtTime interpolates the mid value

import { describe, expect, it } from 'vitest';
import { addAnimation, updateAnimation, removeAnimation, snapMs, easingPreset, easingToPresetName } from './animate.js';
import type { AnimationSpec } from './animate.js';
import { resolveLayerAtTime } from './resolve.js';
import type { EditorState } from './store.js';
import type { Document } from './types/document.js';
import type { Layer } from './types/layer.js';
import { makeTextLayer, withAnimations } from './fixtures.js';

const doc = (layer: Layer): Document => ({
	id: 'd',
	name: 'n',
	pages: [{ id: 'p1', width: 1080, height: 1350, duration: 5000, layers: [layer] }]
});
const stateWith = (layer: Layer): EditorState => ({ document: doc(layer), selectedLayerId: layer.id });

const spec = (over: Partial<AnimationSpec> = {}): AnimationSpec => ({
	property: 'opacity',
	from: 0,
	to: 1,
	startMs: 0,
	durMs: 1000,
	easing: 'linear',
	...over
});

const tracksOf = (s: EditorState) => s.document.pages[0].layers[0].animations;

describe('addAnimation (MC-SC-09)', () => {
	it('TC-AN-01 builds a 2-keyframe track with the spec times and values', () => {
		const out = addAnimation(stateWith(makeTextLayer({ id: 'a' })), 'a', spec({ from: 0.2, to: 0.9, startMs: 500, durMs: 1500 }));
		const track = tracksOf(out)!.find((t) => t.property === 'opacity')!;
		expect(track.keyframes).toEqual([
			{ time: 500, value: 0.2, easing: 'linear' },
			{ time: 2000, value: 0.9, easing: 'linear' }
		]);
	});

	it('TC-AN-02 replaces an existing track for the same property', () => {
		const seeded = withAnimations(makeTextLayer({ id: 'a' }), [
			{ property: 'opacity', keyframes: [{ time: 0, value: 1, easing: 'linear' }, { time: 100, value: 0, easing: 'linear' }] }
		]);
		const out = addAnimation(stateWith(seeded), 'a', spec({ to: 0.5, durMs: 2000 }));
		const opacityTracks = tracksOf(out)!.filter((t) => t.property === 'opacity');
		expect(opacityTracks).toHaveLength(1);
		expect(opacityTracks[0].keyframes[1]).toEqual({ time: 2000, value: 0.5, easing: 'linear' });
	});

	it('TC-AN-03 preserves a different property track', () => {
		const seeded = withAnimations(makeTextLayer({ id: 'a' }), [
			{ property: 'x', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 500, value: 50, easing: 'linear' }] }
		]);
		const out = addAnimation(stateWith(seeded), 'a', spec());
		expect(tracksOf(out)!.map((t) => t.property).sort()).toEqual(['opacity', 'x']);
	});

	it('TC-AN-04 is a no-op for an unknown layer id', () => {
		const before = stateWith(makeTextLayer({ id: 'a' }));
		expect(addAnimation(before, 'ghost', spec())).toBe(before);
	});

	it('TC-AN-05 does not mutate the input state', () => {
		const before = stateWith(makeTextLayer({ id: 'a' }));
		addAnimation(before, 'a', spec());
		expect(tracksOf(before)).toBeUndefined();
	});
});

describe('updateAnimation (MC-SC-10)', () => {
	const seeded = () =>
		stateWith(withAnimations(makeTextLayer({ id: 'a' }), [
			{ property: 'opacity', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 1000, value: 1, easing: 'linear' }] }
		]));

	it('TC-AN-06 patches durMs — the end keyframe moves', () => {
		const out = updateAnimation(seeded(), 'a', 'opacity', { durMs: 2500 });
		expect(tracksOf(out)![0].keyframes[1].time).toBe(2500);
	});

	it('TC-AN-07 patches the to-value', () => {
		const out = updateAnimation(seeded(), 'a', 'opacity', { to: 0.4 });
		expect(tracksOf(out)![0].keyframes[1].value).toBe(0.4);
	});

	it('TC-AN-08 is a no-op when the property is not animated', () => {
		const before = seeded();
		expect(updateAnimation(before, 'a', 'x', { to: 5 })).toBe(before);
	});

	it('TC-AN-09 is a no-op for an unknown layer id', () => {
		const before = seeded();
		expect(updateAnimation(before, 'ghost', 'opacity', { to: 5 })).toBe(before);
	});

	it('TC-AN-19 patching startMs moves both keyframes, preserving duration', () => {
		const out = updateAnimation(seeded(), 'a', 'opacity', { startMs: 500 });
		const kf = tracksOf(out)![0].keyframes;
		expect(kf[0].time).toBe(500);
		expect(kf[1].time).toBe(1500); // startMs(500) + preserved durMs(1000)
	});
});

describe('removeAnimation (MC-SC-10)', () => {
	it('TC-AN-10 removes the property track', () => {
		const seeded = stateWith(withAnimations(makeTextLayer({ id: 'a' }), [
			{ property: 'opacity', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 1000, value: 1, easing: 'linear' }] },
			{ property: 'x', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 500, value: 9, easing: 'linear' }] }
		]));
		const out = removeAnimation(seeded, 'a', 'opacity');
		expect(tracksOf(out)!.map((t) => t.property)).toEqual(['x']);
	});

	it('TC-AN-11 clears animations to undefined when the last track is removed', () => {
		const seeded = stateWith(withAnimations(makeTextLayer({ id: 'a' }), [
			{ property: 'opacity', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 1000, value: 1, easing: 'linear' }] }
		]));
		const out = removeAnimation(seeded, 'a', 'opacity');
		expect(tracksOf(out)).toBeUndefined();
	});

	it('TC-AN-12 keeps the other tracks', () => {
		const seeded = stateWith(withAnimations(makeTextLayer({ id: 'a' }), [
			{ property: 'opacity', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 1000, value: 1, easing: 'linear' }] },
			{ property: 'x', keyframes: [{ time: 0, value: 0, easing: 'linear' }, { time: 500, value: 9, easing: 'linear' }] }
		]));
		const xTrack = tracksOf(removeAnimation(seeded, 'a', 'opacity'))!.find((t) => t.property === 'x')!;
		expect(xTrack.keyframes).toHaveLength(2);
	});

	it('TC-AN-13 is a no-op when the track is absent', () => {
		const before = stateWith(makeTextLayer({ id: 'a' }));
		expect(removeAnimation(before, 'a', 'opacity')).toBe(before);
	});
});

describe('snapMs', () => {
	it('TC-AN-14 snaps to the nearest 0.5s', () => {
		expect(snapMs(740)).toBe(500);
		expect(snapMs(760)).toBe(1000);
		expect(snapMs(1250)).toBe(1500); // .5 rounds up
	});

	it('TC-AN-15 clamps negatives to 0', () => {
		expect(snapMs(-200)).toBe(0);
	});

	it('TC-AN-16 honours a custom step', () => {
		expect(snapMs(1100, 1000)).toBe(1000);
		expect(snapMs(1600, 1000)).toBe(2000);
	});
});

describe('easingPreset', () => {
	it('TC-AN-17 maps preset names to cubic-bézier / linear', () => {
		expect(easingPreset('in')).toEqual([0.42, 0, 1, 1]);
		expect(easingPreset('out')).toEqual([0, 0, 0.58, 1]);
		expect(easingPreset('in-out')).toEqual([0.42, 0, 0.58, 1]);
		expect(easingPreset('linear')).toBe('linear');
	});

	it('TC-AN-20 easingToPresetName reverses easingPreset (tuples + keywords)', () => {
		// Round-trips every preset.
		for (const name of ['in', 'out', 'in-out', 'linear'] as const) {
			expect(easingToPresetName(easingPreset(name))).toBe(name);
		}
		// Named keyword easings map too.
		expect(easingToPresetName('easeIn')).toBe('in');
		expect(easingToPresetName('easeOut')).toBe('out');
		expect(easingToPresetName('easeInOut')).toBe('in-out');
		// Unrecognised custom bézier falls back to linear.
		expect(easingToPresetName([0.1, 0.2, 0.3, 0.4])).toBe('linear');
	});
});

describe('integration with resolveLayerAtTime', () => {
	it('TC-AN-18 add → resolve interpolates the mid value', () => {
		const out = addAnimation(stateWith(makeTextLayer({ id: 'a', opacity: 0 })), 'a', spec({ from: 0, to: 1, startMs: 0, durMs: 1000, easing: 'linear' }));
		const layer = out.document.pages[0].layers[0];
		expect(resolveLayerAtTime(layer, 500).opacity).toBeCloseTo(0.5, 5);
		expect(resolveLayerAtTime(layer, 0).opacity).toBe(0);
		expect(resolveLayerAtTime(layer, 1000).opacity).toBe(1);
	});
});
