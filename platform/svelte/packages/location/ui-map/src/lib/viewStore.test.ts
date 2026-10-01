import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { setDefaultView } from './viewStore.js';

// Minimal localStorage polyfill for the test — vitest's default node env has no localStorage.
beforeEach(() => {
	const store = new Map<string, string>();
	(globalThis as { localStorage?: Storage }).localStorage = {
		getItem: (k: string) => store.get(k) ?? null,
		setItem: (k: string, v: string) => { store.set(k, v); },
		removeItem: (k: string) => { store.delete(k); },
		clear: () => store.clear(),
		get length() { return store.size; },
		key: (i: number) => Array.from(store.keys())[i] ?? null
	} as Storage;
	(globalThis as { window?: { localStorage: Storage } }).window =
		(globalThis as { window?: { localStorage: Storage } }).window ??
		({ localStorage: globalThis.localStorage } as { localStorage: Storage });
	vi.useFakeTimers();
});

afterEach(() => {
	vi.useRealTimers();
});

describe('setDefaultView — throttling', () => {
	it('coalesces 10 rapid calls into at most 1 localStorage write per 500ms window', () => {
		const spy = vi.spyOn(globalThis.localStorage, 'setItem');
		for (let i = 0; i < 10; i++) {
			setDefaultView('test-key', { zoom: 11 + i, center: [100, 9.7] });
		}
		// Within the 500ms window: still pending — at most one write so far.
		expect(spy.mock.calls.length).toBeLessThanOrEqual(1);
		// After advancing the timer, the latest value lands.
		vi.advanceTimersByTime(500);
		expect(spy).toHaveBeenCalled();
		const last = spy.mock.calls[spy.mock.calls.length - 1];
		expect(last[0]).toBe('test-key');
		expect(JSON.parse(last[1] as string)).toEqual({ zoom: 20, center: [100, 9.7] });
	});
});
