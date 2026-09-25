// Land-area unit conversions. Canonical storage is sqm; every other unit
// is defined by its sqm equivalence factor. Source values:
//   1 sqft = 0.09290304 sqm (exact, per international yard agreement)
//   1 wah  = 4 sqm           (1 wah² = (2m)²)
//   1 ngan = 400 sqm         (= 100 wah²)
//   1 rai  = 1600 sqm        (= 4 ngan)

export type LandSizeUnit = 'sqm' | 'sqft' | 'rai' | 'ngan' | 'wah';

export interface LandSizeUnitMeta {
	id: LandSizeUnit;
	display: string;
	thaiDisplay: string;
	sqmFactor: number;
}

export const LAND_SIZE_UNITS: Record<LandSizeUnit, LandSizeUnitMeta> = {
	sqm: { id: 'sqm', display: 'sqm', thaiDisplay: 'ตร.ม.', sqmFactor: 1.0 },
	sqft: { id: 'sqft', display: 'sqft', thaiDisplay: 'ตร.ฟ.', sqmFactor: 0.09290304 },
	rai: { id: 'rai', display: 'rai', thaiDisplay: 'ไร่', sqmFactor: 1600.0 },
	ngan: { id: 'ngan', display: 'ngan', thaiDisplay: 'งาน', sqmFactor: 400.0 },
	wah: { id: 'wah', display: 'wah', thaiDisplay: 'วา²', sqmFactor: 4.0 }
};

export const LAND_SIZE_UNITS_ORDERED: LandSizeUnit[] = ['sqm', 'rai', 'ngan', 'wah', 'sqft'];

export function isLandSizeUnit(value: unknown): value is LandSizeUnit {
	return typeof value === 'string' && value in LAND_SIZE_UNITS;
}

export function toSqm(value: number, unit: LandSizeUnit): number {
	if (!Number.isFinite(value)) return Number.NaN;
	return value * LAND_SIZE_UNITS[unit].sqmFactor;
}

export function fromSqm(sqm: number, unit: LandSizeUnit): number {
	if (!Number.isFinite(sqm)) return Number.NaN;
	return sqm / LAND_SIZE_UNITS[unit].sqmFactor;
}

export function convert(value: number, from: LandSizeUnit, to: LandSizeUnit): number {
	if (from === to) return value;
	return fromSqm(toSqm(value, from), to);
}

// CONTRACT — this is the SINGLE SOURCE OF DISPLAY PRECISION for the area
// dimension. sqm/sqft are integer-ish (1 sqm matters), rai is large so 3
// decimals is meaningful, wah sits in between. Every consumer (SizeInput,
// formatLandSize, formatArea, …) MUST defer to this for display rounding and
// NEVER re-round looser or tighter on its own — doing so re-introduces the
// floating-point tails this contract exists to prevent (e.g. 3342.99997).
// Cross-DIMENSION conversion (area↔length) never flows through unit convert;
// it only happens via the explicit geometry bridges in geo-area.ts (Decision
// on measurement-dimension separation).
export function decimalPlacesFor(unit: LandSizeUnit): number {
	switch (unit) {
		case 'sqm':
			return 0;
		case 'sqft':
			return 0;
		case 'wah':
			return 2;
		case 'ngan':
			return 3;
		case 'rai':
			return 3;
	}
}

export function formatLandSize(value: number, unit: LandSizeUnit): string {
	if (!Number.isFinite(value)) return '';
	const dp = decimalPlacesFor(unit);
	return value.toLocaleString(undefined, {
		minimumFractionDigits: 0,
		maximumFractionDigits: dp
	});
}

/**
 * Plain, ungrouped editable string of a land size at the unit's natural
 * precision — the raw sibling of {@link formatLandSize} (which groups digits for
 * read-only display). The SINGLE rounding path for editable size inputs: SizeInput
 * defers to this so {@link decimalPlacesFor}'s precision contract is enforced in
 * ONE tested place, never re-implemented per consumer. Trims trailing FRACTIONAL
 * zeros only — guards on a decimal point so a whole number stays whole ('3300',
 * never gutted to '33') while 0.19400 → '0.194'. Non-finite → '' (no value).
 */
export function formatLandSizePlain(value: number, unit: LandSizeUnit): string {
	if (!Number.isFinite(value)) return '';
	const fixed = value.toFixed(decimalPlacesFor(unit));
	return fixed.includes('.') ? fixed.replace(/\.?0+$/, '') : fixed;
}

/** Display options for {@link formatArea}. */
export interface AreaFormatOptions {
	/** Target display unit (the stored value is always sqm). */
	unit: LandSizeUnit;
	/** 'full' = grouped digits (8,000) · 'short' = compact (8K). Default 'full'. */
	notation?: 'full' | 'short';
	/** 'auto' = unit-aware precision ({@link decimalPlacesFor}) · 'off' = whole. Default 'auto'. */
	decimals?: 'auto' | 'off';
}

/** Human unit label — 'm²' for sqm, the unit's `display` otherwise. */
function areaUnitLabel(unit: LandSizeUnit): string {
	return unit === 'sqm' ? 'm²' : LAND_SIZE_UNITS[unit].display;
}

/**
 * Format a stored area (square metres) for DISPLAY in the chosen unit, WITH the
 * unit label appended: '5 rai' · '8,000 m²' · '8K m²'. The reusable, options-aware
 * sibling of {@link formatLandSize} (value-only, no label) — consumed directly by
 * any label AND wrapped by the Media Canvas `area` formatter descriptor, so a
 * card and the canvas render land size identically. At its defaults (full /
 * auto) the output matches the prior hand-rolled `formatLandSize + label`.
 * Non-finite → '' (no value to show).
 */
export function formatArea(sqm: number, opts: AreaFormatOptions): string {
	if (!Number.isFinite(sqm)) return '';
	const { unit, notation = 'full', decimals = 'auto' } = opts;
	const value = fromSqm(sqm, unit);
	const maxFrac = decimals === 'off' ? 0 : decimalPlacesFor(unit);
	const num = value.toLocaleString('en-US', {
		notation: notation === 'short' ? 'compact' : 'standard',
		minimumFractionDigits: 0,
		maximumFractionDigits: notation === 'short' ? 1 : maxFrac
	});
	return `${num} ${areaUnitLabel(unit)}`;
}

/**
 * Format a stored area (square metres) as the Thai cadastral compound
 * "rai · ngan · wah²": e.g. 8520 → "5 rai 1 ngan 30 wah". Higher zero
 * components are dropped (120 → "30 wah"); an empty area → "0 wah". The
 * decimal sibling is {@link formatArea} (e.g. "2.13 rai"). A reusable Thai
 * cadastral display primitive (property cards / canvas banners are the
 * intended consumers). Computes in integer wah² internally (1 wah²=4 sqm,
 * 1 ngan=100 wah², 1 rai=400 wah²) so rounding never rolls 99.6 wah into a
 * phantom unit. Non-finite / negative → '' (no value to show).
 */
export function formatAreaCompound(sqm: number): string {
	if (!Number.isFinite(sqm) || sqm < 0) return '';
	const totalWah = Math.round(sqm / 4);
	const rai = Math.floor(totalWah / 400);
	const rem = totalWah % 400;
	const ngan = Math.floor(rem / 100);
	const wah = rem % 100;
	const parts: string[] = [];
	if (rai > 0) parts.push(`${rai.toLocaleString('en-US')} rai`);
	if (ngan > 0) parts.push(`${ngan} ngan`);
	if (wah > 0) parts.push(`${wah} wah`);
	return parts.length > 0 ? parts.join(' ') : '0 wah';
}
