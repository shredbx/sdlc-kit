// @vitest-environment jsdom
//
// ProgressRing primitive contract (TC7 / SC4, SC5). Mounts the real component
// and asserts the OBSERVABLE contract: an accessible progressbar whose
// aria-valuenow reflects value*100, clamped to [0,100], with non-finite input
// treated as 0. Three states (empty / progress / complete) are asserted via the
// aria-label suffix (state is never colour-only — WCAG 1.4.1) and the rendered
// SVG shape (disc+check for complete, arc for progress).

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mount, unmount, flushSync } from 'svelte';
import ProgressRing from './ProgressRing.svelte';

let target: HTMLDivElement;
let component: ReturnType<typeof mount> | null = null;

function mountWith(value: number): void {
	component = mount(ProgressRing, { target, props: { value } });
	flushSync();
}
function bar(): HTMLElement {
	return target.querySelector('[role="progressbar"]') as HTMLElement;
}
function now(): string | null {
	return bar()?.getAttribute('aria-valuenow');
}

beforeEach(() => {
	target = document.createElement('div');
	document.body.appendChild(target);
});

afterEach(() => {
	if (component) {
		unmount(component);
		component = null;
	}
	target.remove();
});

describe('ProgressRing', () => {
	it('renders an accessible progressbar with min/max 0..100', () => {
		mountWith(0.6);
		expect(bar()).toBeTruthy();
		expect(bar().getAttribute('aria-valuemin')).toBe('0');
		expect(bar().getAttribute('aria-valuemax')).toBe('100');
	});

	it('maps value 0..1 to aria-valuenow 0..100', () => {
		mountWith(0.6);
		expect(now()).toBe('60');
	});

	it('shows 0 for an empty section', () => {
		mountWith(0);
		expect(now()).toBe('0');
	});

	it('clamps a value above 1 to 100', () => {
		mountWith(1.5);
		expect(now()).toBe('100');
	});

	it('clamps a value below 0 to 0', () => {
		mountWith(-0.2);
		expect(now()).toBe('0');
	});

	it('treats non-finite values as 0', () => {
		mountWith(NaN);
		expect(now()).toBe('0');
	});

	// ── Three-state contract (design ref Image #10) ──────────────────────────────
	function label(): string | null {
		return bar()?.getAttribute('aria-label');
	}

	it('complete (value >= 1): aria-label says "complete" + renders a filled disc & check', () => {
		mountWith(1);
		expect(label()).toMatch(/: complete$/);
		expect(target.querySelector('.progress-ring__disc')).toBeTruthy();
		expect(target.querySelector('.progress-ring__check')).toBeTruthy();
		expect(target.querySelector('.progress-ring__arc')).toBeNull();
	});

	it('in-progress (0<value<1): aria-label includes the percent + renders an arc, no disc', () => {
		mountWith(0.6);
		expect(label()).toMatch(/: \d+% complete$/);
		expect(target.querySelector('.progress-ring__arc')).toBeTruthy();
		expect(target.querySelector('.progress-ring__disc')).toBeNull();
	});

	it('empty (value 0): aria-label says "not started" + renders no arc/disc', () => {
		mountWith(0);
		expect(label()).toMatch(/: not started$/);
		expect(target.querySelector('.progress-ring__arc')).toBeNull();
		expect(target.querySelector('.progress-ring__disc')).toBeNull();
	});

	it('non-finite is treated as the empty state (NaN → not started)', () => {
		mountWith(NaN);
		expect(label()).toMatch(/: not started$/);
	});
});
