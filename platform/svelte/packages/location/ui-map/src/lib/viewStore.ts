// localStorage-backed defaults for LocationPicker / PolygonPicker viewports.
//
// When a property has no map_view of its own (typical for /new and freshly
// opened drafts), the picker reads the agent's last-used view from
// localStorage so they're not dropped on the equator every time. Consumers
// persist the view on the property; this store captures the *workspace*
// default that survives across properties.
//
// Keys are caller-namespaced (e.g. `br.locationPicker.lastView`) — there's
// no global default. Use one key per picker family per project so the
// edit page's view doesn't contaminate the polygon picker's defaults.
//
// SSR-safe: every operation guards `typeof window`. Returns undefined on
// the server, treats malformed JSON as absent, never throws.

import type { MapViewState } from './types';

/**
 * Read the last-saved view for a key, or undefined if nothing's stored
 * (or on the server). Malformed JSON is treated as absent — never throws.
 */
export function getDefaultView(key: string): MapViewState | undefined {
	if (typeof window === 'undefined') return undefined;
	try {
		const raw = window.localStorage.getItem(key);
		if (!raw) return undefined;
		const parsed = JSON.parse(raw) as MapViewState;
		// Minimal shape check — zoom + center are the load-bearing fields.
		// Reject anything that wouldn't be useful as an initial camera.
		if (typeof parsed !== 'object' || parsed === null) return undefined;
		if (parsed.zoom !== undefined && typeof parsed.zoom !== 'number') return undefined;
		if (parsed.center !== undefined && (!Array.isArray(parsed.center) || parsed.center.length !== 2)) {
			return undefined;
		}
		return parsed;
	} catch {
		return undefined;
	}
}

// Module-level debounce state. Keyed by storage-key so independent pickers
// don't share a timer. Latest call in any 500ms window wins.
const writeQueue = new Map<string, { view: MapViewState; timer: ReturnType<typeof setTimeout> }>();
const DEBOUNCE_MS = 500;

/**
 * Persist a view as the new default for `key`. Debounced to at most 1 write
 * per 500ms per key — high-frequency idle events from smooth pans no longer
 * thrash localStorage. The latest call in any 500ms window wins. No-op on
 * the server or when localStorage is unavailable.
 */
export function setDefaultView(key: string, view: MapViewState): void {
	if (typeof window === 'undefined') return;

	const existing = writeQueue.get(key);
	if (existing) {
		// Coalesce — replace the pending value, let the existing timer fire.
		existing.view = view;
		return;
	}

	const entry: { view: MapViewState; timer: ReturnType<typeof setTimeout> } = {
		view,
		timer: setTimeout(() => {
			const final = writeQueue.get(key);
			writeQueue.delete(key);
			if (!final) return;
			try {
				window.localStorage.setItem(key, JSON.stringify(final.view));
			} catch {
				// QuotaExceededError, SecurityError, etc. — same fail-open contract.
			}
		}, DEBOUNCE_MS)
	};
	writeQueue.set(key, entry);
}

/**
 * Remove the stored default for a key. Useful for "Reset to global default"
 * affordances, and for tests.
 */
export function clearDefaultView(key: string): void {
	if (typeof window === 'undefined') return;
	try {
		window.localStorage.removeItem(key);
	} catch {
		// Same fail-open contract as setDefaultView.
	}
}
