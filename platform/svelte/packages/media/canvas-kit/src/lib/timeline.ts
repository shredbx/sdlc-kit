// Playback / storyboard sequencing — Media Canvas extension (D10/D12).
// Pages act as video frames (Page.duration); the timeline maps a global playhead
// time to a (page, local time) so playback can drive resolveLayerAtTime per page.
// Pure + framework-agnostic.

import type { Page } from './types/document.js';

/** Default frame duration (ms) for a page that declares none. */
export const DEFAULT_FRAME_MS = 4000;

/** Where the global playhead lands: which page and the time within it. */
export interface FrameAt {
	pageIndex: number;
	/** Time within the page, in ms, clamped to [0, page duration]. */
	localTime: number;
}

const effective = (page: Page): number => page.duration ?? DEFAULT_FRAME_MS;

/** Total document duration = sum of each page's effective duration (MC-SC-12). */
export function totalDuration(pages: Page[]): number {
	return pages.reduce((sum, p) => sum + effective(p), 0);
}

/**
 * Map a global playhead time to a page + local time (MC-SC-11 / MC-SC-12).
 * Without loop: clamps to [0, total]. With loop: wraps modulo total.
 * A frame boundary belongs to the START of the next frame.
 */
export function frameAtTime(
	pages: Page[],
	globalTime: number,
	opts: { loop?: boolean } = {}
): FrameAt {
	const total = totalDuration(pages);
	let time: number;
	if (opts.loop && total > 0) {
		time = ((globalTime % total) + total) % total;
	} else {
		time = Math.min(Math.max(globalTime, 0), total);
	}

	let acc = 0;
	for (let i = 0; i < pages.length; i++) {
		const dur = effective(pages[i]);
		if (time < acc + dur) return { pageIndex: i, localTime: time - acc };
		acc += dur;
	}

	// time === total (clamped end, no loop) → end of the last frame.
	const lastIndex = pages.length - 1;
	return { pageIndex: lastIndex, localTime: effective(pages[lastIndex]) };
}
