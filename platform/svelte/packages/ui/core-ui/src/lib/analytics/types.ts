/**
 * Analytics dispatcher contracts — the one-plane abstraction (2606-002).
 *
 * A project compiles a model-driven event registry (the generated `EVENT_ROUTES`)
 * and a set of sinks; the dispatcher reads consent ONCE per emit and fans the
 * event out to exactly the sinks the route names. These types are the seam every
 * sink (the cookieless internal beacon now; Firebase/GA4 hard-gated in D1-C)
 * implements — so adding a sink is a new file, never a change to the dispatcher.
 */

/**
 * The consent snapshot resolved ONCE per `emit` and handed to every sink for that
 * event. The single source of consent truth — sinks NEVER read consent themselves;
 * they decide what to do with `consentGranted` (the internal beacon ignores it,
 * being cookieless/first-party; the Firebase sink will hard-gate on it in D1-C).
 */
export interface AnalyticsCtx {
	/** Whether the visitor has granted analytics consent at emit time. */
	consentGranted: boolean;
}

/**
 * How one event routes — the per-event entry of the generated registry. `sinks`
 * is the allowlist of sink ids that should receive the event; `internalType` /
 * `internalTarget` map the event onto the shared visitor-activity taxonomy when
 * (and only when) the internal sink is in `sinks` and the event is internally
 * tracked. `readonly string[]` so the generated `SinkId[]` registry is assignable
 * here without array-variance complaints.
 */
export interface RouteDescriptor {
	/** Allowlist of sink ids that receive this event. */
	sinks: readonly string[];
	/** Visitor-activity `type` for the internal sink (omitted ⇒ not internally tracked). */
	internalType?: string;
	/** Visitor-activity `target` for the internal sink (omitted ⇒ not internally tracked). */
	internalTarget?: string;
}

/**
 * A destination for analytics events. Each sink owns one delivery channel and is
 * selected per-event by its `id` appearing in a route's `sinks`. The dispatcher
 * never branches per event — it only matches ids and calls `send`.
 */
export interface Sink {
	/** Stable id matched against `RouteDescriptor.sinks`. */
	id: string;
	/**
	 * Deliver one event. `params` is the caller-supplied payload (sink decides how
	 * to shape it); `ctx` carries the once-resolved consent snapshot. Fire-and-forget
	 * — a sink MUST swallow its own failures (analytics never breaks the page).
	 */
	send(
		event: string,
		route: RouteDescriptor,
		params: Record<string, unknown> | undefined,
		ctx: AnalyticsCtx
	): void;
	/**
	 * Optional consent-change hook (forward-wired for the Firebase sink in D1-C, which
	 * will flip gtag's consent state). Sinks that don't care about live consent updates
	 * (the internal beacon) simply omit it.
	 */
	setConsent?(granted: boolean): void;
}
