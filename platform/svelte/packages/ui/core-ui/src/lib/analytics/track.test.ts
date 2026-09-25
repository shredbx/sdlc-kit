import { describe, it, expect } from 'vitest';
import { extractUtm } from './track';

// SE3 (client half): the beacon harvests ONLY the five-key UTM allowlist from the
// landing URL — never arbitrary query data — and attaches nothing when none are present.
describe('extractUtm', () => {
	it('lifts only the allowlisted utm_* keys', () => {
		expect(extractUtm('?utm_source=google&utm_medium=cpc&utm_campaign=spring')).toEqual({
			utm_source: 'google',
			utm_medium: 'cpc',
			utm_campaign: 'spring'
		});
	});

	it('includes utm_term and utm_content', () => {
		expect(extractUtm('?utm_term=villa&utm_content=hero')).toEqual({
			utm_term: 'villa',
			utm_content: 'hero'
		});
	});

	it('ignores non-allowlisted query params (no arbitrary data leaks)', () => {
		expect(extractUtm('?utm_source=fb&ref=abc&token=secret&page=2')).toEqual({
			utm_source: 'fb'
		});
	});

	it('returns undefined when no utm key is present', () => {
		expect(extractUtm('?ref=abc&page=2')).toBeUndefined();
		expect(extractUtm('')).toBeUndefined();
	});

	it('skips empty utm values', () => {
		expect(extractUtm('?utm_source=&utm_medium=cpc')).toEqual({ utm_medium: 'cpc' });
	});
});
