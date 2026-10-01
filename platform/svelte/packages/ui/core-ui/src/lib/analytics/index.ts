/**
 * Analytics — the visitor-activity beacon + the one-plane dispatcher (2606-002).
 *
 * Platform-neutral: any project posts the same event shape to its own `/api/track`
 * endpoint (the `track` beacon), and composes the `createAnalytics` dispatcher over
 * its generated event registry + sinks. Reusable by BR, BS, and any future consumer.
 */

export { track } from './track';
export type { TrackEvent, TrackEventType, TrackTargetType } from './track';

export { createAnalytics } from './dispatcher';
export type { Analytics } from './dispatcher';
export { internalSink } from './sinks/internal';
export { firebaseSink } from './sinks/firebase';
export type { Sink, RouteDescriptor, AnalyticsCtx } from './types';
