import type {
	GeocodeQuery,
	GeocodeResult,
	MapAdapter,
	MapHandle,
	MountOpts,
	ReverseGeocodeQuery
} from '../types.js';
import { MapNotImplementedError } from '../types.js';

// =============================================================================
// Mapbox adapter — DEFERRED to Phase 4 of the location-picker handoff
// =============================================================================
// Per Decision #0269 the public-facing <Map> primitive will use Mapbox tiles
// via Leaflet 1.9.x raster. That work is out of scope for task 2605-123
// (admin LocationPicker only — no public display). When Phase 4 begins:
//
//   1. Add `leaflet` + `@types/leaflet` to dependencies
//   2. Implement mount() using onMount + dynamic import of leaflet (SSR-safe).
//      The returned MapHandle MUST implement fitBounds(box, padding?) for
//      contract parity with the Google adapter (admin-region framing) —
//      Leaflet's L.Map.fitBounds([[s,w],[n,e]], { padding }) maps directly.
//   3. Implement geocode() using Mapbox Geocoding API v5 (forward geocoding,
//      country=th bias, ≤5 results). Populate GeocodeResult.bounds from the
//      feature's `bbox` ([w,s,e,n]) so empty-state admin framing works on the
//      public surface too.
//
// Reference pattern:
//   clients/andrei/projects/land-canvas/apps/web/svelte/src/lib/components/
//   MapBackground.svelte (362 lines, proven SSR-gated Leaflet integration)
// =============================================================================

export function createMapboxAdapter(_accessToken: string): MapAdapter {
	return {
		async mount(_opts: MountOpts): Promise<MapHandle> {
			throw new MapNotImplementedError('mapbox');
		},
		async search(_query: GeocodeQuery): Promise<GeocodeResult[]> {
			throw new MapNotImplementedError('mapbox');
		},
		async reverseGeocode(_query: ReverseGeocodeQuery): Promise<GeocodeResult | null> {
			throw new MapNotImplementedError('mapbox');
		},
		destroy(_handle: MapHandle): void {
			// no-op
		}
	};
}
