// TDD — @sbx/text-template core (INC-C). Parse a flat template into text/token nodes, serialize
// back, resolve via a consumer TokenResolver, and list the distinct tokens. The user's headline
// example drives TC-TT-01: "Welcome to [property.address.country], explore [property.propertyType]".
import { describe, it, expect } from 'vitest';
import type { TokenResolver, Template } from './template.js';
import { parseTemplate, serializeTemplate, resolveTemplate, templateTokens } from './template.js';

describe('parseTemplate', () => {
	it('TC-TT-01 splits text + [token] nodes in order (the headline example)', () => {
		expect(
			parseTemplate('Welcome to [property.address.country], explore [property.propertyType]')
		).toEqual([
			{ type: 'text', text: 'Welcome to ' },
			{ type: 'token', token: 'property.address.country' },
			{ type: 'text', text: ', explore ' },
			{ type: 'token', token: 'property.propertyType' }
		]);
	});

	it('TC-TT-02 handles a token at the very start and end', () => {
		expect(parseTemplate('[a.b] middle [c.d]')).toEqual([
			{ type: 'token', token: 'a.b' },
			{ type: 'text', text: ' middle ' },
			{ type: 'token', token: 'c.d' }
		]);
	});

	it('TC-TT-03 plain text with no tokens → a single text node', () => {
		expect(parseTemplate('just words')).toEqual([{ type: 'text', text: 'just words' }]);
	});

	it('TC-TT-04 empty source → []', () => {
		expect(parseTemplate('')).toEqual([]);
	});

	it('TC-TT-05 a bracket run that is not a [path] stays literal text', () => {
		// 'not a token' has spaces → not a dotted path → no token, the prose survives intact.
		expect(parseTemplate('a [not a token] b')).toEqual([{ type: 'text', text: 'a [not a token] b' }]);
	});
});

describe('serializeTemplate', () => {
	it('TC-TT-06 is the inverse of parse for the text+token structure', () => {
		const src = 'Price is [property.price] today';
		expect(serializeTemplate(parseTemplate(src))).toBe(src);
	});
});

describe('resolveTemplate', () => {
	const resolver: TokenResolver = {
		resolve: (token, format) => {
			if (format === 'enum:x') return 'ENUM';
			const map: Record<string, string> = {
				'property.price': '฿7,000,000',
				'property.address.country': 'Thailand'
			};
			return map[token] ?? '';
		}
	};

	it('TC-TT-07 substitutes tokens via the resolver and keeps the literal text', () => {
		expect(
			resolveTemplate(parseTemplate('Buy in [property.address.country] for [property.price]'), resolver)
		).toBe('Buy in Thailand for ฿7,000,000');
	});

	it('TC-TT-08 a missing token resolves to empty (graceful — never "undefined")', () => {
		expect(resolveTemplate(parseTemplate('[property.unknown] left'), resolver)).toBe(' left');
	});

	it('TC-TT-09 passes a token node format + formatOptions through to the resolver', () => {
		const nodes: Template = [
			{ type: 'token', token: 'x', format: 'enum:x', formatOptions: { case: 'upper' } }
		];
		expect(resolveTemplate(nodes, resolver)).toBe('ENUM');
	});
});

describe('templateTokens', () => {
	it('TC-TT-10 lists the DISTINCT token paths in first-seen order', () => {
		expect(templateTokens(parseTemplate('[a.b] [c.d] [a.b]'))).toEqual(['a.b', 'c.d']);
	});
});
