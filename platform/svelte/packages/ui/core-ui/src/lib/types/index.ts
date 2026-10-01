/**
 * Types — Foundation Type System
 *
 * Shared TypeScript interfaces for all SBX foundation types.
 */

export type {
	MoneyValue,
	CurrencyConfig,
	ImageData,
	ImageVariant,
	ImageAspectRatio,
	DictEntry,
	DictContentType,
	DictLocalizedLabel,
	DictEnum,
	CatalogueType,
	CatalogueData,
	CatalogueItem,
	FoundationType,
	FieldDefinition,
	DocumentData
} from './foundation';

export { CURRENCIES, formatMoney, moneyFromFloat } from './foundation';

// Admin types (shared control panel interfaces)
export type { AdminUser, AdminSection, AdminConfig } from './admin';
