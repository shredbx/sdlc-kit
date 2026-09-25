// Shared Google Maps SDK loader — single source of truth so multiple
// pickers on the same page (LocationPicker + PolygonPicker) don't append
// duplicate <script> tags. Two-loader regression symptom: console warns
// "Element with name X already defined" + the second picker stalls on
// "Loading map…" because Maps' internal CustomElementRegistry refuses
// to re-register its web components.
//
// We request the union of libraries every consumer might need:
//   - places — used by GeocodeSearch dropdown (LocationPicker)
// `drawing` was dropped: google.maps.drawing.DrawingManager is no longer
// shipped on the `weekly` channel and PolygonPicker now uses a custom
// native draw handler (map click → Polyline + vertex markers → close ring),
// so no consumer loads the drawing library anymore.

import { MapConfigError } from './types.js';

const CALLBACK_NAME = '__sbx_ui_map_google_ready__';
const LIBRARIES = 'places';

// Minimal namespace shape — each consumer declares its own narrower
// view of what it actually touches. The shared loader doesn't care
// about the full surface, it just returns whatever's on window.google.
type GoogleMapsNamespaceLike = unknown;

let loadPromise: Promise<GoogleMapsNamespaceLike> | null = null;

export function loadGoogleMaps(apiKey: string): Promise<GoogleMapsNamespaceLike> {
	if (typeof window === 'undefined') {
		return Promise.reject(new MapConfigError('Google Maps cannot load during SSR'));
	}
	if (loadPromise) return loadPromise;
	const existing = (window as unknown as { google?: { maps?: unknown } }).google?.maps;
	if (existing) {
		loadPromise = Promise.resolve(existing);
		return loadPromise;
	}

	loadPromise = new Promise((resolve, reject) => {
		const w = window as unknown as Record<string, unknown>;
		w[CALLBACK_NAME] = () => {
			delete w[CALLBACK_NAME];
			const ns = (window as unknown as { google?: { maps?: unknown } }).google?.maps;
			if (ns) resolve(ns);
			else reject(new MapConfigError('Google Maps loaded but namespace missing'));
		};

		const script = document.createElement('script');
		const params = new URLSearchParams({
			key: apiKey,
			libraries: LIBRARIES,
			callback: CALLBACK_NAME,
			loading: 'async',
			v: 'weekly'
		});
		script.src = `https://maps.googleapis.com/maps/api/js?${params.toString()}`;
		script.async = true;
		script.defer = true;
		script.onerror = () => {
			delete (window as unknown as Record<string, unknown>)[CALLBACK_NAME];
			loadPromise = null;
			reject(new MapConfigError('Google Maps script failed to load (network or referrer restriction)'));
		};
		document.head.appendChild(script);
	});

	return loadPromise;
}
