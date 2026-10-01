import { describe, expect, it } from 'vitest';
import { parseAddressComponents, type GoogleAddressComponent } from './google.js';

function comp(long: string, short: string, ...types: string[]): GoogleAddressComponent {
	return { long_name: long, short_name: short, types };
}

describe('parseAddressComponents', () => {
	it('returns empty for undefined input', () => {
		expect(parseAddressComponents(undefined)).toEqual({});
	});

	it('returns empty for empty array', () => {
		expect(parseAddressComponents([])).toEqual({});
	});

	it('parses a full Thai address (Koh Phangan)', () => {
		// Representative of what Google returns for "Thong Sala, Koh Phangan"
		const components: GoogleAddressComponent[] = [
			comp('123', '123', 'street_number'),
			comp('Thong Sala Road', 'Thong Sala Rd', 'route'),
			comp('Ko Pha Ngan', 'Ko Pha Ngan', 'locality', 'political'),
			comp('Ko Pha-ngan District', 'Ko Pha-ngan District', 'administrative_area_level_2', 'political'),
			comp('Surat Thani', 'Surat Thani', 'administrative_area_level_1', 'political'),
			comp('Thailand', 'TH', 'country', 'political'),
			comp('84280', '84280', 'postal_code')
		];

		expect(parseAddressComponents(components)).toEqual({
			street_number: '123',
			route: 'Thong Sala Road',
			locality: 'Ko Pha Ngan',
			admin_area_2: 'Ko Pha-ngan District',
			admin_area_1: 'Surat Thani',
			country: 'Thailand',
			country_code: 'TH',
			postal_code: '84280'
		});
	});

	it('falls back to sublocality when locality missing', () => {
		const components: GoogleAddressComponent[] = [
			comp('Bang Rak', 'Bang Rak', 'sublocality_level_1', 'sublocality', 'political'),
			comp('Bangkok', 'Bangkok', 'administrative_area_level_1', 'political')
		];
		const parsed = parseAddressComponents(components);
		expect(parsed.sub_locality).toBe('Bang Rak');
		expect(parsed.admin_area_1).toBe('Bangkok');
		expect(parsed.locality).toBeUndefined();
	});

	it('prefers locality over sublocality when both present', () => {
		const components: GoogleAddressComponent[] = [
			comp('Sublocality Name', 'Sub', 'sublocality_level_1'),
			comp('Locality Name', 'Loc', 'locality')
		];
		const parsed = parseAddressComponents(components);
		expect(parsed.locality).toBe('Locality Name');
		expect(parsed.sub_locality).toBe('Sublocality Name');
	});

	it('captures both country long_name and country_code', () => {
		const components: GoogleAddressComponent[] = [
			comp('United States', 'US', 'country', 'political')
		];
		const parsed = parseAddressComponents(components);
		expect(parsed.country).toBe('United States');
		expect(parsed.country_code).toBe('US');
	});

	it('ignores unknown component types', () => {
		const components: GoogleAddressComponent[] = [
			comp('Some Plus Code', '12345+67', 'plus_code'),
			comp('Bangkok', 'Bangkok', 'locality')
		];
		const parsed = parseAddressComponents(components);
		expect(parsed.locality).toBe('Bangkok');
		// No spurious keys
		expect(Object.keys(parsed)).toHaveLength(1);
	});

	it('keeps first occurrence when duplicates exist', () => {
		// Real-world: some addresses return multiple admin_area_level_1
		// entries (e.g., for ambiguous regions). First wins.
		const components: GoogleAddressComponent[] = [
			comp('Province A', 'A', 'administrative_area_level_1'),
			comp('Province B', 'B', 'administrative_area_level_1')
		];
		const parsed = parseAddressComponents(components);
		expect(parsed.admin_area_1).toBe('Province A');
	});

	it('handles street_number without route gracefully', () => {
		const components: GoogleAddressComponent[] = [comp('42', '42', 'street_number')];
		const parsed = parseAddressComponents(components);
		expect(parsed.street_number).toBe('42');
		expect(parsed.route).toBeUndefined();
	});
});
