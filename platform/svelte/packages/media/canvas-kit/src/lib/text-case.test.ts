// Direct contract for the shared capitalizeTokens primitive (S-FORMAT). Transitively covered
// by format-labels + enum-formatter, pinned here so the two modes can't silently drift.
//   TC-TC-01  default (preserve case) — capitalize first of each token, keep separators + acronyms
//   TC-TC-02  lowerRest — proper Title Case (lowercases the rest)
//   TC-TC-03  empty string → '' (never throws)
import { describe, it, expect } from 'vitest';
import { capitalizeTokens } from './text-case.js';

describe('capitalizeTokens', () => {
	it('TC-TC-01 default preserves separators + existing case (gentle capitalize)', () => {
		expect(capitalizeTokens('symbol')).toBe('Symbol');
		expect(capitalizeTokens('freehold-leasehold')).toBe('Freehold-Leasehold');
		expect(capitalizeTokens('POA')).toBe('POA'); // acronym survives — no lowercasing
	});

	it('TC-TC-02 lowerRest gives proper Title Case', () => {
		expect(capitalizeTokens('FOR SALE', { lowerRest: true })).toBe('For Sale');
		expect(capitalizeTokens('pool villa', { lowerRest: true })).toBe('Pool Villa');
	});

	it('TC-TC-03 empty string passes through', () => {
		expect(capitalizeTokens('')).toBe('');
		expect(capitalizeTokens('', { lowerRest: true })).toBe('');
	});
});
