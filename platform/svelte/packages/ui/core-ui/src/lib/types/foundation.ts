/**
 * Foundation Type Interfaces
 *
 * TypeScript counterparts of Go foundation types (pkg/money, pkg/image, pkg/dictionary, pkg/catalogue).
 * Used by @sbx/core-ui components for type-safe rendering across all SBX applications.
 *
 * Type → View Level mapping (from type.yml protocol):
 *   primitive  → primitives (Input, Display)
 *   money      → primitives + blocks (MoneyDisplay, MoneyInput)
 *   dictionary → blocks (DictBadge, DictSelect, DictFilter)
 *   catalogue  → sections (CatalogueSection)
 *   image      → blocks (ImageCard, ImageGrid, ImageCarousel)
 *   video      → sections (VideoPlayer)
 *   document   → layouts (EntityDetail, EntityList, EntityForm)
 */

// =============================================================================
// MONEY — Fowler's Money pattern (uint64 minor units, ISO 4217)
// =============================================================================

/** Monetary value stored in smallest currency unit (satang, cents) */
export interface MoneyValue {
	/** Amount in smallest unit (e.g., 1290000000 = ฿12,900,000.00) */
	amount: number;
	/** ISO 4217 currency code */
	currency: string;
	/** Decimal places for this currency (2 for THB/USD, 0 for JPY) */
	fraction: number;
}

// Currency registry \u2014 GENERATED single source of truth.
// CurrencyConfig + CURRENCIES are projected from currencies.yml via
// `sbx generate currencies` (see currencies.generated.ts). Re-exported here so
// existing `$lib/types` imports keep working unchanged.
import { CURRENCIES, type CurrencyConfig } from './currencies.generated';
export { CURRENCIES, type CurrencyConfig };

/** Format a MoneyValue for display (mirrors Go money.Display()) */
export function formatMoney(value: MoneyValue): string {
	const cur = CURRENCIES[value.currency] ?? {
		code: value.currency,
		symbol: value.currency,
		decimals: value.fraction,
		symbolPosition: 'prefix' as const
	};

	const floatVal = value.amount / Math.pow(10, cur.decimals);
	const formatted = floatVal.toLocaleString('en-US', {
		minimumFractionDigits: cur.decimals,
		maximumFractionDigits: cur.decimals
	});

	return cur.symbolPosition === 'suffix' ? `${formatted}${cur.symbol}` : `${cur.symbol}${formatted}`;
}

/** Parse a float amount to MoneyValue */
export function moneyFromFloat(amount: number, currency: string = 'THB'): MoneyValue {
	const cur = CURRENCIES[currency];
	const decimals = cur?.decimals ?? 2;
	return {
		amount: Math.round(amount * Math.pow(10, decimals)),
		currency,
		fraction: decimals
	};
}

// =============================================================================
// IMAGE — Binary asset with CDN URL
// =============================================================================

/** Image metadata (mirrors Go image.Image) */
export interface ImageData {
	id: string;
	key: string;
	url: string;
	width: number;
	height: number;
	format: string;
	size: number;
	alt_text: string;
	purpose: string;
	created_by: string;
	created_at: string;
	updated_at: string;
	version: number;
}

/** Image display variant */
export type ImageVariant = 'thumbnail' | 'card' | 'cover' | 'avatar' | 'full';

/** Image aspect ratios */
export type ImageAspectRatio = '1:1' | '4:3' | '16:9' | '3:2' | '2:3' | 'auto';

// =============================================================================
// DICTIONARY — Localized enum with rich content types
// =============================================================================

/** Dictionary entry (mirrors Go dictionary.Entry) */
export interface DictEntry {
	code: string;
	label: string;
	description?: string;
	sort_order: number;
	icon?: string;
	color?: string;
	image_url?: string;
	video_url?: string;
	deprecated: boolean;
}

/** Dictionary content type — what kind of value the entry represents */
export type DictContentType = 'text' | 'icon' | 'image' | 'video' | 'color';

/** Localized dictionary label */
export interface DictLocalizedLabel {
	locale: string;
	label: string;
	description?: string;
}

/** Full dictionary definition */
export interface DictEnum {
	name: string;
	purpose: string;
	entries: DictEntry[];
}

// =============================================================================
// CATALOGUE — Named document collection
// =============================================================================

/** Catalogue type */
export type CatalogueType = 'static' | 'dynamic' | 'hybrid';

/** Catalogue definition (mirrors Go catalogue.Catalogue) */
export interface CatalogueData {
	id: string;
	name: string;
	slug: string;
	entity_type: string;
	type: CatalogueType;
	description?: string;
	filter?: Record<string, unknown>;
	include_ids?: string[];
	exclude_ids?: string[];
	metadata?: Record<string, unknown>;
	visible: boolean;
	created_at: string;
	updated_at: string;
}

/** Catalogue item reference */
export interface CatalogueItem {
	catalogue_id: string;
	document_id: string;
	sort_order: number;
	added_at: string;
	added_by?: string;
}

// =============================================================================
// DOCUMENT — Identity-based entity with typed fields
// =============================================================================

/** Foundation type categories (from type.yml protocol) */
export type FoundationType = 'primitive' | 'document' | 'dictionary' | 'catalogue' | 'money' | 'image' | 'video';

/** Field definition for type-aware rendering */
export interface FieldDefinition {
	name: string;
	type: FoundationType | 'string' | 'number' | 'boolean' | 'date' | 'datetime' | 'uuid';
	label?: string;
	purpose?: string;
	required?: boolean;
	dictionary_ref?: string;
}

/** Generic document data (any document-typed entity) */
export interface DocumentData {
	id: string;
	[key: string]: unknown;
}
