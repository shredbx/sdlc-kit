/**
 * Dimension display config — the generic, reusable map from a visitor-activity
 * breakdown KEY (the dimension keys the `pkg/visitoractivity` store emits in its
 * `breakdowns` map) to a human-readable section title.
 *
 * This is the single source of truth a consumer loops over to render breakdown
 * sections — adding a backend dimension surfaces in the UI with zero per-section
 * code (only a label here if a nicer title is wanted). The label set mirrors the
 * Go `dimensionExpr` registry keys (store.go).
 *
 * A consumer (BR now, BS later) can override or subset the labels by spreading
 * this default and reassigning entries — it never copies the map.
 */

/** A dimension key as emitted by the backend `breakdowns` map. */
export type DimensionKey =
	| 'referer'
	| 'device'
	| 'country'
	| 'language'
	| 'os'
	| 'browser'
	| 'channel'
	| 'hour'
	| 'weekday'
	| 'utm_source'
	| 'utm_medium'
	| 'utm_campaign';

/** Map a dimension key → its display title. */
export type DimensionLabels = Partial<Record<DimensionKey, string>>;

/**
 * The default human labels for every known dimension. Projects override by
 * spreading: `{ ...DEFAULT_DIMENSION_LABELS, referer: 'Where they came from' }`.
 */
export const DEFAULT_DIMENSION_LABELS: Record<DimensionKey, string> = {
	referer: 'Sources',
	device: 'Devices',
	country: 'Countries',
	language: 'Languages',
	os: 'Operating system',
	browser: 'Browser',
	channel: 'Traffic channel',
	hour: 'By hour',
	weekday: 'By weekday',
	utm_source: 'Campaign source',
	utm_medium: 'Campaign medium',
	utm_campaign: 'Campaign'
};
