// TDD — @sbx/canvas-kit makeEnumFormatter (INC-D · the enum/list display formatter).
// A string-valued formatter (valueType 'string' — INC-A) that maps a raw enum code to a
// marketing ALIAS from a consumer-owned map, keyed by the format hint's arg (the enum SET),
// falling back to a humanized code; a 'case' knob restyles the result. The alias map is read
// at format time so a later persistence layer can swap marketing's edits in without rebuilds.
//   TC-EN-01  alias hit: 'sale' @ set 'transactionType' → 'For Sale'
//   TC-EN-02  no alias → humanized code ('pool-villa' → 'Pool Villa')
//   TC-EN-03  unknown SET → humanized (the map has no such set)
//   TC-EN-04  case knob: upper / lower / title restyle the alias
//   TC-EN-05  it is a string-valued descriptor (valueType 'string') so resolve passes raw text
//   TC-EN-06  editing the alias map after construction is reflected (read at format time)
import { describe, it, expect } from 'vitest';
import type { EnumAliasMap } from './enum-formatter.js';
import { makeEnumFormatter } from './enum-formatter.js';

const aliases: EnumAliasMap = {
	transactionType: { sale: 'For Sale', lease: 'For Lease' },
	propertyType: { condo: 'Condominium' }
};

describe('makeEnumFormatter', () => {
	it('TC-EN-01 maps a code to its alias for the named set', () => {
		const fmt = makeEnumFormatter(aliases);
		expect(fmt.format('sale', { case: 'default' }, 'transactionType')).toBe('For Sale');
		expect(fmt.format('lease', { case: 'default' }, 'transactionType')).toBe('For Lease');
		expect(fmt.format('condo', { case: 'default' }, 'propertyType')).toBe('Condominium');
	});

	it('TC-EN-02 humanizes a code with no alias (separators → spaces, title-cased)', () => {
		const fmt = makeEnumFormatter(aliases);
		expect(fmt.format('pool-villa', { case: 'default' }, 'propertyType')).toBe('Pool Villa');
		expect(fmt.format('town_house', { case: 'default' }, 'propertyType')).toBe('Town House');
	});

	it('TC-EN-03 humanizes when the SET is unknown', () => {
		const fmt = makeEnumFormatter(aliases);
		expect(fmt.format('sale', { case: 'default' }, 'no-such-set')).toBe('Sale');
		expect(fmt.format('sale', { case: 'default' })).toBe('Sale'); // no arg at all
	});

	it('TC-EN-04 the case knob restyles the resolved display', () => {
		const fmt = makeEnumFormatter(aliases);
		expect(fmt.format('sale', { case: 'upper' }, 'transactionType')).toBe('FOR SALE');
		expect(fmt.format('sale', { case: 'lower' }, 'transactionType')).toBe('for sale');
		expect(fmt.format('lease', { case: 'title' }, 'transactionType')).toBe('For Lease');
	});

	it('TC-EN-05 is a string-valued descriptor (INC-A raw-string path)', () => {
		const fmt = makeEnumFormatter(aliases);
		expect(fmt.valueType).toBe('string');
		expect(fmt.name).toBe('enum');
	});

	it('TC-EN-06 reads the alias map at format time (later persistence can swap edits in)', () => {
		const live: EnumAliasMap = { transactionType: { sale: 'For Sale' } };
		const fmt = makeEnumFormatter(live);
		expect(fmt.format('sale', { case: 'default' }, 'transactionType')).toBe('For Sale');
		live.transactionType.sale = 'Buy'; // marketing edits the alias
		expect(fmt.format('sale', { case: 'default' }, 'transactionType')).toBe('Buy');
	});
});
