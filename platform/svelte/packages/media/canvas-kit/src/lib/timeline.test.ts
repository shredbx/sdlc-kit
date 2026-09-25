// TDD RED — playback / storyboard sequencing (FDD4.PD.TEST_RED).
// Scenarios: MC-SC-11 (playback over a page duration, loop),
//            MC-SC-12 (multi-frame storyboard plays in sequence).
//
// CASE ENUMERATION
//   TC-TL-01 success  single page, t mid-duration → that page, local time (MC-SC-11)
//   TC-TL-02 edge     t at the end clamps to the page end (no loop)
//   TC-TL-03 edge     loop wraps t past the duration (MC-SC-11)
//   TC-TL-04 edge     negative t clamps to 0
//   TC-TL-05 success  two frames, t in frame 2 → pageIndex 1 + local time (MC-SC-12)
//   TC-TL-06 success  two frames, t in frame 1 → pageIndex 0 + local time (MC-SC-12)
//   TC-TL-07 edge     a frame boundary belongs to the start of the next frame
//   TC-TL-08 success  totalDuration sums every page's duration
//   TC-TL-09 edge     loop wraps across the whole multi-frame timeline

import { describe, expect, it } from 'vitest';
import { frameAtTime, totalDuration } from './timeline.js';
import type { Page } from './types/document.js';

const page = (id: string, duration: number): Page => ({ id, width: 1080, height: 1350, layers: [], duration });

describe('frameAtTime — single page (MC-SC-11)', () => {
	const pages = [page('p1', 4000)];

	it('TC-TL-01 maps a mid-duration time to the page and local time', () => {
		expect(frameAtTime(pages, 2000)).toEqual({ pageIndex: 0, localTime: 2000 });
	});

	it('TC-TL-02 clamps to the page end without loop', () => {
		expect(frameAtTime(pages, 99999)).toEqual({ pageIndex: 0, localTime: 4000 });
	});

	it('TC-TL-03 wraps past the duration when looping', () => {
		expect(frameAtTime(pages, 5000, { loop: true })).toEqual({ pageIndex: 0, localTime: 1000 });
	});

	it('TC-TL-04 clamps a negative time to 0', () => {
		expect(frameAtTime(pages, -500)).toEqual({ pageIndex: 0, localTime: 0 });
	});
});

describe('frameAtTime — storyboard sequence (MC-SC-12)', () => {
	const pages = [page('p1', 4000), page('p2', 3000)];

	it('TC-TL-05 lands in frame 2 with the correct local time', () => {
		expect(frameAtTime(pages, 5000)).toEqual({ pageIndex: 1, localTime: 1000 });
	});

	it('TC-TL-06 lands in frame 1 with the correct local time', () => {
		expect(frameAtTime(pages, 2000)).toEqual({ pageIndex: 0, localTime: 2000 });
	});

	it('TC-TL-07 puts a frame boundary at the start of the next frame', () => {
		expect(frameAtTime(pages, 4000)).toEqual({ pageIndex: 1, localTime: 0 });
	});

	it('TC-TL-09 wraps across the whole timeline when looping', () => {
		// total = 7000; t=8000 → 1000 → still inside frame 1
		expect(frameAtTime(pages, 8000, { loop: true })).toEqual({ pageIndex: 0, localTime: 1000 });
	});
});

describe('totalDuration (MC-SC-12)', () => {
	it('TC-TL-08 sums every page duration', () => {
		expect(totalDuration([page('p1', 4000), page('p2', 3000)])).toBe(7000);
	});
});
