// Tests for buildStaticMapPath (static-map.ts) and toStaticMapsColor.
//
// Coverage matrix ported from BR canvas/maps.ts reference, extended with the
// generalized descriptor interface. Guards verified:
//   - Color normalization (#hex / 3-digit / rgb→neutral)
//   - Ring [lng,lat]→lat,lng flip
//   - Null-island + invalid-coords omission
//   - Ring < 4 positions omission
//   - Marker styling (color:0xRRGGBB|lat,lng)
//   - Path stroke/fill/weight styling
//   - Scale param
//   - Auto-fit: center/zoom omitted when polygon present and no center given
//   - Security invariants: /api/map/static, NO key=, NO googleapis
//
// The legacy StaticMapOpts overload is NOT re-tested here — the old tests were
// for the old interface; the canonical surface is now StaticMapDescriptor.

import { describe, expect, it } from 'vitest';
import { buildStaticMapPath, toStaticMapsColor } from './static-map.js';

// ── toStaticMapsColor ─────────────────────────────────────────────────────────

describe('toStaticMapsColor — color normalization', () => {
	it('converts full 6-digit hex to 0xRRGGBB uppercase', () => {
		expect(toStaticMapsColor('#0D4F4F')).toBe('0x0D4F4F');
		expect(toStaticMapsColor('#e5392b')).toBe('0xE5392B');
		expect(toStaticMapsColor('#ffffff')).toBe('0xFFFFFF');
		expect(toStaticMapsColor('#000000')).toBe('0x000000');
	});

	it('expands 3-digit hex to 0xRRGGBB uppercase', () => {
		expect(toStaticMapsColor('#abc')).toBe('0xAABBCC');
		expect(toStaticMapsColor('#f00')).toBe('0xFF0000');
		expect(toStaticMapsColor('#0f0')).toBe('0x00FF00');
	});

	it('returns neutral for rgb() values (not a #hex literal)', () => {
		expect(toStaticMapsColor('rgb(100,200,50)')).toBe('0x333333');
		expect(toStaticMapsColor('rgba(0,0,0,0.5)')).toBe('0x333333');
	});

	it('returns neutral for named colors', () => {
		expect(toStaticMapsColor('red')).toBe('0x333333');
		expect(toStaticMapsColor('teal')).toBe('0x333333');
	});

	it('returns neutral for empty string', () => {
		expect(toStaticMapsColor('')).toBe('0x333333');
	});

	it('returns neutral for undefined', () => {
		expect(toStaticMapsColor(undefined)).toBe('0x333333');
	});

	it('returns neutral for 4-digit hex (not a valid 3 or 6 digit hex)', () => {
		expect(toStaticMapsColor('#abcd')).toBe('0x333333');
	});
});

// ── Security invariants ───────────────────────────────────────────────────────

const VALID_CENTER = { lat: 13.7563, lng: 100.5018 };
const VALID_SIZE = { width: 640, height: 400 };

describe('buildStaticMapPath — security: no API key, no googleapis', () => {
	it('returns a relative path starting with /api/map/static', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).toMatch(/^\/api\/map\/static\?/);
	});

	it('does not contain key= in any form', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: '#e5392b' }
		});
		expect(result).not.toMatch(/key=/i);
		expect(result).not.toMatch(/AIza/i);
	});

	it('does not reference googleapis.com', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).not.toContain('googleapis');
		expect(result).not.toContain('google.com');
		expect(result).not.toContain('gstatic');
	});

	it('is a relative path with no host component (no ://)', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).not.toContain('://');
		expect(result).not.toMatch(/^\/\//);
	});

	it('polygon path result also contains no key= and no googleapis', () => {
		const ring: Array<[number, number]> = [
			[100.5, 13.75],
			[100.6, 13.75],
			[100.6, 13.85],
			[100.5, 13.75] // closed ring, 4 positions
		];
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring } });
		expect(result).not.toMatch(/key=/i);
		expect(result).not.toContain('googleapis');
	});
});

// ── Core params ───────────────────────────────────────────────────────────────

describe('buildStaticMapPath — core params', () => {
	it('includes center, zoom, size, maptype, scale', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: { width: 640, height: 640 },
			scale: 2,
			mapType: 'satellite'
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('center')).toBe('13.7563,100.5018');
		expect(params.get('zoom')).toBe('15');
		expect(params.get('size')).toBe('640x640');
		expect(params.get('scale')).toBe('2');
		expect(params.get('maptype')).toBe('satellite');
	});

	it('defaults scale to 1 when not specified', () => {
		const result = buildStaticMapPath({ center: VALID_CENTER, zoom: 14, size: VALID_SIZE });
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('scale')).toBe('1');
	});

	it('defaults maptype to roadmap for unknown value', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			mapType: 'vectors'
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('maptype')).toBe('roadmap');
	});

	it('defaults maptype to roadmap when undefined', () => {
		const result = buildStaticMapPath({ center: VALID_CENTER, zoom: 14, size: VALID_SIZE });
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('maptype')).toBe('roadmap');
	});

	it('passes through all valid map types', () => {
		for (const mt of ['roadmap', 'satellite', 'hybrid', 'terrain'] as const) {
			const result = buildStaticMapPath({ center: VALID_CENTER, zoom: 14, size: VALID_SIZE, mapType: mt });
			const params = new URLSearchParams(result.split('?')[1]);
			expect(params.get('maptype')).toBe(mt);
		}
	});
});

// ── Zoom clamping ─────────────────────────────────────────────────────────────

describe('buildStaticMapPath — zoom clamping', () => {
	it('clamps zoom above 21 to 21', () => {
		const result = buildStaticMapPath({ center: VALID_CENTER, zoom: 30, size: VALID_SIZE });
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('zoom')).toBe('21');
	});

	it('clamps zoom below 0 to 0', () => {
		const result = buildStaticMapPath({ center: VALID_CENTER, zoom: -5, size: VALID_SIZE });
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('zoom')).toBe('0');
	});

	it('rounds fractional zoom values', () => {
		const result = buildStaticMapPath({ center: VALID_CENTER, zoom: 14.7, size: VALID_SIZE });
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('zoom')).toBe('15');
	});
});

// ── Coordinate rounding ───────────────────────────────────────────────────────

describe('buildStaticMapPath — coordinate rounding', () => {
	it('rounds center coords to 6 decimal places', () => {
		const result = buildStaticMapPath({
			center: { lat: 13.75634567891, lng: 100.50181234567 },
			zoom: 14,
			size: VALID_SIZE
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('center')).toBe('13.756346,100.501812');
	});
});

// ── Null-island and invalid coords ────────────────────────────────────────────

describe('buildStaticMapPath — null-island and invalid-coords guard', () => {
	it('returns empty string when center is null-island (0,0)', () => {
		const result = buildStaticMapPath({
			center: { lat: 0, lng: 0 },
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).toBe('');
	});

	it('returns empty string when center lat is out of range', () => {
		const result = buildStaticMapPath({
			center: { lat: 200, lng: 100 },
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).toBe('');
	});

	it('returns empty string when center lng is out of range', () => {
		const result = buildStaticMapPath({
			center: { lat: 13, lng: 200 },
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).toBe('');
	});

	it('returns empty string when center coords are NaN', () => {
		const result = buildStaticMapPath({
			center: { lat: NaN, lng: NaN },
			zoom: 14,
			size: VALID_SIZE
		});
		expect(result).toBe('');
	});

	it('returns empty string when nothing is drawable (no center, no marker, no polygon)', () => {
		const result = buildStaticMapPath({ size: VALID_SIZE });
		expect(result).toBe('');
	});

	it('omits marker when its coords are null-island (0,0)', () => {
		// No center → nothing drawable → returns ''
		const result = buildStaticMapPath({
			size: VALID_SIZE,
			marker: { lat: 0, lng: 0 }
		});
		expect(result).toBe('');
	});

	it('omits marker when its coords are invalid', () => {
		const result = buildStaticMapPath({
			size: VALID_SIZE,
			marker: { lat: 999, lng: 999 }
		});
		expect(result).toBe('');
	});
});

// ── Marker encoding ───────────────────────────────────────────────────────────

describe('buildStaticMapPath — marker encoding', () => {
	it('encodes a marker with CSS hex color: color:0xRRGGBB|lat,lng', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: '#e5392b' }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('markers')).toBe('color:0xE5392B|13.7563,100.5018');
	});

	it('normalizes 3-digit hex in marker color', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: '#abc' }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('markers')).toContain('color:0xAABBCC');
	});

	it('uses neutral color for rgb() marker color', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: 'rgb(200,50,0)' }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('markers')).toContain('color:0x333333');
	});

	it('uses neutral color when marker color is undefined', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018 }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('markers')).toContain('color:0x333333');
	});

	it('encodes the point with rounded coords', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: VALID_SIZE,
			marker: { lat: 13.756346789, lng: 100.501812345, color: '#0d4f4f' }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		// lat rounded to 6dp: 13.756347, lng: 100.501812
		expect(params.get('markers')).toBe('color:0x0D4F4F|13.756347,100.501812');
	});
});

// ── Polygon / path encoding ───────────────────────────────────────────────────

// A valid closed ring: 4 positions, [lng, lat] order (GeoJSON)
const VALID_RING: Array<[number, number]> = [
	[100.5, 13.75],
	[100.6, 13.75],
	[100.6, 13.85],
	[100.5, 13.75] // closed — first repeated
];

describe('buildStaticMapPath — polygon encoding', () => {
	it('flips [lng,lat] to lat,lng in path points', () => {
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: VALID_RING } });
		const params = new URLSearchParams(result.split('?')[1]);
		const path = params.get('path') ?? '';
		// First position in ring is [lng=100.5, lat=13.75] → should appear as 13.75,100.5
		expect(path).toContain('13.75,100.5');
		// NOT 100.5,13.75 (wrong order)
		expect(path).not.toContain('100.5,13.75');
	});

	it('emits stroke/fill colors and weight in path param', () => {
		const result = buildStaticMapPath({
			size: VALID_SIZE,
			polygon: { ring: VALID_RING, strokeColor: '#0d4f4f', fillColor: '#c8a851', weight: 2 }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		const path = params.get('path') ?? '';
		expect(path).toContain('color:0x0D4F4Fff');
		expect(path).toContain('fillcolor:0xC8A85133');
		expect(path).toContain('weight:2');
	});

	it('uses strokeColor for fillColor when fillColor is omitted', () => {
		const result = buildStaticMapPath({
			size: VALID_SIZE,
			polygon: { ring: VALID_RING, strokeColor: '#0d4f4f' }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		const path = params.get('path') ?? '';
		expect(path).toContain('color:0x0D4F4Fff');
		expect(path).toContain('fillcolor:0x0D4F4F33');
	});

	it('defaults weight to 3 when not specified', () => {
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: VALID_RING } });
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('path')).toContain('weight:3');
	});

	it('uses neutral colors when stroke/fill are undefined', () => {
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: VALID_RING } });
		const params = new URLSearchParams(result.split('?')[1]);
		const path = params.get('path') ?? '';
		expect(path).toContain('color:0x333333ff');
		expect(path).toContain('fillcolor:0x33333333');
	});

	it('omits path when ring has fewer than 4 positions', () => {
		const shortRing: Array<[number, number]> = [
			[100.5, 13.75],
			[100.6, 13.75],
			[100.5, 13.75]
		];
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: shortRing } });
		// Nothing drawable → empty string
		expect(result).toBe('');
	});

	it('omits path when ring contains non-finite positions', () => {
		const badRing: Array<[number, number]> = [
			[100.5, 13.75],
			[NaN, 13.75],
			[100.6, 13.85],
			[100.5, 13.75]
		];
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: badRing } });
		expect(result).toBe('');
	});
});

// ── Auto-fit (center/zoom omitted with polygon) ───────────────────────────────

describe('buildStaticMapPath — auto-fit mode', () => {
	it('omits center and zoom from URL when neither is provided but polygon is present', () => {
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: VALID_RING } });
		expect(result).toMatch(/^\/api\/map\/static\?/);
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('center')).toBeNull();
		expect(params.get('zoom')).toBeNull();
		// But path IS present
		expect(params.get('path')).not.toBeNull();
	});

	it('includes center and zoom when BOTH are explicitly provided with polygon', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 12,
			size: VALID_SIZE,
			polygon: { ring: VALID_RING }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('center')).toBe('13.7563,100.5018');
		expect(params.get('zoom')).toBe('12');
		expect(params.get('path')).not.toBeNull();
	});

	it('auto-fit result still starts with /api/map/static', () => {
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: VALID_RING } });
		expect(result).toMatch(/^\/api\/map\/static\?/);
	});
});

// ── Scale ─────────────────────────────────────────────────────────────────────

describe('buildStaticMapPath — scale', () => {
	it('emits scale=2 for BIND variant', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: { width: 640, height: 640 },
			scale: 2
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('scale')).toBe('2');
		expect(params.get('size')).toBe('640x640');
	});

	it('emits scale=1 for THUMB variant', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 15,
			size: { width: 320, height: 320 },
			scale: 1
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('scale')).toBe('1');
		expect(params.get('size')).toBe('320x320');
	});
});

// ── URL length guard ──────────────────────────────────────────────────────────

describe('buildStaticMapPath — URL length guard', () => {
	it('returns empty string when composed path exceeds 7500 chars', () => {
		// Generate a ring with enough points to blow past the limit
		const bigRing: Array<[number, number]> = [];
		for (let i = 0; i < 500; i++) {
			bigRing.push([100.5 + i * 0.001, 13.75 + i * 0.001]);
		}
		// Close the ring
		bigRing.push(bigRing[0]);
		const result = buildStaticMapPath({ size: VALID_SIZE, polygon: { ring: bigRing } });
		expect(result).toBe('');
	});
});

// ── Combined marker + polygon ─────────────────────────────────────────────────

describe('buildStaticMapPath — marker and polygon combined', () => {
	it('emits both markers and path params', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 13,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: '#e5392b' },
			polygon: { ring: VALID_RING, strokeColor: '#0d4f4f' }
		});
		const params = new URLSearchParams(result.split('?')[1]);
		expect(params.get('markers')).toContain('color:0xE5392B');
		expect(params.get('path')).toContain('color:0x0D4F4Fff');
	});
});

// ── Multi-marker / multi-polygon (markers[] / polygons[]) ─────────────────────

const VALID_RING_2: Array<[number, number]> = [
	[101.0, 14.0],
	[101.1, 14.0],
	[101.1, 14.1],
	[101.0, 14.0]
];

describe('buildStaticMapPath — markers[] multi-entry', () => {
	it('TC-SM-M1 emits two repeated markers= params for two markers', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			markers: [
				{ lat: 13.7563, lng: 100.5018, color: '#e5392b' },
				{ lat: 13.8, lng: 100.6, color: '#0d4f4f' }
			]
		});
		expect(result).toMatch(/^\/api\/map\/static\?/);
		const raw = result.split('?')[1];
		const all = new URLSearchParams(raw).getAll('markers');
		expect(all).toHaveLength(2);
		expect(all[0]).toContain('color:0xE5392B');
		expect(all[1]).toContain('color:0x0D4F4F');
	});

	it('TC-SM-M2 skips invalid markers (null-island) in the array', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			markers: [
				{ lat: 0, lng: 0 }, // null-island — skipped
				{ lat: 13.7563, lng: 100.5018, color: '#e5392b' }
			]
		});
		const all = new URLSearchParams(result.split('?')[1]).getAll('markers');
		expect(all).toHaveLength(1);
		expect(all[0]).toContain('color:0xE5392B');
	});

	it('TC-SM-M3 back-compat: singular marker= still emits one markers= entry', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: '#e5392b' }
		});
		const all = new URLSearchParams(result.split('?')[1]).getAll('markers');
		expect(all).toHaveLength(1);
		expect(all[0]).toContain('color:0xE5392B');
	});

	it('TC-SM-M4 singular marker + markers[] together emit both entries', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			marker: { lat: 13.7563, lng: 100.5018, color: '#e5392b' },
			markers: [{ lat: 13.8, lng: 100.6, color: '#0d4f4f' }]
		});
		const all = new URLSearchParams(result.split('?')[1]).getAll('markers');
		expect(all).toHaveLength(2);
	});
});

describe('buildStaticMapPath — polygons[] multi-entry', () => {
	it('TC-SM-P1 emits two repeated path= params for two polygons', () => {
		const result = buildStaticMapPath({
			size: VALID_SIZE,
			polygons: [
				{ ring: VALID_RING, strokeColor: '#0d4f4f' },
				{ ring: VALID_RING_2, strokeColor: '#e5392b' }
			]
		});
		expect(result).toMatch(/^\/api\/map\/static\?/);
		const all = new URLSearchParams(result.split('?')[1]).getAll('path');
		expect(all).toHaveLength(2);
		expect(all[0]).toContain('color:0x0D4F4Fff');
		expect(all[1]).toContain('color:0xE5392Bff');
	});

	it('TC-SM-P2 skips an invalid polygon (< 4 positions) in the array', () => {
		const shortRing: Array<[number, number]> = [[100.5, 13.75], [100.6, 13.75], [100.5, 13.75]];
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			polygons: [
				{ ring: shortRing }, // invalid — skipped
				{ ring: VALID_RING, strokeColor: '#0d4f4f' }
			]
		});
		const all = new URLSearchParams(result.split('?')[1]).getAll('path');
		expect(all).toHaveLength(1);
		expect(all[0]).toContain('color:0x0D4F4Fff');
	});

	it('TC-SM-P3 singular polygon + polygons[] together emit both path= entries', () => {
		const result = buildStaticMapPath({
			size: VALID_SIZE,
			polygon: { ring: VALID_RING },
			polygons: [{ ring: VALID_RING_2 }]
		});
		const all = new URLSearchParams(result.split('?')[1]).getAll('path');
		expect(all).toHaveLength(2);
	});
});

describe('buildStaticMapPath — multi security invariant', () => {
	it('TC-SM-S1 multi-marker + multi-polygon result contains no key= and no googleapis', () => {
		const result = buildStaticMapPath({
			center: VALID_CENTER,
			zoom: 14,
			size: VALID_SIZE,
			markers: [
				{ lat: 13.7563, lng: 100.5018, color: '#e5392b' },
				{ lat: 13.8, lng: 100.6 }
			],
			polygons: [
				{ ring: VALID_RING },
				{ ring: VALID_RING_2 }
			]
		});
		expect(result).toMatch(/^\/api\/map\/static\?/);
		expect(result).not.toMatch(/key=/i);
		expect(result).not.toContain('googleapis');
		expect(result).not.toContain('google.com');
		expect(result).not.toContain('://');
	});
});
