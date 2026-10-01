// Types --------------------------------------------------------------------
export type {
	AddressDiffField,
	AdminFrameQuery,
	BoundingBox,
	GeocodeQuery,
	GeocodeResult,
	GeocodeService,
	LngLat,
	LocationPickerProps,
	MapAdapter,
	MapHandle,
	MapPolygon,
	MapProvider,
	MapSurface,
	MapViewState,
	MapViewport,
	MountOpts,
	PolygonPickerProps,
	PolygonRing,
	ReverseGeocodeQuery,
	StructuredAddress
} from './types.js';
export { MapConfigError, MapNotImplementedError } from './types.js';

// Pure helpers -------------------------------------------------------------
export {
	clampCoords,
	clampLat,
	coordsEqual,
	isNullIsland,
	isValidCoords,
	normalizeLng
} from './coords.js';

// Canonical markers — ONE source of truth for the property pin (red teardrop + white circle)
// and the area glyph (dashed halo + name caption). Every map surface derives from here so the
// location marker never drifts into several looks (task 2607-115).
export {
	AREA_TONE_RGB,
	PIN_RED,
	areaMarkerElement,
	locationPinDataUri,
	locationPinElement,
	locationPinSvg,
	pricePillElement,
	clusterPillElement,
	estimatePillSize
} from './markers.js';
export type { PricePillOptions } from './markers.js';

// Pixel-collision declustering — pure, unit-tested (used by the board adapter).
export { COLLIDE_GAP, declusterByPixel, markersCollide } from './cluster.js';
export type { PixelPoint, ClusterCell } from './cluster.js';

// Provider resolution ------------------------------------------------------
export { readBrowserEnv, resolveApiKey, resolveProvider } from './env.js';

// View defaults — localStorage-backed last-used camera per picker family ---
export { clearDefaultView, getDefaultView, setDefaultView } from './viewStore.js';

// Adapter factories --------------------------------------------------------
export { createGoogleAdapter } from './adapters/google.js';
export { createMapboxAdapter } from './adapters/mapbox.js';

// Static map path builder (proxy-safe — no API key emitted) ---------------
export { buildStaticMapPath, toStaticMapsColor } from './static-map.js';
export type {
	StaticMapDescriptor,
	StaticMapMarker,
	StaticMapMarkerDescriptor,
	StaticMapOpts,
	StaticMapPathPoint,
	StaticMapPolygonDescriptor,
	StaticMapType
} from './static-map.js';

// Components ---------------------------------------------------------------
export { default as LocationPicker } from './LocationPicker.svelte';
export { default as GeocodeSearch } from './GeocodeSearch.svelte';
export { default as AddressDiffConfirm } from './AddressDiffConfirm.svelte';
export { default as PolygonPicker } from './PolygonPicker.svelte';

// Parcel viewer — provider-agnostic, read-only map (#0304). MapStage is the map SCREEN
// (map + pin + plot + legend + fallback); MapControls is its switcher + "Get directions".
// MapViewer is the FULL-BLEED shell wrapping both. A consumer that wants the map inside a
// floating dialog renders <MapStage/> + <MapControls/> in its own shell. createMapViewerAdapter(domain)
// picks the map (public → MapLibre terrain, admin → Google). The heavy engine is
// dynamically imported inside its adapter, so consumers stay free of the WebGL bundle.
export { default as MapViewer } from './MapViewer.svelte';
export { default as MapStage } from './MapStage.svelte';
export { default as MapControls } from './MapControls.svelte';
export { default as PlotLegend } from './PlotLegend.svelte';
export { createMapViewerAdapter } from './map-viewer.js';
export { createMaplibreViewerAdapter } from './adapters/maplibre-viewer.js';
export { createGoogleViewerAdapter } from './adapters/google-viewer.js';
export type { MapViewerAdapter, MapViewerDomain, MapViewerMountOpts } from './types.js';

// Board — the MULTI-marker map screen (catalog Map view + homepage map section, task
// 2607-133). Sibling of MapStage/MapViewer, not a replacement: a viewer frames ONE property,
// a board plots MANY and reports the selection. createMapBoardAdapter(domain) picks the map
// the same way createMapViewerAdapter does (#0304); the WebGL engine is dynamically imported
// inside the adapter, so a page that never mounts a board ships none of it.
export { default as MapBoard } from './MapBoard.svelte';
export { createMapBoardAdapter } from './map-board.js';
export { createMaplibreBoardAdapter } from './adapters/maplibre-board.js';
export type { MapBoardAdapter, MapBoardFrame, MapBoardMountOpts, MapMarkerPoint } from './types.js';
export {
	BRAND_WATER_FILL,
	BRAND_WATERWAY_LINE,
	EMPTY_FEATURE_COLLECTION,
	OPENFREEMAP_STYLE,
	TERRAIN_DEM_ENCODING,
	TERRAIN_DEM_TILES,
	applyBrandWaterWash,
	directionsUrl,
	edgeToLineFeature,
	edgesToHitFeatures,
	resolveViewerStyle,
	ringToPolygonFeature
} from './maplibre.js';
export type {
	EdgeHitCollection,
	EdgeHitFeature,
	EmptyFeatureCollection,
	GeoFeature,
	LineStringGeometry,
	PolygonGeometry
} from './maplibre.js';
export {
	MAX_PARCEL_VERTICES,
	formatParcelArea,
	formatParcelLength,
	nodeLabels,
	normalizedRingPoints,
	parcelEdges,
	parcelStats,
	ringBounds,
	sanitizeRing
} from './parcel.js';
export type { LandSizeUnit, ParcelEdge, ParcelStats, Point2D } from './parcel.js';
