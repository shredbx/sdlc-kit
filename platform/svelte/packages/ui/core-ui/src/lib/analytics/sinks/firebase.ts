/**
 * Firebase / GA4 sink — the HARD-GATED gtag channel (2606-002, D1-C).
 *
 * The legal posture is the whole point: NOTHING about gtag (no stub, no script,
 * no Consent-Mode default block) exists until analytics consent is granted. There
 * is no gtag stub in app.html and no default-denied block anywhere — THIS sink
 * defines `window.dataLayer` + `window.gtag` ITSELF, the first time consent flips
 * true, immediately before it injects the GA script. So zero third-party
 * (`googletagmanager.com`) bytes load until `ctx.consentGranted === true` (or an
 * explicit `setConsent(true)`); a denied visitor never touches Google.
 *
 * SvelteKit-agnostic by design: the `measurementId` is INJECTED by the consumer
 * (BR passes `env.PUBLIC_FIREBASE_MEASUREMENT_ID`), so this file imports no `$env`
 * and stays reusable by any project. An empty id makes the sink self-no-op (dev
 * without the value still runs).
 */

import type { Sink, RouteDescriptor, AnalyticsCtx } from '../types';

declare global {
	interface Window {
		gtag?: (...args: unknown[]) => void;
		dataLayer?: unknown[];
	}
}

/**
 * The Firebase/GA4 sink (`id: 'firebase'`). `measurementId` is the GA4 id
 * (`G-XXXX`); an empty string makes every method a no-op (the dev/no-id path).
 */
export function firebaseSink({ measurementId }: { measurementId: string }): Sink {
	// Module-local idempotency latch: the gtag stub + script are installed at most
	// once per page load, the first time consent is granted (send or setConsent).
	let loaded = false;

	// Consent-resolution buffer. Events fired BEFORE consent is decided (e.g. the
	// landing page's page_view / property_view, which fire in onMount before the
	// layout effect has hydrated consent and called setConsent) are held in memory
	// — NEVER sent — until consent resolves: flushed on grant, dropped on deny. This
	// captures the landing-load events for already-consented visitors WITHOUT ever
	// touching Google before consent (the hard-gate holds: no gtag, no network, no
	// cookie while pending). `resolved` flips on the first definitive consent signal.
	let resolved = false;
	const pending: Array<{ event: string; params?: Record<string, unknown> }> = [];
	const MAX_PENDING = 50;

	/** Replay every buffered event through gtag, then clear the buffer. */
	function flush(): void {
		for (const p of pending) window.gtag?.('event', p.event, p.params ?? {});
		pending.length = 0;
	}

	/**
	 * Install the gtag stub and inject the GA script ONCE. Guarded so it never runs
	 * without an id, server-side, or twice. The stub (dataLayer + gtag) is defined
	 * BEFORE the async script so any `gtag(...)` calls queued before the script
	 * arrives are replayed by GA when it loads.
	 */
	function ensureLoaded(): void {
		if (loaded || !measurementId || typeof window === 'undefined' || typeof document === 'undefined')
			return;
		loaded = true;
		window.dataLayer = window.dataLayer || [];
		// Canonical snippet shape — gtag.js executes ONLY `[object Arguments]` queue
		// entries; a rest-parameter array is silently skipped (script loads, dataLayer
		// fills, zero hits reach GA — the 2026-07-12 silent-dead incident).
		window.gtag = function () {
			// eslint-disable-next-line prefer-rest-params
			window.dataLayer!.push(arguments);
		};
		window.gtag('js', new Date());
		// SPA: suppress GA's automatic first page_view — every page_view is emitted
		// explicitly through the dispatcher so client-side navigations are counted.
		window.gtag('config', measurementId, { send_page_view: false });
		const s = document.createElement('script');
		s.async = true;
		s.src = `https://www.googletagmanager.com/gtag/js?id=${measurementId}`;
		document.head.appendChild(s);
	}

	return {
		id: 'firebase',
		send(
			event: string,
			_route: RouteDescriptor,
			params: Record<string, unknown> | undefined,
			ctx: AnalyticsCtx
		): void {
			// HARD GATE: a denied visitor is kept entirely off Google.
			if (ctx.consentGranted) {
				resolved = true;
				ensureLoaded();
				flush(); // drain anything buffered before this grant
				window.gtag?.('event', event, params ?? {});
				return;
			}
			// Consent not (yet) granted. Before it is DECIDED, buffer in memory (no
			// network) so a subsequent grant can replay the landing-load events; once
			// decided-and-denied, drop silently. Either way nothing reaches Google.
			if (!resolved) {
				pending.push({ event, params });
				if (pending.length > MAX_PENDING) pending.shift();
			}
		},
		setConsent(granted: boolean): void {
			// Live consent flip from the layout effect — the first definitive signal.
			resolved = true;
			if (granted) {
				ensureLoaded();
				window.gtag?.('consent', 'update', { analytics_storage: 'granted' });
				flush(); // replay events buffered before consent resolved
			} else {
				pending.length = 0; // consent denied → drop the buffer, never sent
				if (loaded) window.gtag?.('consent', 'update', { analytics_storage: 'denied' });
			}
		}
	};
}
