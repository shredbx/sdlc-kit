/**
 * Viz — visitor-activity visualization primitives.
 *
 * Token-driven, dependency-free (hand-rolled inline SVG + CSS) widgets for
 * small analytics datasets. A consumer maps its brand tokens onto the neutral
 * `--va-*` custom properties documented in each component and never copies the
 * markup. Reused across projects (BR now, BS later).
 *
 * - StatCard         — headline metric (icon + value + label)
 * - TrendChart       — bars (views/day) + overlaid line (daily visits) + tooltip
 * - DeviceBars       — horizontal split with count + %
 * - BarList          — ranked rows with an inline proportion bar
 * - BreakdownSection — a titled dimension section wrapping BarList (generic)
 *
 * Plus the default dimension display config, which a consumer loops over to
 * render breakdown sections from the backend `breakdowns` map.
 */

export { default as StatCard } from './StatCard.svelte';
export { default as TrendChart } from './TrendChart.svelte';
export { default as DeviceBars } from './DeviceBars.svelte';
export { default as BarList } from './BarList.svelte';
export { default as BreakdownSection } from './BreakdownSection.svelte';

export { DEFAULT_DIMENSION_LABELS } from './dimensions';

export type { TrendPoint, LabeledValue, DimensionStat } from './types';
export type { DimensionKey, DimensionLabels } from './dimensions';
