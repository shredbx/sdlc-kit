// MapBoard adapter factory (task 2607-133) — the multi-marker sibling of
// createMapViewerAdapter (#0304). Same rule: the provider is resolved from the SURFACE's
// domain, never from the visitor's role, and never named at a call site.
//
// Only the public (keyless MapLibre) board exists today — the catalog Map view and the
// homepage map section are both public surfaces. An admin board would be added HERE, exactly
// as the Google viewer was, without touching a single consumer.
import type { MapBoardAdapter, MapViewerDomain } from './types.js';
import { createMaplibreBoardAdapter } from './adapters/maplibre-board.js';

export function createMapBoardAdapter(_domain: MapViewerDomain = 'public'): MapBoardAdapter {
	return createMaplibreBoardAdapter();
}
