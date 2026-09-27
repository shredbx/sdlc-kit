/**
 * Visitor-activity beacon — the shared `track()` util.
 *
 * Posts a single anonymous activity event to the project's `/api/track`
 * ingestion endpoint (the workspace convention). It is reused as-is by every
 * project (BR now, BS later) — the SERVER derives all PII-sensitive fields
 * (visitor id, device, is_bot, is_staff, referer origin, ip); the client only
 * declares the event semantics + the current path/referer.
 *
 * LANDMINE (see docs/plans/2026-05-27-visitor-activity-PART-B-handoff.md §B-d):
 * the whole `/api/*` group sits behind a CSRF gate that 403s any write missing
 * the `X-Requested-With: XMLHttpRequest` header. `navigator.sendBeacon` CANNOT
 * set that header, so it is BANNED here — we use `fetch(..., {keepalive:true})`
 * which both sets the header AND survives an unload/navigation.
 *
 * Analytics must NEVER break the page: every failure path is swallowed.
 */

/**
 * The closed event taxonomy the beacon accepts. These unions MIRROR the Go
 * `EventType` / `TargetType` named types in `pkg/visitoractivity` and the DB
 * CHECK constraints — three aligned sources of truth. Typing them (rather than
 * bare `string`) means a mistyped name fails `svelte-check` at build time
 * instead of silently 422-ing at runtime. Add a new event/target here AND in the
 * Go enum AND in the DB CHECK, in lockstep.
 */
export type TrackEventType = 'property_view' | 'listing_view' | 'page_view' | 'click';
export type TrackTargetType = 'property' | 'listing' | 'page';

/** A single activity event the client can declare. */
export interface TrackEvent {
	/** Event semantics — must be a known {@link TrackEventType}. */
	type: TrackEventType;
	/** What the event is about — must be a known {@link TrackTargetType}. */
	target: TrackTargetType;
	/** Required when `target` is an addressable entity (e.g. a property UUID). */
	target_id?: string;
	/** Per-project extension payload (the server caps it at ~2 KB marshaled). */
	props?: Record<string, unknown>;
}

/**
 * The ONLY query params the beacon will ever harvest from the landing URL. Five
 * standard UTM keys — the campaign-attribution allowlist. The full query string
 * is NEVER forwarded: only these named keys, only when present, are lifted into
 * `props.utm`. This keeps the beacon privacy-safe (no arbitrary URL data leaks
 * into the props JSONB) and matches the server-side `props->>'utm_*'` reads.
 */
const UTM_KEYS = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_term', 'utm_content'] as const;

/**
 * Extract the UTM allowlist from a query string into a plain object, or
 * `undefined` when no UTM key is present (so we never attach an empty `utm`).
 * Only the five {@link UTM_KEYS} are read; everything else in the query string
 * is ignored. Values are taken verbatim (URLSearchParams already decodes them).
 */
export function extractUtm(search: string): Record<string, string> | undefined {
	const params = new URLSearchParams(search);
	const utm: Record<string, string> = {};
	for (const key of UTM_KEYS) {
		const value = params.get(key);
		if (value) utm[key] = value;
	}
	return Object.keys(utm).length > 0 ? utm : undefined;
}

/**
 * Fire one activity event. Returns immediately (fire-and-forget). The request
 * uses `keepalive` so it completes even if the page is navigating away, and
 * `credentials: 'same-origin'` so the auth cookie rides along (the server uses
 * it only to set `is_staff`, never to identify the visitor).
 */
export function track(event: TrackEvent): void {
	// Guard SSR / non-browser environments — there is no document/location there.
	if (typeof fetch === 'undefined' || typeof location === 'undefined') return;
	try {
		// Campaign attribution — fold the UTM allowlist from the landing URL into
		// props.utm. Only attach `utm` when at least one allowlisted key is present
		// (never an empty object). The full query string is never stored — only the
		// five allowlisted keys. An explicit caller-supplied props.utm wins.
		const utm = extractUtm(location.search);
		const props =
			utm && !event.props?.utm ? { ...event.props, utm } : event.props;
		const body = JSON.stringify({
			...event,
			...(props ? { props } : {}),
			path: location.pathname,
			referer: (typeof document !== 'undefined' && document.referrer) || undefined
		});
		void fetch('/api/track', {
			method: 'POST',
			keepalive: true,
			credentials: 'same-origin',
			headers: {
				'Content-Type': 'application/json',
				// Satisfies the global CSRF gate; sendBeacon can't set this → banned.
				'X-Requested-With': 'XMLHttpRequest'
			},
			body
		}).catch(() => {
			/* analytics must never surface to the page */
		});
	} catch {
		/* analytics must never break the page (e.g. JSON cycle, blocked fetch) */
	}
}
