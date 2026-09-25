/**
 * Analytics dispatcher — the one-plane `emit()` fan-out (2606-002).
 *
 * `createAnalytics` binds a project's generated event registry, its sinks, and a
 * consent reader into a single instance with `emit` + `setConsent`. `emit` is the
 * ONLY routing logic in the system: it resolves consent once, then forwards the
 * event to every sink whose id the route lists — no per-event if/else anywhere.
 * Each sink decides what to do with the event and the consent snapshot.
 */

import type { RouteDescriptor, Sink, AnalyticsCtx } from './types';

/** A bound analytics instance for one project (one event taxonomy + sink set). */
export interface Analytics<E extends string> {
	/**
	 * Emit one event. Resolves consent ONCE (the single plane), looks up the route,
	 * and dispatches to each listed sink. Unknown events (no route) are silently
	 * ignored so a stale call site can never throw.
	 */
	emit(event: E, params?: Record<string, unknown>): void;
	/**
	 * Propagate a consent change to every sink that opts into live updates (the
	 * Firebase sink in D1-C); sinks without `setConsent` are skipped.
	 */
	setConsent(granted: boolean): void;
}

/**
 * Compose the dispatcher from a project's routes, sinks, and consent reader.
 *
 * - `routes`: the generated `Record<E, RouteDescriptor>` event registry.
 * - `sinks`: the destinations this project wires (internal now; firebase in D1-C).
 * - `consent`: a thunk read once per `emit` — the single consent source of truth.
 */
export function createAnalytics<E extends string>(config: {
	routes: Record<E, RouteDescriptor>;
	sinks: Sink[];
	consent: () => boolean;
}): Analytics<E> {
	return {
		emit(event, params) {
			// Resolve consent ONCE here — every sink for this event sees the same
			// snapshot. This is the single consent plane; sinks never re-read it.
			const ctx: AnalyticsCtx = { consentGranted: config.consent() };
			const route = config.routes[event];
			if (!route) return;
			// The ONLY routing logic: id match → send. No per-event branching.
			for (const sink of config.sinks) {
				if (route.sinks.includes(sink.id)) sink.send(event, route, params, ctx);
			}
		},
		setConsent(granted) {
			// Forward live consent changes to sinks that care (e.g. firebase/gtag in
			// D1-C); the internal beacon omits setConsent and is skipped.
			for (const sink of config.sinks) sink.setConsent?.(granted);
		}
	};
}
