/**
 * Shared types for the visitor-activity visualization primitives.
 *
 * These primitives are token-driven and dependency-free (hand-rolled inline
 * SVG + CSS). They render small analytics datasets (≤90 day points, a short
 * device split, a ranked referer list) and are reused across projects — a
 * consumer maps its brand tokens onto the neutral `--va-*` custom properties
 * (see each component's `<style>` header) and never touches the markup.
 */

/** One day's view counts for a trend series. */
export interface TrendPoint {
	/** UTC day — ISO date string or Date; rendered as a short label. */
	day: string | Date;
	/** Total hits that day ("Views"). */
	views: number;
	/** Distinct daily visitors that day ("Daily visits"). */
	visits: number;
}

/** One labelled magnitude — used by DeviceBars and BarList. */
export interface LabeledValue {
	/** Row label. An empty string is rendered as "Direct" by BarList. */
	label: string;
	/** Magnitude driving the bar width and the displayed count. */
	value: number;
}

/**
 * One bucket of a generic visitor-activity breakdown — the wire shape the
 * `pkg/visitoractivity` store emits for every dimension (country, language, os,
 * browser, channel, hour, weekday, referer, utm_*). Mirrors the Go
 * `DimensionStat` JSON tags exactly (Decision D1 — the single result shape that
 * superseded the old per-dimension by_referer/by_device structs).
 */
export interface DimensionStat {
	/** The grouped value, e.g. a country code, an OS family, an hour. */
	label: string;
	/** Distinct daily visitors in this bucket (COUNT DISTINCT visitor_id). */
	unique_views: number;
	/** Total hits in this bucket (COUNT(*)). */
	total_hits: number;
}
