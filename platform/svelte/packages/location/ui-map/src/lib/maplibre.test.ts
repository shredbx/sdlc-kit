import { describe, expect, it } from 'vitest';
import type { GeoCoordinate } from '@sbx/units';
import {
	OPENFREEMAP_STYLE,
	directionsUrl,
	edgeToLineFeature,
	edgesToHitFeatures,
	resolveViewerStyle,
	ringToPolygonFeature
} from './maplibre.js';

const SQUARE: GeoCoordinate[] = [
	[100.0, 9.75],
	[100.001, 9.75],
	[100.001, 9.751],
	[100.0, 9.751]
];

describe('resolveViewerStyle', () => {
	it('returns the OpenFreeMap dev style endpoint', () => {
		expect(resolveViewerStyle()).toBe(OPENFREEMAP_STYLE);
	});
	it('is a key-free https url (no api key on the wire)', () => {
		expect(OPENFREEMAP_STYLE.startsWith('https://')).toBe(true);
		expect(OPENFREEMAP_STYLE.toLowerCase()).not.toContain('key=');
	});
});

describe('ringToPolygonFeature', () => {
	it('closes the ring (repeats the first position last)', () => {
		const f = ringToPolygonFeature(SQUARE);
		expect(f).not.toBeNull();
		const coords = f!.geometry.coordinates[0];
		expect(coords).toHaveLength(SQUARE.length + 1);
		expect(coords[0]).toEqual(coords[coords.length - 1]);
		expect(f!.geometry.type).toBe('Polygon');
	});
	it('does not double-close an already-closed ring', () => {
		const f = ringToPolygonFeature([...SQUARE, SQUARE[0]]);
		expect(f!.geometry.coordinates[0]).toHaveLength(SQUARE.length + 1);
	});
	it('returns null for a ring that cannot form a polygon', () => {
		expect(ringToPolygonFeature([])).toBeNull();
		expect(ringToPolygonFeature([[100, 9.75]])).toBeNull();
		expect(
			ringToPolygonFeature([
				[100, 9.75],
				[100.001, 9.75]
			])
		).toBeNull();
	});
});

describe('edgeToLineFeature', () => {
	it('builds the 2-point segment for edge i (node i → node i+1)', () => {
		const f = edgeToLineFeature(SQUARE, 0);
		expect(f).not.toBeNull();
		expect(f!.geometry.type).toBe('LineString');
		expect(f!.geometry.coordinates).toEqual([SQUARE[0], SQUARE[1]]);
	});
	it('wraps the closing edge (last → first)', () => {
		const f = edgeToLineFeature(SQUARE, 3);
		expect(f!.geometry.coordinates).toEqual([SQUARE[3], SQUARE[0]]);
	});
	it('returns null for a null index or out-of-range index', () => {
		expect(edgeToLineFeature(SQUARE, null)).toBeNull();
		expect(edgeToLineFeature(SQUARE, -1)).toBeNull();
		expect(edgeToLineFeature(SQUARE, 99)).toBeNull();
	});
});

describe('edgesToHitFeatures', () => {
	it('emits one indexed LineString per edge (n edges, closing edge included)', () => {
		const fc = edgesToHitFeatures(SQUARE);
		expect(fc.type).toBe('FeatureCollection');
		expect(fc.features).toHaveLength(SQUARE.length);
		// Edge i carries its index and the segment node i → node (i+1)%n.
		fc.features.forEach((f, i) => {
			expect(f.properties.edge).toBe(i);
			expect(f.geometry.type).toBe('LineString');
			expect(f.geometry.coordinates).toEqual([SQUARE[i], SQUARE[(i + 1) % SQUARE.length]]);
		});
	});
	it('returns an empty collection for a ring too small to have edges', () => {
		expect(edgesToHitFeatures([]).features).toHaveLength(0);
		expect(edgesToHitFeatures([[100, 9.75]]).features).toHaveLength(0);
	});
});

describe('directionsUrl', () => {
	it('builds a key-free google maps directions deep-link to the pin', () => {
		const url = directionsUrl({ lat: 9.75, lng: 100.0 });
		expect(url).toContain('google.com/maps/dir/');
		expect(url).toContain('destination=9.75,100');
		expect(url.toLowerCase()).not.toContain('key=');
	});
});
