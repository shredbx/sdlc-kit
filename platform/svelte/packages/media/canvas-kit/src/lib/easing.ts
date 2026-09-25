// Easing evaluation — maps linear progress p∈[0,1] to eased progress.
// Named presets use the CSS cubic-bezier equivalents; explicit [x1,y1,x2,y2]
// curves are solved with Newton-Raphson + a bisection fallback (the WebKit
// UnitBezier method). Pure; no dependencies.

import type { Easing } from './types/animation.js';

const PRESETS: Record<string, [number, number, number, number]> = {
	easeIn: [0.42, 0, 1, 1],
	easeOut: [0, 0, 0.58, 1],
	easeInOut: [0.42, 0, 0.58, 1]
};

/** Eased progress at linear progress `p` for the given curve. */
export function ease(easing: Easing, p: number): number {
	if (easing === 'linear') return p;
	const c = Array.isArray(easing) ? easing : PRESETS[easing];
	if (!c) return p;
	return unitBezier(c[0], c[1], c[2], c[3], p);
}

function unitBezier(x1: number, y1: number, x2: number, y2: number, x: number): number {
	if (x <= 0) return 0;
	if (x >= 1) return 1;

	const cx = 3 * x1;
	const bx = 3 * (x2 - x1) - cx;
	const ax = 1 - cx - bx;
	const cy = 3 * y1;
	const by = 3 * (y2 - y1) - cy;
	const ay = 1 - cy - by;

	const sampleX = (t: number) => ((ax * t + bx) * t + cx) * t;
	const sampleY = (t: number) => ((ay * t + by) * t + cy) * t;
	const sampleDX = (t: number) => (3 * ax * t + 2 * bx) * t + cx;

	// Newton-Raphson: solve sampleX(t) = x.
	let t = x;
	for (let i = 0; i < 8; i++) {
		const xErr = sampleX(t) - x;
		if (Math.abs(xErr) < 1e-6) return sampleY(t);
		const d = sampleDX(t);
		if (Math.abs(d) < 1e-6) break;
		t -= xErr / d;
	}

	// Bisection fallback.
	let lo = 0;
	let hi = 1;
	t = x;
	while (lo < hi) {
		const xVal = sampleX(t);
		if (Math.abs(xVal - x) < 1e-6) break;
		if (x > xVal) lo = t;
		else hi = t;
		t = (lo + hi) / 2;
	}
	return sampleY(t);
}
