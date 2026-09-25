// @sbx/ui-seo · schema — reusable schema.org (JSON-LD) builders.
//
// The per-entity structured-data mapping lives ONCE here so every consumer reuses the
// SAME RealEstateListing / Article graph (NOT hand-rolled per page). A consumer passes
// the typed fields it already has; the builder returns a JsonLd for <SeoHead jsonLd={...}>.
// schema.org is the #1 per-entity GEO (generative-engine-optimization) lever — clean,
// machine-readable facts that answer engines can lift directly.

import type { JsonLd } from './types';

/** Serialize a JsonLd node (or list) for an inline `<script type="application/ld+json">`.
 *  Every `<` is rewritten to its backslash-u003c unicode escape, so an embedded `</script>` or
 *  markup in user-derived text (title/description) can never break out of the script
 *  element (XSS-safe). */
export function serializeJsonLd(node: JsonLd | JsonLd[]): string {
	return JSON.stringify(node).replace(/</g, '\\u003c');
}

/** Drop undefined / null / empty members so the emitted graph carries only present
 *  fields — no `"price": null` noise, keeping the structured data lean for crawlers. */
function compact<T extends Record<string, unknown>>(obj: T): T {
	const out: Record<string, unknown> = {};
	for (const [key, value] of Object.entries(obj)) {
		if (value === undefined || value === null || value === '') continue;
		if (Array.isArray(value) && value.length === 0) continue;
		out[key] = value;
	}
	return out as T;
}

// ── Real-estate listing ─────────────────────────────────────────────────────────

export interface PostalAddressInput {
	streetAddress?: string;
	addressLocality?: string;
	addressRegion?: string;
	postalCode?: string;
	/** ISO 3166-1 alpha-2 country code, e.g. 'TH'. */
	addressCountry?: string;
}

export interface RealEstateListingInput {
	/** Canonical absolute URL of the listing page. */
	url: string;
	/** Listing / property title. */
	name: string;
	description: string;
	/** Absolute image URL(s). */
	image?: string | string[];
	/** Offer price in MAJOR currency units (e.g. THB, not satang). */
	price?: number;
	/** ISO 4217 currency code, e.g. 'THB'. */
	priceCurrency?: string;
	/** BR offering intent → GoodRelations businessFunction (Sell / LeaseOut). */
	offering?: 'for_sale' | 'for_lease';
	/** ISO 8601 date the listing was posted. */
	datePosted?: string;
	numberOfBedrooms?: number;
	numberOfBathrooms?: number;
	/** Floor area in square metres. */
	floorSizeSqm?: number;
	address?: PostalAddressInput;
	geo?: { latitude: number; longitude: number };
}

const BUSINESS_FUNCTION: Record<NonNullable<RealEstateListingInput['offering']>, string> = {
	for_sale: 'http://purl.org/goodrelations/v1#Sell',
	for_lease: 'http://purl.org/goodrelations/v1#LeaseOut'
};

/**
 * Build a schema.org `RealEstateListing` graph — the listing WebPage carrying a nested
 * `Residence` (physical attributes) and an `Offer` (price + availability). Only the
 * fields you pass are emitted; absent members are dropped.
 */
export function buildRealEstateListingSchema(input: RealEstateListingInput): JsonLd {
	const residence = compact({
		'@type': 'Residence',
		name: input.name,
		numberOfBedrooms: input.numberOfBedrooms,
		numberOfBathroomsTotal: input.numberOfBathrooms,
		floorSize:
			input.floorSizeSqm === undefined
				? undefined
				: { '@type': 'QuantitativeValue', value: input.floorSizeSqm, unitCode: 'MTK' },
		address:
			input.address === undefined
				? undefined
				: compact({ '@type': 'PostalAddress', ...input.address }),
		geo:
			input.geo === undefined
				? undefined
				: {
						'@type': 'GeoCoordinates',
						latitude: input.geo.latitude,
						longitude: input.geo.longitude
					}
	});

	const offer =
		input.price === undefined
			? undefined
			: compact({
					'@type': 'Offer',
					price: input.price,
					priceCurrency: input.priceCurrency,
					availability: 'https://schema.org/InStock',
					businessFunction: input.offering ? BUSINESS_FUNCTION[input.offering] : undefined
				});

	return compact({
		'@context': 'https://schema.org',
		'@type': 'RealEstateListing',
		url: input.url,
		name: input.name,
		description: input.description,
		image: input.image,
		datePosted: input.datePosted,
		// Only nest the residence when it carries an attribute beyond its @type.
		mainEntity: Object.keys(residence).length > 1 ? residence : undefined,
		offers: offer
	}) as JsonLd;
}

// ── Article (content / blog / guide pages) ───────────────────────────────────────

export interface ArticleInput {
	/** Canonical absolute URL of the article page. */
	url: string;
	headline: string;
	description?: string;
	image?: string | string[];
	datePublished?: string;
	dateModified?: string;
	authorName?: string;
	authorType?: 'Person' | 'Organization';
}

/** Build a schema.org `Article` graph for content / blog / guide pages. */
export function buildArticleSchema(input: ArticleInput): JsonLd {
	return compact({
		'@context': 'https://schema.org',
		'@type': 'Article',
		mainEntityOfPage: input.url,
		headline: input.headline,
		description: input.description,
		image: input.image,
		datePublished: input.datePublished,
		dateModified: input.dateModified,
		author:
			input.authorName === undefined
				? undefined
				: { '@type': input.authorType ?? 'Organization', name: input.authorName }
	}) as JsonLd;
}

// ── ItemList (listing / index / search-result pages) ──────────────────────────────
// The collection-level GEO lever: a listing page emits ONE ItemList describing the items
// it shows, in display order — answer engines + Google read the set & ordering off the
// list page, while each detail page stays the canonical home for the full record. We emit
// the lean "summary" form (position + url + name) so the list never contradicts the
// detail graph; the consumer builds the array off its already-loaded, already-ordered data.

export interface ItemListEntryInput {
	/** Canonical absolute URL of the item's own detail page. */
	url: string;
	/** Item title — names the result for answer engines. */
	name?: string;
	/** Absolute image URL (optional thumbnail). */
	image?: string;
}

export interface ItemListInput {
	/** Items in DISPLAY order — `position` is assigned 1-based from array order, so the
	 *  graph mirrors what the page actually shows for the active filter/sort state. */
	items: ItemListEntryInput[];
	/** Canonical absolute URL of the list page itself (optional). */
	url?: string;
	/** Human name for the collection, e.g. 'Properties for sale' (optional). */
	name?: string;
}

/**
 * Build a schema.org `ItemList` graph for a listing / index / search-result page. Each
 * entry becomes a `ListItem` (position + url, plus name/image when given). An empty list
 * yields a valid minimal node (no `itemListElement`). Absent members are dropped.
 */
export function buildItemListSchema(input: ItemListInput): JsonLd {
	const itemListElement = input.items.map((it, i) =>
		compact({ '@type': 'ListItem', position: i + 1, url: it.url, name: it.name, image: it.image })
	);
	return compact({
		'@context': 'https://schema.org',
		'@type': 'ItemList',
		name: input.name,
		url: input.url,
		numberOfItems: itemListElement.length || undefined,
		itemListElement
	}) as JsonLd;
}

// ── FAQPage (question/answer rich results) ────────────────────────────────────────

export interface FaqPageEntryInput {
	/** The question text. */
	question: string;
	/** The accepted answer (plain text or HTML — Google strips markup). */
	answer: string;
}

export interface FaqPageInput {
	/** Absolute URL of the FAQ page. */
	url?: string;
	/** Published Q/A pairs in display order. */
	items: FaqPageEntryInput[];
}

/**
 * Build a schema.org `FAQPage` graph (Google FAQ rich results). Each entry becomes a
 * `Question` with one `acceptedAnswer`. Entries missing a question or answer are dropped
 * — schema.org requires both.
 */
export function buildFAQPageSchema(input: FaqPageInput): JsonLd {
	const mainEntity = input.items
		.filter((it) => it.question.trim() !== '' && it.answer.trim() !== '')
		.map((it) => ({
			'@type': 'Question',
			name: it.question,
			acceptedAnswer: { '@type': 'Answer', text: it.answer }
		}));
	return compact({
		'@context': 'https://schema.org',
		'@type': 'FAQPage',
		url: input.url,
		mainEntity
	}) as JsonLd;
}

// ── BreadcrumbList (navigation context) ───────────────────────────────────────────

export interface BreadcrumbCrumbInput {
	/** Crumb label. */
	name: string;
	/** Absolute URL of the crumb. Omit for the current (last) page per schema.org. */
	url?: string;
}

/**
 * Build a schema.org `BreadcrumbList` from an ordered trail. `position` is 1-based; the
 * crumb URL is emitted as `item` (omit `url` on the current page). Absent members dropped.
 */
export function buildBreadcrumbListSchema(crumbs: BreadcrumbCrumbInput[]): JsonLd {
	const itemListElement = crumbs.map((c, i) =>
		compact({ '@type': 'ListItem', position: i + 1, name: c.name, item: c.url })
	);
	return compact({
		'@context': 'https://schema.org',
		'@type': 'BreadcrumbList',
		itemListElement
	}) as JsonLd;
}

// ── Organization / LocalBusiness (site identity) ──────────────────────────────────

export interface OrganizationInput {
	/** Business / brand name. */
	name: string;
	/** Canonical absolute site URL (homepage). */
	url: string;
	/** One-line description of the business (helps answer engines summarise it). */
	description?: string;
	/** Absolute logo image URL. */
	logo?: string;
	/** Absolute photo of the business/premises (schema.org `image`). Distinct from the
	 *  brand `logo`; falls back to `logo` when no dedicated photo is supplied. */
	image?: string;
	/** schema.org `@type` — defaults to 'Organization'; BR passes 'RealEstateAgent'
	 *  (a `LocalBusiness` subtype). */
	type?: string;
	/** Public contact telephone. */
	telephone?: string;
	/** Public contact email. */
	email?: string;
	/** Social / external profile URLs (schema.org `sameAs`). */
	sameAs?: string[];
	address?: PostalAddressInput;
	/** Geographic area served, e.g. 'Phuket, Thailand'. */
	areaServed?: string | string[];
}

/**
 * Build a schema.org `Organization` graph (BR passes `type: 'RealEstateAgent'`) — the
 * site-identity node emitted once on the home page: who the business is, where, how to
 * reach it. Absent members are dropped.
 */
export function buildOrganizationSchema(input: OrganizationInput): JsonLd {
	return compact({
		'@context': 'https://schema.org',
		'@type': input.type ?? 'Organization',
		name: input.name,
		url: input.url,
		description: input.description,
		logo: input.logo,
		image: input.image ?? input.logo,
		telephone: input.telephone,
		email: input.email,
		sameAs: input.sameAs,
		address:
			input.address === undefined
				? undefined
				: compact({ '@type': 'PostalAddress', ...input.address }),
		areaServed: input.areaServed
	}) as JsonLd;
}
