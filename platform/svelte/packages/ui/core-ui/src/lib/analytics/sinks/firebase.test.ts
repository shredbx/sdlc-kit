// @vitest-environment jsdom
//
// Firebase/GA4 sink — the HARD-GATE proof (2606-002, D1-C). jsdom gives us a real
// `window`/`document` so we can assert deterministically — WITHOUT a browser or any
// network — that nothing gtag-related exists until consent is granted, and that the
// script injects exactly once when it is. This is the legal-posture acceptance test:
// a denied visitor never touches googletagmanager.com.

import { describe, it, expect, beforeEach } from 'vitest';
import { firebaseSink } from './firebase';
import type { RouteDescriptor, AnalyticsCtx } from '../types';

const ID = 'G-TEST1234';
const route: RouteDescriptor = { sinks: ['firebase'] };
const granted: AnalyticsCtx = { consentGranted: true };
const denied: AnalyticsCtx = { consentGranted: false };

/** The injected GA script, or null if none has been added. */
function gaScript(): Element | null {
	return document.querySelector('script[src*="googletagmanager"]');
}

describe('firebaseSink — hard-gated GA4 channel', () => {
	beforeEach(() => {
		// Fresh page between tests: no injected script, no stub. Each test builds its
		// own sink (the `loaded` latch is module-local PER sink instance via closure).
		document.head.innerHTML = '';
		delete (window as unknown as { gtag?: unknown }).gtag;
		delete (window as unknown as { dataLayer?: unknown }).dataLayer;
	});

	it('HARD GATE: send() with consent denied loads NOTHING (no script, no gtag, no push)', () => {
		const sink = firebaseSink({ measurementId: ID });
		sink.send('page_view', route, { a: 1 }, denied);

		// THE proof: zero third-party script, no stub defined, no queue.
		expect(gaScript()).toBeNull();
		expect(window.gtag).toBeUndefined();
		expect(window.dataLayer).toBeUndefined();
	});

	it('send() with consent granted injects the GA script and pushes the event', () => {
		const sink = firebaseSink({ measurementId: ID });
		sink.send('page_view', route, { page: '/' }, granted);

		const s = gaScript();
		expect(s).not.toBeNull();
		expect(s!.getAttribute('src')).toBe(`https://www.googletagmanager.com/gtag/js?id=${ID}`);
		expect(typeof window.gtag).toBe('function');
		// The dataLayer is the gtag queue (the stub pushes raw arguments objects).
		// Assert an ('event','page_view',…) tuple landed. arguments objects are
		// array-like, so spread each into a real array before matching.
		const pushed = (window.dataLayer ?? []).map((a) => Array.from(a as ArrayLike<unknown>));
		expect(pushed).toContainEqual(['event', 'page_view', { page: '/' }]);
	});

	it('queue entries are REAL Arguments objects — gtag.js silently ignores plain arrays', () => {
		// Regression guard for the 2026-07-12 silent-dead incident: a rest-parameter
		// stub (`(...args) => dataLayer.push(args)`) fills the queue with Arrays, which
		// gtag.js's processor skips — the script loads, the dataLayer looks full, and
		// zero hits reach GA. Only `[object Arguments]` entries execute as commands.
		// (The Array.from-based assertions elsewhere in this file accept both shapes,
		// which is exactly how the drift shipped.)
		const sink = firebaseSink({ measurementId: ID });
		sink.send('page_view', route, { page: '/' }, granted);

		const queue = window.dataLayer ?? [];
		expect(queue.length).toBeGreaterThan(0);
		for (const entry of queue) {
			expect(Object.prototype.toString.call(entry)).toBe('[object Arguments]');
		}
	});

	it("guards an empty measurementId: consent granted still loads NOTHING", () => {
		const sink = firebaseSink({ measurementId: '' });
		sink.send('page_view', route, {}, granted);

		expect(gaScript()).toBeNull();
		expect(window.gtag).toBeUndefined();
	});

	it('setConsent(true) loads gtag; a later setConsent(false) pushes a denied update', () => {
		const sink = firebaseSink({ measurementId: ID });

		sink.setConsent(true);
		expect(gaScript()).not.toBeNull();
		expect(typeof window.gtag).toBe('function');

		sink.setConsent(false);
		const pushed = (window.dataLayer ?? []).map((a) => Array.from(a as ArrayLike<unknown>));
		expect(pushed).toContainEqual(['consent', 'update', { analytics_storage: 'denied' }]);
	});

	it('setConsent(false) before any load is inert (no stub, no script)', () => {
		const sink = firebaseSink({ measurementId: ID });
		sink.setConsent(false);

		expect(gaScript()).toBeNull();
		expect(window.gtag).toBeUndefined();
	});

	it('idempotent: two granted sends inject the GA script only ONCE', () => {
		const sink = firebaseSink({ measurementId: ID });
		sink.send('page_view', route, {}, granted);
		sink.send('view_property', route, { id: '1' }, granted);

		expect(document.querySelectorAll('script[src*="googletagmanager"]').length).toBe(1);
	});

	it('BUFFER: an event fired before consent is decided is held (nothing sent), then FLUSHED on grant', () => {
		const sink = firebaseSink({ measurementId: ID });
		// Landing-load page_view fires (onMount) before the consent effect resolves.
		sink.send('page_view', route, { page: '/' }, denied);
		// Hard-gate intact while buffered: no script, no stub, no network.
		expect(gaScript()).toBeNull();
		expect(window.gtag).toBeUndefined();

		// Consent resolves granted (the layout effect) → gtag loads and the buffered
		// event replays — so an already-consented visitor's landing view reaches GA4.
		sink.setConsent(true);
		expect(gaScript()).not.toBeNull();
		const pushed = (window.dataLayer ?? []).map((a) => Array.from(a as ArrayLike<unknown>));
		expect(pushed).toContainEqual(['event', 'page_view', { page: '/' }]);
	});

	it('BUFFER: events buffered before a DENY are dropped — never sent, even after a later grant', () => {
		const sink = firebaseSink({ measurementId: ID });
		sink.send('page_view', route, { page: '/' }, denied); // buffered
		sink.setConsent(false); // resolve denied → drop buffer
		expect(gaScript()).toBeNull();

		sink.setConsent(true); // a later grant loads gtag…
		const pushed = (window.dataLayer ?? []).map((a) => Array.from(a as ArrayLike<unknown>));
		// …but the dropped page_view is NOT replayed (it was emitted pre-consent and denied).
		expect(pushed.some((p) => p[0] === 'event' && p[1] === 'page_view')).toBe(false);
	});
});
