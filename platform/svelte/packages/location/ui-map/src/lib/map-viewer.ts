// MapViewer adapter factory (#0304) — resolve the read-only fullscreen map provider
// for a SURFACE by its DOMAIN, not the user's role: public pages always get the public
// (MapLibre terrain) map — even for an admin visitor — while the Google map is reserved
// for admin pages (it carries the map-type/terrain switcher + is license-OK for
// authoring). New providers are added HERE only; consumers just render <MapViewer
// domain=… /> and never name a provider.
import type { MapViewerAdapter, MapViewerDomain } from './types.js';
import { createMaplibreViewerAdapter } from './adapters/maplibre-viewer.js';
import { createGoogleViewerAdapter } from './adapters/google-viewer.js';

export function createMapViewerAdapter(domain: MapViewerDomain): MapViewerAdapter {
	return domain === 'admin' ? createGoogleViewerAdapter() : createMaplibreViewerAdapter();
}
