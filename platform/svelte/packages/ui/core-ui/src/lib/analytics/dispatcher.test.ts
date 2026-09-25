import { describe, it, expect, vi, beforeEach } from 'vitest';

// Spy on the shared beacon so the internal sink's `track()` call is observable
// without any network. The factory must define the spy inline (vi.mock is hoisted
// above imports), then we re-import it below for assertions.
vi.mock('./track', () => ({
	track: vi.fn()
}));

import { track } from './track';
import { createAnalytics } from './dispatcher';
import { internalSink } from './sinks/internal';
import type { RouteDescriptor, Sink } from './types';

const trackMock = vi.mocked(track);

// A minimal three-route registry mirroring the generated EVENT_ROUTES shape:
//  · page_view  → internal + a probe sink, internally tracked as page_view/page
//  · firebase_only → ONLY a non-internal sink (proves the internal sink no-ops AND
//                    that an event never reaches a sink it doesn't list)
type Ev = 'page_view' | 'firebase_only' | 'property_view';
const routes: Record<Ev, RouteDescriptor> = {
	page_view: { sinks: ['internal', 'probe'], internalType: 'page_view', internalTarget: 'page' },
	firebase_only: { sinks: ['probe'] },
	property_view: { sinks: ['internal'], internalType: 'property_view', internalTarget: 'property' }
};

describe('createAnalytics dispatcher', () => {
	beforeEach(() => {
		trackMock.mockClear();
	});

	it('routes an event ONLY to the sinks listed in route.sinks', () => {
		const seen: string[] = [];
		const probe: Sink = { id: 'probe', send: () => seen.push('probe') };
		const a = createAnalytics<Ev>({
			routes,
			sinks: [internalSink(), probe],
			consent: () => false
		});

		// page_view lists both → internal (track) + probe fire.
		a.emit('page_view');
		expect(trackMock).toHaveBeenCalledTimes(1);
		expect(seen).toEqual(['probe']);

		// firebase_only lists only 'probe' → the internal sink must NOT be invoked.
		a.emit('firebase_only');
		expect(trackMock).toHaveBeenCalledTimes(1); // unchanged
		expect(seen).toEqual(['probe', 'probe']);
	});

	it('fires the internal sink REGARDLESS of consent (cookieless first-party)', () => {
		const denied = createAnalytics<Ev>({
			routes,
			sinks: [internalSink()],
			consent: () => false
		});
		denied.emit('page_view');
		expect(trackMock).toHaveBeenCalledTimes(1);

		trackMock.mockClear();

		const granted = createAnalytics<Ev>({
			routes,
			sinks: [internalSink()],
			consent: () => true
		});
		granted.emit('page_view');
		expect(trackMock).toHaveBeenCalledTimes(1);
	});

	it('emit(page_view) calls track with exactly {type:page_view, target:page}', () => {
		const a = createAnalytics<Ev>({
			routes,
			sinks: [internalSink()],
			consent: () => false
		});
		a.emit('page_view');
		expect(trackMock).toHaveBeenCalledTimes(1);
		expect(trackMock).toHaveBeenCalledWith({
			type: 'page_view',
			target: 'page',
			target_id: undefined,
			props: undefined
		});
	});

	it('lifts the <target>_id param to target_id (entity insights) and routes the rest to props', () => {
		const a = createAnalytics<Ev>({ routes, sinks: [internalSink()], consent: () => false });
		// property_view (internalTarget 'property') → property_id rides as target_id,
		// NOT in props — this is the contract the visitor-activity aggregation keys on.
		a.emit('property_view', { property_id: 'p1' });
		expect(trackMock).toHaveBeenLastCalledWith({
			type: 'property_view',
			target: 'property',
			target_id: 'p1',
			props: undefined
		});
		// A coexisting non-id param still flows to props while the id stays lifted.
		a.emit('property_view', { property_id: 'p2', foo: 'bar' });
		expect(trackMock).toHaveBeenLastCalledWith({
			type: 'property_view',
			target: 'property',
			target_id: 'p2',
			props: { foo: 'bar' }
		});
	});

	it('silently ignores an unknown event (no route → no throw, no sink call)', () => {
		const seen: string[] = [];
		const probe: Sink = { id: 'probe', send: () => seen.push('probe') };
		const a = createAnalytics<Ev>({ routes, sinks: [internalSink(), probe], consent: () => false });
		// @ts-expect-error — exercising a stale/unknown event name at runtime.
		a.emit('does_not_exist');
		expect(trackMock).not.toHaveBeenCalled();
		expect(seen).toEqual([]);
	});

	it('setConsent forwards to sinks that opt in, skips those without the hook', () => {
		const flips: boolean[] = [];
		const fb: Sink = { id: 'firebase', send: () => {}, setConsent: (g) => flips.push(g) };
		// internalSink() has NO setConsent → must be skipped without throwing.
		const a = createAnalytics<Ev>({ routes, sinks: [internalSink(), fb], consent: () => false });
		a.setConsent(true);
		expect(flips).toEqual([true]);
	});
});
