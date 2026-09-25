// TDD — @sbx/canvas-kit SourceProvider location/region capability (Decision #0297).
// Verifies that the optional location() and region() methods on SourceProvider
// satisfy the contract: present = capable, null return = capable but empty,
// absent = not capable. Also exercises the 'location' and 'region' FieldType tokens.
//
// CASE ENUMERATION
//   TC-SC-01 success  a provider with location() resolves { lat, lng } for a record
//   TC-SC-02 edge     location() returns null when the record has no location stored
//   TC-SC-03 success  a provider without location() does not expose the method (not capable)
//   TC-SC-04 success  a provider with region() resolves a GeoJSON ring [lng,lat][]
//   TC-SC-05 edge     region() returns null when the record has no polygon stored
//   TC-SC-06 success  a provider without region() does not expose the method (not capable)
//   TC-SC-07 success  'location' and 'region' are valid FieldType values
//   TC-SC-08 success  a provider can expose both location() and region()

import { describe, expect, it } from 'vitest';
import type { SourceProvider, FieldType } from './source.js';

// ---------------------------------------------------------------------------
// Minimal stub helpers — these implement ONLY what the test exercises so the
// contract is verified without depending on a real network or API.
// ---------------------------------------------------------------------------

/** A provider that declares location capability and returns a known point. */
function makeLocationProvider(lat: number | null, lng: number | null): SourceProvider {
	return {
		kind: 'property',
		label: 'Property',
		fields() {
			return [{ token: 'location', label: 'Location', type: 'location', category: 'field' }];
		},
		async resolve(_refId) {
			return {};
		},
		async list(_query) {
			return { records: [] };
		},
		async location(_refId) {
			if (lat === null || lng === null) return null;
			return { lat, lng };
		}
	};
}

/** A provider that declares region capability and returns a known polygon ring. */
function makeRegionProvider(ring: Array<[number, number]> | null): SourceProvider {
	return {
		kind: 'property',
		label: 'Property',
		fields() {
			return [{ token: 'region', label: 'Region', type: 'region', category: 'field' }];
		},
		async resolve(_refId) {
			return {};
		},
		async list(_query) {
			return { records: [] };
		},
		async region(_refId) {
			return ring;
		}
	};
}

/** A provider with NO location or region capability (the minimum required interface). */
function makeBasicProvider(): SourceProvider {
	return {
		kind: 'guide',
		label: 'Guide',
		fields() {
			return [];
		},
		async resolve(_refId) {
			return {};
		},
		async list(_query) {
			return { records: [] };
		}
	};
}

// ---------------------------------------------------------------------------
// TC-SC-01..03 — location() capability
// ---------------------------------------------------------------------------

describe('SourceProvider.location() capability', () => {
	it('TC-SC-01 provider with location() resolves { lat, lng } for a record', async () => {
		const provider = makeLocationProvider(13.7563, 100.5018);
		expect(provider.location).toBeDefined();
		const result = await provider.location!('rec-1');
		expect(result).toEqual({ lat: 13.7563, lng: 100.5018 });
	});

	it('TC-SC-02 location() returns null when the record has no location stored', async () => {
		const provider = makeLocationProvider(null, null);
		const result = await provider.location!('rec-empty');
		expect(result).toBeNull();
	});

	it('TC-SC-03 provider without location() does not expose the method (not capable)', () => {
		const provider = makeBasicProvider();
		expect(provider.location).toBeUndefined();
	});
});

// ---------------------------------------------------------------------------
// TC-SC-04..06 — region() capability
// ---------------------------------------------------------------------------

describe('SourceProvider.region() capability', () => {
	const exampleRing: Array<[number, number]> = [
		[100.5, 13.75],
		[100.51, 13.75],
		[100.51, 13.76],
		[100.5, 13.76],
		[100.5, 13.75] // closed ring
	];

	it('TC-SC-04 provider with region() resolves a GeoJSON ring [lng,lat][]', async () => {
		const provider = makeRegionProvider(exampleRing);
		expect(provider.region).toBeDefined();
		const result = await provider.region!('rec-1');
		expect(result).toEqual(exampleRing);
		// Verify GeoJSON [lng, lat] ordering (first element = [lng, lat])
		const [lng, lat] = result![0];
		expect(lng).toBe(100.5);
		expect(lat).toBe(13.75);
	});

	it('TC-SC-05 region() returns null when the record has no polygon stored', async () => {
		const provider = makeRegionProvider(null);
		const result = await provider.region!('rec-empty');
		expect(result).toBeNull();
	});

	it('TC-SC-06 provider without region() does not expose the method (not capable)', () => {
		const provider = makeBasicProvider();
		expect(provider.region).toBeUndefined();
	});
});

// ---------------------------------------------------------------------------
// TC-SC-07 — 'location' and 'region' are valid FieldType values
// ---------------------------------------------------------------------------

describe('FieldType: location and region tokens', () => {
	it('TC-SC-07 location and region are valid FieldType values (compile-time + runtime)', () => {
		// If FieldType does not include these, this assignment would be a TS type error.
		const locationType: FieldType = 'location';
		const regionType: FieldType = 'region';
		expect(locationType).toBe('location');
		expect(regionType).toBe('region');
	});
});

// ---------------------------------------------------------------------------
// TC-SC-08 — a provider can expose both location() and region()
// ---------------------------------------------------------------------------

describe('SourceProvider: combined location + region capability', () => {
	it('TC-SC-08 a provider can expose both location() and region()', async () => {
		const ring: Array<[number, number]> = [[100.5, 13.75], [100.51, 13.75], [100.5, 13.75]];
		const provider: SourceProvider = {
			kind: 'property',
			label: 'Property',
			fields() { return []; },
			async resolve(_refId) { return {}; },
			async list(_query) { return { records: [] }; },
			async location(_refId) { return { lat: 13.75, lng: 100.5 }; },
			async region(_refId) { return ring; }
		};
		expect(provider.location).toBeDefined();
		expect(provider.region).toBeDefined();
		const loc = await provider.location!('r1');
		expect(loc).toEqual({ lat: 13.75, lng: 100.5 });
		const reg = await provider.region!('r1');
		expect(reg).toHaveLength(3);
	});
});
