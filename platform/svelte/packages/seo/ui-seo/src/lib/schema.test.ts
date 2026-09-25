// @sbx/ui-seo · schema builders — unit tests. Lock the JSON-LD shape the collection /
// site-identity GEO levers emit, and the XSS-safe serialization contract.

import { describe, it, expect } from 'vitest';
import {
	serializeJsonLd,
	buildItemListSchema,
	buildBreadcrumbListSchema,
	buildFAQPageSchema,
	buildOrganizationSchema
} from './schema';

describe('buildItemListSchema', () => {
	it('assigns 1-based positions in array (display) order', () => {
		const node = buildItemListSchema({
			items: [{ url: 'https://x/a' }, { url: 'https://x/b' }, { url: 'https://x/c' }]
		});
		expect(node['@type']).toBe('ItemList');
		const els = node.itemListElement as Array<Record<string, unknown>>;
		expect(els.map((e) => e.position)).toEqual([1, 2, 3]);
		expect(els.map((e) => e.url)).toEqual(['https://x/a', 'https://x/b', 'https://x/c']);
		expect(node.numberOfItems).toBe(3);
	});

	it('carries name/image only when given, and drops absent members', () => {
		const node = buildItemListSchema({
			name: 'Properties for sale',
			url: 'https://x/properties',
			items: [{ url: 'https://x/a', name: 'Villa A', image: 'https://x/a.jpg' }, { url: 'https://x/b' }]
		});
		expect(node.name).toBe('Properties for sale');
		const els = node.itemListElement as Array<Record<string, unknown>>;
		expect(els[0]).toEqual({ '@type': 'ListItem', position: 1, url: 'https://x/a', name: 'Villa A', image: 'https://x/a.jpg' });
		// second item has no name/image keys at all
		expect(els[1]).toEqual({ '@type': 'ListItem', position: 2, url: 'https://x/b' });
	});

	it('an empty list is a valid minimal node (no itemListElement / numberOfItems)', () => {
		const node = buildItemListSchema({ items: [] });
		expect(node['@type']).toBe('ItemList');
		expect(node).not.toHaveProperty('itemListElement');
		expect(node).not.toHaveProperty('numberOfItems');
	});
});

describe('buildBreadcrumbListSchema', () => {
	it('emits ordered ListItems with item=url, omitting item on the current page', () => {
		const node = buildBreadcrumbListSchema([
			{ name: 'Home', url: 'https://x/' },
			{ name: 'Properties', url: 'https://x/properties' },
			{ name: 'Villa A' } // current page — no url
		]);
		expect(node['@type']).toBe('BreadcrumbList');
		const els = node.itemListElement as Array<Record<string, unknown>>;
		expect(els[0]).toEqual({ '@type': 'ListItem', position: 1, name: 'Home', item: 'https://x/' });
		expect(els[2]).toEqual({ '@type': 'ListItem', position: 3, name: 'Villa A' });
	});
});

describe('buildOrganizationSchema', () => {
	it('defaults @type to Organization and mirrors logo into image', () => {
		const node = buildOrganizationSchema({ name: 'BR', url: 'https://x/', logo: 'https://x/logo.png' });
		expect(node['@type']).toBe('Organization');
		expect(node.logo).toBe('https://x/logo.png');
		expect(node.image).toBe('https://x/logo.png');
	});

	it('accepts a RealEstateAgent type and nests a PostalAddress', () => {
		const node = buildOrganizationSchema({
			type: 'RealEstateAgent',
			name: 'Bestie Real Estate',
			url: 'https://x/',
			telephone: '+66 0 000 0000',
			sameAs: ['https://facebook.com/br'],
			address: { addressLocality: 'Phuket', addressCountry: 'TH' },
			areaServed: 'Phuket, Thailand'
		});
		expect(node['@type']).toBe('RealEstateAgent');
		expect(node.address).toEqual({ '@type': 'PostalAddress', addressLocality: 'Phuket', addressCountry: 'TH' });
		expect(node.sameAs).toEqual(['https://facebook.com/br']);
		expect(node.areaServed).toBe('Phuket, Thailand');
		expect(node).not.toHaveProperty('email');
	});
});

describe('serializeJsonLd', () => {
	it('escapes every "<" so user-derived text cannot break out of the script tag', () => {
		const node = buildItemListSchema({ items: [{ url: 'https://x/a', name: '</script><b>hi' }] });
		const out = serializeJsonLd(node);
		expect(out).not.toContain('</script>');
		expect(out).not.toContain('<b>');
		expect(out).toContain('\\u003c');
	});

	it('serializes an array of nodes (ItemList + BreadcrumbList on one page)', () => {
		const out = serializeJsonLd([
			buildItemListSchema({ items: [{ url: 'https://x/a' }] }),
			buildBreadcrumbListSchema([{ name: 'Home', url: 'https://x/' }])
		]);
		const parsed = JSON.parse(out.replace(/\\u003c/g, '<')) as Array<Record<string, unknown>>;
		expect(parsed).toHaveLength(2);
		expect(parsed[0]['@type']).toBe('ItemList');
		expect(parsed[1]['@type']).toBe('BreadcrumbList');
	});
});

describe('buildFAQPageSchema', () => {
	it('maps items to Question/acceptedAnswer pairs in order', () => {
		const node = buildFAQPageSchema({
			url: 'https://example.com/faq',
			items: [
				{ question: 'How do I buy?', answer: 'Start with a budget.' },
				{ question: 'What is chanote?', answer: 'The strongest title deed.' }
			]
		}) as Record<string, unknown>;
		expect(node['@type']).toBe('FAQPage');
		expect(node.url).toBe('https://example.com/faq');
		const main = node.mainEntity as Array<Record<string, unknown>>;
		expect(main).toHaveLength(2);
		expect(main[0]['@type']).toBe('Question');
		expect(main[0].name).toBe('How do I buy?');
		expect((main[0].acceptedAnswer as Record<string, unknown>).text).toBe('Start with a budget.');
	});

	it('drops entries missing a question or answer (schema.org requires both)', () => {
		const node = buildFAQPageSchema({
			items: [
				{ question: 'Kept?', answer: 'Yes.' },
				{ question: '', answer: 'orphan answer' },
				{ question: 'orphan question', answer: '   ' }
			]
		}) as Record<string, unknown>;
		expect(node.mainEntity as unknown[]).toHaveLength(1);
	});
});
