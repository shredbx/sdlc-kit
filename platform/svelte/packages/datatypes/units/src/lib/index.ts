// Land-area unit conversions
export {
	LAND_SIZE_UNITS,
	LAND_SIZE_UNITS_ORDERED,
	convert,
	decimalPlacesFor,
	formatArea,
	formatAreaCompound,
	formatLandSize,
	formatLandSizePlain,
	fromSqm,
	isLandSizeUnit,
	toSqm,
	type AreaFormatOptions,
	type LandSizeUnit,
	type LandSizeUnitMeta
} from './land-units.js';

// Generic number formatting (S-FORMAT inc3)
export { formatNumber, type NumberFormatOptions } from './number.js';

// ISO date / datetime formatting (S-FORMAT inc3)
export { formatDate, type DateFormatOptions } from './date.js';

// Geodesic area + distance computation for GeoJSON polygons
export {
	formatLengthAuto,
	formatLengthFeet,
	geojsonPolygonAreaSqm,
	haversineMeters,
	isClosed,
	polygonAreaSqm,
	polygonFromLatLng,
	polygonPerimeterMeters,
	ringFromLatLng,
	type GeoCoordinate,
	type GeoJSONPolygon
} from './geo-area.js';
