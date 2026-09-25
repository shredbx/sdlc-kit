// Generic number formatter (S-FORMAT inc3). A pure Intl.NumberFormat wrapper consumed by
// the Media Canvas `number` formatter descriptor so a canvas label and any product card
// render numeric fields identically. Non-finite → '' (nothing to show).

/** Display options for {@link formatNumber}. */
export interface NumberFormatOptions {
	/** 'full' = standard grouped (1,234) · 'short' = compact notation (1.2K). Default 'full'. */
	notation?: 'full' | 'short';
	/** 'auto' = smart trailing zeros removed · 'off' = whole integer. Default 'auto'. */
	decimals?: 'auto' | 'off';
	/** 'on' = thousands separator (1,234) · 'off' = no grouping (1234). Default 'on'. */
	grouping?: 'on' | 'off';
}

/**
 * Format a raw number for display. Locale: 'en-US' (consistent across environments —
 * locale-switching is a knob the caller controls via its own `locale` option if needed).
 * Non-finite → '' (NaN, ±Infinity never produce a useful display string).
 */
export function formatNumber(value: number, opts: NumberFormatOptions = {}): string {
	if (!Number.isFinite(value)) return '';
	const { notation = 'full', decimals = 'auto', grouping = 'on' } = opts;
	return value.toLocaleString('en-US', {
		notation: notation === 'short' ? 'compact' : 'standard',
		minimumFractionDigits: 0,
		maximumFractionDigits: decimals === 'off' ? 0 : notation === 'short' ? 1 : 6,
		useGrouping: grouping === 'on'
	});
}
