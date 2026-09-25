import type {
	BoundingBox,
	GeocodeQuery,
	GeocodeResult,
	LngLat,
	MapAdapter,
	MapHandle,
	MapViewport,
	MountOpts,
	ReverseGeocodeQuery,
	StructuredAddress
} from '../types.js';
import { MapConfigError } from '../types.js';
import { loadGoogleMaps as loadGoogleMapsShared } from '../google-loader.js';

// Google Maps JS SDK loader -------------------------------------------------

interface GoogleMapsNamespace {
	Map: new (
		container: HTMLElement,
		opts: {
			center: LngLat;
			zoom: number;
			disableDefaultUI?: boolean;
			mapTypeControl?: boolean;
			mapTypeId?: string;
			zoomControl?: boolean;
			zoomControlOptions?: { position: number };
			fullscreenControl?: boolean;
			gestureHandling?: string;
			keyboardShortcuts?: boolean;
			disableDoubleClickZoom?: boolean;
			clickableIcons?: boolean;
		}
	) => GoogleMap;
	Geocoder: new () => GoogleGeocoder;
	event: {
		clearInstanceListeners(instance: unknown): void;
		// Fire a named event on an instance. Used to nudge the map to re-read
		// its container size after a fullscreen toggle ('resize').
		trigger(instance: unknown, eventName: string): void;
	};
	// Positional subset of google.maps.ControlPosition — only the corners we
	// pin controls to. Runtime values come from the SDK.
	ControlPosition: { RIGHT_BOTTOM: number; RIGHT_TOP: number; LEFT_BOTTOM: number; LEFT_TOP: number };
}

interface GoogleMap {
	setCenter(loc: LngLat): void;
	setZoom(zoom: number): void;
	setMapTypeId(mapTypeId: string): void;
	getCenter(): { lat(): number; lng(): number } | undefined;
	getZoom(): number | undefined;
	// Google accepts a LatLngBoundsLiteral ({north,south,east,west}) directly,
	// plus an optional pixel padding. We pass the literal so we never have to
	// construct a LatLngBounds instance for framing.
	fitBounds(
		bounds: { north: number; south: number; east: number; west: number },
		padding?: number
	): void;
	addListener(event: string, handler: () => void): { remove(): void };
}

// google.maps.LatLngBounds as returned on a geocoder result's
// geometry.viewport / geometry.bounds. We only read its two corners.
interface GoogleLatLngBounds {
	getNorthEast(): { lat(): number; lng(): number };
	getSouthWest(): { lat(): number; lng(): number };
}

export interface GoogleAddressComponent {
	long_name: string;
	short_name: string;
	types: string[];
}

interface GoogleGeocoderResult {
	formatted_address: string;
	geometry: {
		location: { lat(): number; lng(): number };
		// Always present — Google's recommended display window for the result.
		viewport?: GoogleLatLngBounds;
		// Present only for results with a precise extent (e.g. a region).
		bounds?: GoogleLatLngBounds;
	};
	address_components?: GoogleAddressComponent[];
}

// Normalize a Google LatLngBounds (geometry.viewport / .bounds) into the
// workspace BoundingBox. Prefers the tighter `bounds` when present, else the
// always-available `viewport`. Returns undefined when neither exists.
function boundsFromGeometry(geometry: GoogleGeocoderResult['geometry']): BoundingBox | undefined {
	const b = geometry.bounds ?? geometry.viewport;
	if (!b) return undefined;
	const ne = b.getNorthEast();
	const sw = b.getSouthWest();
	return { north: ne.lat(), east: ne.lng(), south: sw.lat(), west: sw.lng() };
}

interface GoogleGeocoder {
	geocode(
		request:
			| { address: string; region?: string; bounds?: unknown }
			| { location: LngLat; region?: string },
		callback: (results: GoogleGeocoderResult[] | null, status: string) => void
	): void;
}

// Map Google address_components types → StructuredAddress fields.
// Priority order matters: locality wins over sublocality_level_1 for
// `locality`; admin_area_level_2 covers Thai district (อำเภอ) while
// admin_area_level_1 maps to province (จังหวัด).
export function parseAddressComponents(
	components: GoogleAddressComponent[] | undefined
): StructuredAddress {
	const out: StructuredAddress = {};
	if (!components) return out;
	for (const c of components) {
		const t = c.types;
		if (!out.street_number && t.includes('street_number')) out.street_number = c.long_name;
		if (!out.route && t.includes('route')) out.route = c.long_name;
		if (!out.sub_locality && (t.includes('sublocality') || t.includes('sublocality_level_1')))
			out.sub_locality = c.long_name;
		if (!out.locality && t.includes('locality')) out.locality = c.long_name;
		if (!out.admin_area_2 && t.includes('administrative_area_level_2'))
			out.admin_area_2 = c.long_name;
		if (!out.admin_area_1 && t.includes('administrative_area_level_1'))
			out.admin_area_1 = c.long_name;
		if (!out.postal_code && t.includes('postal_code')) out.postal_code = c.long_name;
		if (!out.country && t.includes('country')) {
			out.country = c.long_name;
			out.country_code = c.short_name;
		}
	}
	return out;
}

declare global {
	interface Window {
		google?: { maps?: GoogleMapsNamespace };
	}
}

// Thin typed wrapper over the shared loader — the loader returns an
// opaque namespace; the adapter narrows it to its own typed view.
function loadGoogleMaps(apiKey: string): Promise<GoogleMapsNamespace> {
	return loadGoogleMapsShared(apiKey) as Promise<GoogleMapsNamespace>;
}

// Adapter factory -----------------------------------------------------------

export function createGoogleAdapter(apiKey: string): MapAdapter {
	if (!apiKey) {
		throw new MapConfigError('PUBLIC_GOOGLE_MAPS_API_KEY is empty');
	}

	return {
		async mount(opts: MountOpts): Promise<MapHandle> {
			const maps = await loadGoogleMaps(opts.apiKey || apiKey);
			// `disableDefaultUI: true` is the floor for BOTH modes — it kills all
			// of Google's default UI (street view, pegman, rotate, AND the native
			// fullscreen button whose fullscreen covers only this bare map div,
			// dropping the picker's overlays). On top of that floor:
			//  • read-only → a static pin preview: no gestures, no zoom control,
			//    no keyboard — zero interactive controls render.
			//  • interactive → the editor chrome: a compact zoom stack bottom-right.
			//    Fullscreen + map-type are the picker's own custom in-canvas
			//    controls (which fullscreen the whole canvas, overlays included),
			//    matching PolygonPicker. (MAP-03/05.)
			const chrome = opts.readonly
				? {
						gestureHandling: 'none',
						keyboardShortcuts: false,
						disableDoubleClickZoom: true,
						clickableIcons: false
					}
				: {
						zoomControl: true,
						zoomControlOptions: { position: maps.ControlPosition.RIGHT_BOTTOM },
						fullscreenControl: false
					};
			const map = new maps.Map(opts.container, {
				center: opts.center,
				zoom: opts.zoom,
				disableDefaultUI: true,
				...chrome,
				// Default base map type when the consumer requests one (admin
				// pickers pass 'hybrid'). Undefined → Google's roadmap default.
				mapTypeId: opts.mapType
			});

			// Center-lock-on-zoom (opts.lockCenterOnZoom — point pickers). Google
			// anchors scroll / double-click zoom at the CURSOR, dragging the map
			// center (and a center pin) off the placed spot. We keep the pre-zoom
			// center fixed so zoom happens ABOUT the pin. `anchor` is the center to
			// preserve (updated only when a PAN or our own camera move settles —
			// never from a zoom's cursor drift); `programmatic` suppresses the lock
			// during handle-driven setCenter/setZoom/fitBounds.
			let anchor: LngLat = { lat: opts.center.lat, lng: opts.center.lng };
			let anchorZoom = opts.zoom;
			let programmatic = false;

			const handle: MapHandle & { _listeners: Array<{ remove(): void }> } = {
				provider: 'google',
				_listeners: [],
				destroy() {
					this._listeners.forEach((l) => l.remove());
					this._listeners = [];
					maps.event.clearInstanceListeners(map);
				},
				setCenter(loc: LngLat) {
					programmatic = true;
					anchor = loc;
					map.setCenter(loc);
				},
				setZoom(zoom: number) {
					programmatic = true;
					map.setZoom(zoom);
				},
				setMapType(mapType: string) {
					map.setMapTypeId(mapType);
				},
				resize() {
					// Google paints tiles at the container's last-measured size.
					// After a fullscreen enter/exit the container's box changes;
					// fire the resize event so Google re-reads it, then restore the
					// center (a resize pins the top-left corner, which would
					// otherwise shift the center off the pin).
					const c = map.getCenter();
					maps.event.trigger(map, 'resize');
					if (c) map.setCenter({ lat: c.lat(), lng: c.lng() });
				},
				fitBounds(box: BoundingBox, padding?: number) {
					programmatic = true;
					map.fitBounds(
						{ north: box.north, south: box.south, east: box.east, west: box.west },
						padding
					);
				},
				getViewport(): MapViewport {
					const c = map.getCenter();
					return {
						lat: c?.lat() ?? opts.center.lat,
						lng: c?.lng() ?? opts.center.lng,
						zoom: map.getZoom() ?? opts.zoom
					};
				}
			};

			handle._listeners.push(
				// Pull the center back to the pin mid-zoom (reduces drift flicker); the
				// idle handler below is the authoritative correction.
				map.addListener('zoom_changed', () => {
					if (opts.lockCenterOnZoom && !programmatic) map.setCenter(anchor);
				}),
				map.addListener('idle', () => {
					const c = map.getCenter();
					const z = map.getZoom();
					if (opts.lockCenterOnZoom && !programmatic && z !== anchorZoom) {
						// A USER zoom settled — keep the pin centered; discard the
						// cursor-anchored drift Google applied (anchor is NOT updated).
						map.setCenter(anchor);
					} else if (c) {
						// Pan settled / our own camera move / lock off — adopt the center.
						anchor = { lat: c.lat(), lng: c.lng() };
					}
					anchorZoom = z ?? anchorZoom;
					programmatic = false;
					if (handle.onmove) handle.onmove(handle.getViewport());
				}),
				// `dragend` fires only after a user-initiated pan stops (mouse,
				// touch, or keyboard arrow). It does NOT fire on programmatic
				// setCenter — exactly the discriminator we need to gate
				// reverse-geocoding to real user motion.
				map.addListener('dragend', () => {
					if (handle.ondragend) handle.ondragend(handle.getViewport());
				})
			);

			return handle;
		},

		async reverseGeocode(query: ReverseGeocodeQuery): Promise<GeocodeResult | null> {
			const maps = await loadGoogleMaps(apiKey);
			const geocoder = new maps.Geocoder();
			return new Promise<GeocodeResult | null>((resolve, reject) => {
				geocoder.geocode(
					{ location: { lat: query.lat, lng: query.lng }, region: query.country },
					(results, status) => {
						if (status === 'ZERO_RESULTS') {
							resolve(null);
							return;
						}
						if (status !== 'OK' || !results || results.length === 0) {
							reject(new Error(`Google reverse geocode error: ${status}`));
							return;
						}
						// Google returns results most-specific → least. For
						// ocean / empty coordinates the first hit is a Plus
						// Code (`6PX2M5HC+MP`) — not useful in an address
						// form. Skip those and prefer the first hit with at
						// least one human-readable administrative component.
						const useful = results.find((r) => {
							const c = parseAddressComponents(r.address_components);
							return Boolean(
								c.route ||
									c.locality ||
									c.sub_locality ||
									c.admin_area_2 ||
									c.admin_area_1
							);
						});
						const r = useful ?? results[0];
						resolve({
							display_name: r.formatted_address,
							lat: r.geometry.location.lat(),
							lng: r.geometry.location.lng(),
							components: parseAddressComponents(r.address_components),
							bounds: boundsFromGeometry(r.geometry)
						});
					}
				);
			});
		},

		async search(query: GeocodeQuery): Promise<GeocodeResult[]> {
			const maps = await loadGoogleMaps(apiKey);
			const geocoder = new maps.Geocoder();
			return new Promise<GeocodeResult[]>((resolve, reject) => {
				geocoder.geocode(
					{ address: query.q, region: query.country },
					(results, status) => {
						if (status === 'ZERO_RESULTS') {
							resolve([]);
							return;
						}
						if (status !== 'OK' || !results) {
							reject(new Error(`Google Geocoder error: ${status}`));
							return;
						}
						const limit = query.limit ?? 5;
						resolve(
							results.slice(0, limit).map((r) => ({
								display_name: r.formatted_address,
								lat: r.geometry.location.lat(),
								lng: r.geometry.location.lng(),
								components: parseAddressComponents(r.address_components),
								bounds: boundsFromGeometry(r.geometry)
							}))
						);
					}
				);
			});
		},

		destroy(handle: MapHandle): void {
			handle.destroy();
		}
	};
}
