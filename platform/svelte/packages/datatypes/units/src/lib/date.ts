// ISO date / datetime formatter (S-FORMAT inc3). A pure Intl.DateTimeFormat wrapper
// consumed by the Media Canvas `date` formatter descriptor. Input: ISO 8601 string
// (RFC3339 from the API layer). Invalid / empty → '' (nothing to show).

/** Display options for {@link formatDate}. */
export interface DateFormatOptions {
	/** Intl DateTimeStyle shorthand — controls how much detail is shown.
	 *  'short' = '6/14/2026' · 'medium' = 'Jun 14, 2026' · 'long' = 'June 14, 2026'
	 *  · 'full' = 'Sunday, June 14, 2026'. Default 'medium'. */
	style?: 'short' | 'medium' | 'long' | 'full';
	/** BCP 47 locale tag. Default 'en-US'. */
	locale?: string;
}

/**
 * Format an ISO date string for display using {@link Intl.DateTimeFormat}.
 * The date is interpreted in UTC (no timezone offset applied) to avoid
 * day-boundary shifts when times are omitted or midnight-anchored.
 * Invalid / empty input → ''.
 */
export function formatDate(value: string, opts: DateFormatOptions = {}): string {
	if (!value) return '';
	const { style = 'medium', locale = 'en-US' } = opts;
	const d = new Date(value);
	if (isNaN(d.getTime())) return '';
	return new Intl.DateTimeFormat(locale, { dateStyle: style, timeZone: 'UTC' }).format(d);
}
