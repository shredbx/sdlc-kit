/**
 * Internal sink — the cookieless first-party visitor-activity beacon (2606-002).
 *
 * Adapts a dispatcher event onto the existing `track()` util (POST /api/track).
 * It fires REGARDLESS of consent: the beacon is anonymous and first-party (the
 * server derives all PII), exactly as the inline page-view call did before this
 * abstraction — so consent gating is a Firebase/GA4 concern, never this one.
 *
 * An event with no `internalType`/`internalTarget` on its route is not internally
 * tracked → the sink no-ops for it (the route may still fan out to other sinks).
 */

import { track, type TrackEventType, type TrackTargetType } from '../track';
import type { Sink, RouteDescriptor, AnalyticsCtx } from '../types';

/** The cookieless internal beacon sink (`id: 'internal'`). */
export function internalSink(): Sink {
	return {
		id: 'internal',
		send(
			event: string,
			route: RouteDescriptor,
			params: Record<string, unknown> | undefined,
			_ctx: AnalyticsCtx
		): void {
			// Not internally tracked (e.g. a firebase-only event) → nothing to beacon.
			if (!route.internalType || !route.internalTarget) return;
			// Lift the addressable entity id out of params so the server keys the entity
			// correctly (the visitor-activity insights aggregate on target_id). The id
			// rides as `<target>_id` by convention (property_view → property_id,
			// property_video_play → property_id) or an explicit `target_id`; whichever is
			// present wins, and the remaining params become the props payload. Events whose
			// target is `page` (search, contact_click, cta_click, …) carry no entity id, so
			// target_id stays undefined and all params flow to props — exactly as before.
			const idKey = `${route.internalTarget}_id`;
			const { [idKey]: entityId, target_id, ...rest } = params ?? {};
			// `||` (not `??`) so an empty-string id coalesces to the fallback / undefined —
			// an absent entity id omits target_id rather than posting "".
			const id = entityId || target_id;
			track({
				type: route.internalType as TrackEventType,
				target: route.internalTarget as TrackTargetType,
				target_id: typeof id === 'string' ? id : undefined,
				props: Object.keys(rest).length ? rest : undefined
			});
		}
	};
}
