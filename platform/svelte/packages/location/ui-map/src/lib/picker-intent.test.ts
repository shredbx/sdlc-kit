import { describe, expect, it } from 'vitest';
import {
	coordsChanged,
	deepestAdminQuery,
	deepestFilledLevel,
	DEFAULT_ADMIN_ZOOM,
	initialPinState,
	targetZoom
} from './picker-intent.js';

describe('initialPinState', () => {
	const KO_PHANGAN = { lat: 9.7489, lng: 100.031 };

	it('returns dirty=false when lat/lng are undefined', () => {
		expect(initialPinState(undefined, undefined, KO_PHANGAN)).toEqual({
			center: KO_PHANGAN,
			dirty: false
		});
	});

	it('returns dirty=true with clamped coords when both valid', () => {
		expect(initialPinState(13.7563, 100.5018, KO_PHANGAN)).toEqual({
			center: { lat: 13.7563, lng: 100.5018 },
			dirty: true
		});
	});

	it('treats null-island (0,0) as unset', () => {
		expect(initialPinState(0, 0, KO_PHANGAN)).toEqual({
			center: KO_PHANGAN,
			dirty: false
		});
	});

	it('treats NaN coords as unset', () => {
		expect(initialPinState(NaN, 100, KO_PHANGAN)).toEqual({
			center: KO_PHANGAN,
			dirty: false
		});
	});

	it('treats out-of-range coords as unset', () => {
		expect(initialPinState(13, 999, KO_PHANGAN)).toEqual({
			center: KO_PHANGAN,
			dirty: false
		});
	});
});

describe('coordsChanged', () => {
	it('returns true when next differs from prev', () => {
		expect(coordsChanged({ lat: 9, lng: 100 }, { lat: 13, lng: 100 })).toBe(true);
	});

	it('returns false when coords are equal within epsilon (the bug — mount idle at same center)', () => {
		// This is exactly the mount-time case: idle fires with the requested center.
		expect(coordsChanged({ lat: 9.7489, lng: 100.031 }, { lat: 9.7489, lng: 100.031 })).toBe(false);
	});

	it('returns false when coords differ by less than 1e-6 (epsilon noise)', () => {
		expect(coordsChanged({ lat: 9.7489, lng: 100.031 }, { lat: 9.7489000001, lng: 100.031 })).toBe(false);
	});

	it('returns true when coords differ by 1e-5 (meaningful pan)', () => {
		expect(coordsChanged({ lat: 9.7489, lng: 100.031 }, { lat: 9.749, lng: 100.031 })).toBe(true);
	});
});

describe('deepestAdminQuery', () => {
	it('returns null for undefined frame', () => {
		expect(deepestAdminQuery(undefined)).toBeNull();
	});

	it('returns null when every level is empty or whitespace', () => {
		expect(deepestAdminQuery({})).toBeNull();
		expect(deepestAdminQuery({ subDistrict: '  ', province: '' })).toBeNull();
	});

	it('assembles finest → coarsest from a full Thai cascade', () => {
		expect(
			deepestAdminQuery({
				subDistrict: 'Baan Tai',
				district: 'Koh Phangan',
				province: 'Surat Thani',
				country: 'Thailand'
			})
		).toBe('Baan Tai, Koh Phangan, Surat Thani, Thailand');
	});

	it('anchors on the deepest filled level, appending coarser filled levels only', () => {
		// Only province + country set — district/sub-district empty.
		expect(deepestAdminQuery({ province: 'Surat Thani', country: 'Thailand' })).toBe(
			'Surat Thani, Thailand'
		);
	});

	it('skips empty intermediate levels but keeps order', () => {
		// district missing between sub-district and province.
		expect(
			deepestAdminQuery({ subDistrict: 'Baan Tai', province: 'Surat Thani', country: 'Thailand' })
		).toBe('Baan Tai, Surat Thani, Thailand');
	});

	it('trims surrounding whitespace on each level', () => {
		expect(deepestAdminQuery({ district: '  Koh Phangan  ', country: ' Thailand ' })).toBe(
			'Koh Phangan, Thailand'
		);
	});
});

describe('deepestFilledLevel', () => {
	it('returns null for an undefined or empty frame', () => {
		expect(deepestFilledLevel(undefined)).toBeNull();
		expect(deepestFilledLevel({})).toBeNull();
		expect(deepestFilledLevel({ subDistrict: '  ', province: '' })).toBeNull();
	});

	it('returns the finest filled level (sub-district wins over coarser)', () => {
		expect(
			deepestFilledLevel({
				subDistrict: 'Baan Tai',
				district: 'Koh Phangan',
				province: 'Surat Thani',
				country: 'Thailand'
			})
		).toBe('subDistrict');
	});

	it('walks up to the deepest level actually filled', () => {
		expect(deepestFilledLevel({ province: 'Surat Thani', country: 'Thailand' })).toBe('province');
		expect(deepestFilledLevel({ district: 'Koh Phangan', country: 'Thailand' })).toBe('district');
		expect(deepestFilledLevel({ country: 'Thailand' })).toBe('country');
	});
});

describe('targetZoom (fixed zoom ladder)', () => {
	// User-confirmed ladder values (Google zoom integers).
	it('exposes the confirmed default ladder', () => {
		expect(DEFAULT_ADMIN_ZOOM).toEqual({
			location: 16,
			subDistrict: 13,
			district: 11,
			province: 9,
			country: 6
		});
	});

	it('returns the location zoom (16, max) when pinned — regardless of admin fields', () => {
		expect(targetZoom(undefined, true)).toBe(16);
		expect(targetZoom({ province: 'Surat Thani' }, true)).toBe(16);
		expect(
			targetZoom({ subDistrict: 'Baan Tai', district: 'Koh Phangan' }, true)
		).toBe(16);
	});

	it('maps the deepest filled admin level to its ladder zoom when NOT pinned', () => {
		expect(targetZoom({ province: 'Surat Thani' }, false)).toBe(9);
		expect(targetZoom({ district: 'Koh Phangan', province: 'Surat Thani' }, false)).toBe(11);
		expect(
			targetZoom(
				{ subDistrict: 'Baan Tai', district: 'Koh Phangan', province: 'Surat Thani' },
				false
			)
		).toBe(13);
		expect(targetZoom({ country: 'Thailand' }, false)).toBe(6);
	});

	it('returns null when nothing is filled and no pin is set (no framing opinion)', () => {
		expect(targetZoom(undefined, false)).toBeNull();
		expect(targetZoom({}, false)).toBeNull();
		expect(targetZoom({ province: '  ' }, false)).toBeNull();
	});

	it('honours a consumer-supplied ladder override', () => {
		const ladder = { location: 18, subDistrict: 14, district: 12, province: 10, country: 5 };
		expect(targetZoom(undefined, true, ladder)).toBe(18);
		expect(targetZoom({ province: 'Surat Thani' }, false, ladder)).toBe(10);
	});
});

// SC-LOC-001 — drag/search must NOT commit. The pure intent contract is that
// the picker only commits when its caller invokes emitChange; coordsChanged is
// the gate emitChange uses, and it is the ONLY commit-deciding helper. Search
// (handleSelect) and drag (ondragend) no longer call emitChange at all in the
// component — they pan + preview only. This test pins the pure invariant that
// re-confirming the SAME coordinate is a no-op (so even an explicit Set on an
// unmoved camera doesn't double-fire onchange), which is what decouples the
// passive preview from the commit.
describe('coordsChanged decouples preview from commit', () => {
	it('treats a re-pan to the same point as no change (preview is not a commit)', () => {
		const point = { lat: 9.7489, lng: 100.031 };
		expect(coordsChanged(point, point)).toBe(false);
	});

	it('reports a real move as a change (the Set button commits a moved camera)', () => {
		expect(coordsChanged({ lat: 9.7489, lng: 100.031 }, { lat: 9.8, lng: 100.031 })).toBe(true);
	});
});
