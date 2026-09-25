/**
 * money.ts — pure formatting helpers behind MoneyInput.
 *
 * Storage contract: a string of plain digits in MAJOR units (whole THB, not
 * satang). Display adds thousands separators + currency suffix; the caption
 * renders an order-of-magnitude shorthand so a 15,000,000 entry can be
 * sanity-checked against "≈ 15M THB". Extracted from MoneyInput.svelte so the
 * rules are unit-testable in isolation.
 */

/**
 * Keep only digits. Money is whole units in this storage model, so decimal
 * points, commas, currency symbols and stray labels are all dropped — the
 * caller converts to satang (×100) on submit, not the input layer.
 */
export function stripToDigits(raw: string): string {
	return raw.replace(/[^\d]/g, '');
}

/** Group integer digits with thousands separators, preserving any decimal tail. */
export function withCommas(digits: string): string {
	if (!digits) return digits;
	const [intPart, ...rest] = digits.split('.');
	const intWithCommas = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
	return rest.length > 0 ? `${intWithCommas}.${rest.join('.')}` : intWithCommas;
}

/** Blur-state display: "15,000,000 THB" or "45,000 THB / mo". Empty stays empty. */
export function formatMoneyDisplay(value: string, currency: string, suffix: string): string {
	if (!value) return '';
	return `${withCommas(value)} ${currency}${suffix}`;
}

// One decimal place under 100 of the unit, none at/above — and trim a bare ".0".
function shorthand(amount: number): string {
	return amount >= 100 ? amount.toFixed(0) : amount.toFixed(1).replace(/\.0$/, '');
}

/**
 * Display-only shorthand for amounts >= 10K: "≈ 15M THB" / "≈ 45K THB".
 * Returns '' below 10K or for non-numeric input — too small to misread.
 */
export function moneyCaption(value: string, currency: string, suffix: string): string {
	if (!value) return '';
	const num = Number(value);
	if (!Number.isFinite(num) || num < 10_000) return '';
	if (num >= 1_000_000) {
		return `≈ ${shorthand(num / 1_000_000)}M ${currency}${suffix}`;
	}
	return `≈ ${shorthand(num / 1_000)}K ${currency}${suffix}`;
}

// --- Canonical price formatter (options-aware, minor-unit) -------------------
// The single reusable money renderer. The helpers above back MoneyInput (MAJOR
// units, digit strings); formatPrice is the DISPLAY side for stored prices in
// MINOR units (satang) + an ISO code. The Media Canvas `currency` formatter
// descriptor wraps it, and any label can call it directly — so the canvas and a
// property card render a price identically, collapsing the ad-hoc THB Intl
// variants (formatManagePrice / canvas currency / etc.) onto one function.

const PRICE_LOCALE = 'en-US';

/** Display options for {@link formatPrice}. */
export interface PriceFormatOptions {
	/** 'symbol' → ฿2,500,000 · 'code' → 2,500,000 THB · 'none' → 2,500,000. Default 'symbol'. */
	display?: 'symbol' | 'code' | 'none';
	/** 'full' → grouped digits · 'short' → compact (฿2.5M). Default 'full'. */
	notation?: 'full' | 'short';
	/** 'off' → whole units · 'on' → the currency's minor-unit decimals (THB→2). Default 'off'. */
	decimals?: 'off' | 'on';
}

/** A currency's minor-unit exponent (THB/USD → 2, JPY → 0), from Intl; unknown → 2. */
function minorUnitExponent(currency: string): number {
	try {
		return (
			new Intl.NumberFormat(PRICE_LOCALE, { style: 'currency', currency }).resolvedOptions()
				.maximumFractionDigits ?? 2
		);
	} catch {
		return 2;
	}
}

/**
 * Format a stored price (MINOR units, e.g. satang) + ISO currency code for
 * DISPLAY, options-aware. The minor→major scale comes from the currency's own
 * exponent (THB → ÷100, JPY → ÷1), so the caller never hardcodes /100.
 * `display`: 'symbol' (฿2,500,000) · 'code' ({value} {code} → 2,500,000 THB) ·
 * 'none' (2,500,000). `notation`: 'full' | 'short' (฿2.5M). `decimals`: 'off'
 * (whole) | 'on' (the currency fraction). Non-finite → '' (no value to show).
 * Business fallbacks like POA for ≤0 stay with the caller — this stays generic.
 */
export function formatPrice(minor: number, currency: string, opts: PriceFormatOptions = {}): string {
	if (!Number.isFinite(minor)) return '';
	const { display = 'symbol', notation = 'full', decimals = 'off' } = opts;
	const cur = currency && currency.length === 3 ? currency.toUpperCase() : 'THB';
	const exp = minorUnitExponent(cur);
	const major = minor / Math.pow(10, exp);
	const compact = notation === 'short';
	const fracDigits = decimals === 'on' ? exp : 0;
	const num = (withSymbol: boolean): string =>
		new Intl.NumberFormat(PRICE_LOCALE, {
			...(withSymbol ? { style: 'currency', currency: cur, currencyDisplay: 'narrowSymbol' } : {}),
			notation: compact ? 'compact' : 'standard',
			minimumFractionDigits: compact ? 0 : fracDigits,
			maximumFractionDigits: compact ? 1 : fracDigits
		}).format(major);
	if (display === 'symbol') return num(true);
	if (display === 'code') return `${num(false)} ${cur}`;
	return num(false); // 'none'
}
