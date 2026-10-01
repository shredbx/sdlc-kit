/**
 * Contrast Validator - Zone-based color contrast validation
 *
 * Implements the Color Contrast Zone Model from domain expert knowledge.
 * Divides colors into LIGHT (L >= 0.5) and DARK (L < 0.5) zones.
 * Valid pairings require opposite zones.
 *
 * Why this approach?
 * - SIMPLER than calculating WCAG ratios (threshold check vs complex math)
 * - PREVENTIVE - catches issues at design time
 * - UNIVERSAL - works with any color system
 * - DETERMINISTIC - no judgment calls, just zone membership
 *
 * @see .sbx/domain-experts/ui-design-systems/knowledge/color-systems/color-contrast-zones.yml
 *
 * Usage:
 *   import { getZone, validatePairing, validateBrandContrast } from './contrastValidator';
 *
 *   // Get zone for a color
 *   const zone = getZone('#ffffff'); // 'light'
 *
 *   // Validate bg/fg pairing
 *   const result = validatePairing('#0f0f1a', '#f0f0f0'); // { valid: true }
 *
 *   // Validate brand config
 *   const report = validateBrandContrast(hubBrand);
 */

// =============================================================================
// TYPES
// =============================================================================

/** Lightness zone classification */
export type Zone = 'light' | 'dark';

/** Validation result for a color pairing */
export interface ValidationResult {
	valid: boolean;
	backgroundZone: Zone;
	foregroundZone: Zone;
	reason: string;
	suggestion?: string;
}

/** Validation options */
export interface ValidationOptions {
	/** Element is disabled (allows same-zone) */
	isDisabled?: boolean;
	/** Element is decorative (allows same-zone) */
	isDecorative?: boolean;
	/** Text is large (18pt+ or 14pt+ bold, allows same-zone) */
	isLargeText?: boolean;
}

/** Brand contrast validation report */
export interface ContrastReport {
	brandId: string;
	valid: boolean;
	issues: ContrastIssue[];
	checkedPairings: number;
}

/** Single contrast issue in report */
export interface ContrastIssue {
	pairing: string;
	background: { token: string; value: string; zone: Zone };
	foreground: { token: string; value: string; zone: Zone };
	severity: 'error' | 'warning';
	message: string;
}

// =============================================================================
// ZONE THRESHOLD
// =============================================================================

/** OKLCH lightness threshold (L >= 0.5 = light zone) */
const ZONE_THRESHOLD = 0.5;

// =============================================================================
// COLOR PARSING
// =============================================================================

/**
 * Parse hex color to RGB values.
 */
function parseHex(hex: string): { r: number; g: number; b: number } | null {
	// Remove # prefix
	const clean = hex.replace('#', '');

	// Handle 3-char and 6-char hex
	let r: number, g: number, b: number;

	if (clean.length === 3) {
		r = parseInt(clean[0] + clean[0], 16);
		g = parseInt(clean[1] + clean[1], 16);
		b = parseInt(clean[2] + clean[2], 16);
	} else if (clean.length === 6) {
		r = parseInt(clean.substring(0, 2), 16);
		g = parseInt(clean.substring(2, 4), 16);
		b = parseInt(clean.substring(4, 6), 16);
	} else {
		return null;
	}

	if (isNaN(r) || isNaN(g) || isNaN(b)) {
		return null;
	}

	return { r, g, b };
}

/**
 * Parse OKLCH color string.
 * Format: oklch(L C H) or oklch(L% C H)
 */
function parseOKLCH(color: string): { l: number; c: number; h: number } | null {
	const match = color.match(/oklch\(\s*([\d.]+)%?\s+([\d.]+)\s+([\d.]+)\s*\)/);
	if (!match) return null;

	let l = parseFloat(match[1]);
	const c = parseFloat(match[2]);
	const h = parseFloat(match[3]);

	// Normalize percentage to 0-1
	if (l > 1) l = l / 100;

	return { l, c, h };
}

/**
 * Convert RGB to OKLCH lightness.
 * Simplified conversion focusing on perceptual lightness.
 *
 * Note: Full OKLCH conversion requires a color library.
 * This approximation uses relative luminance as proxy for L.
 */
function rgbToOKLCHLightness(r: number, g: number, b: number): number {
	// Normalize to 0-1
	const rn = r / 255;
	const gn = g / 255;
	const bn = b / 255;

	// Convert to linear RGB
	const toLinear = (c: number): number => {
		return c <= 0.04045 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
	};

	const rl = toLinear(rn);
	const gl = toLinear(gn);
	const bl = toLinear(bn);

	// Relative luminance (Y in XYZ)
	const y = 0.2126 * rl + 0.7152 * gl + 0.0722 * bl;

	// Approximate OKLCH lightness from luminance
	// OKLCH L is roughly cube root of luminance (perceptual)
	const l = Math.cbrt(y);

	return l;
}

// =============================================================================
// ZONE DETECTION
// =============================================================================

/**
 * Get the lightness zone for a color.
 * Supports hex (#rrggbb, #rgb) and oklch() formats.
 *
 * @param color - Color value (hex or oklch)
 * @returns Zone ('light' or 'dark') or null if unparseable
 */
export function getZone(color: string): Zone | null {
	const trimmed = color.trim().toLowerCase();

	// Handle OKLCH format
	if (trimmed.startsWith('oklch')) {
		const parsed = parseOKLCH(trimmed);
		if (!parsed) return null;
		return parsed.l >= ZONE_THRESHOLD ? 'light' : 'dark';
	}

	// Handle hex format
	if (trimmed.startsWith('#') || /^[0-9a-f]{3,6}$/i.test(trimmed)) {
		const rgb = parseHex(trimmed);
		if (!rgb) return null;
		const lightness = rgbToOKLCHLightness(rgb.r, rgb.g, rgb.b);
		return lightness >= ZONE_THRESHOLD ? 'light' : 'dark';
	}

	// Handle named colors (basic set)
	const namedColors: Record<string, Zone> = {
		white: 'light',
		black: 'dark',
		transparent: 'light' // Assume light for transparent
	};

	if (namedColors[trimmed]) {
		return namedColors[trimmed];
	}

	// Unknown format
	return null;
}

/**
 * Get OKLCH lightness value for a color.
 * Returns null if color cannot be parsed.
 */
export function getLightness(color: string): number | null {
	const trimmed = color.trim().toLowerCase();

	// Handle OKLCH format
	if (trimmed.startsWith('oklch')) {
		const parsed = parseOKLCH(trimmed);
		return parsed?.l ?? null;
	}

	// Handle hex format
	if (trimmed.startsWith('#') || /^[0-9a-f]{3,6}$/i.test(trimmed)) {
		const rgb = parseHex(trimmed);
		if (!rgb) return null;
		return rgbToOKLCHLightness(rgb.r, rgb.g, rgb.b);
	}

	return null;
}

// =============================================================================
// PAIRING VALIDATION
// =============================================================================

/**
 * Validate a background/foreground color pairing.
 * Colors from opposite zones are valid.
 *
 * @param background - Background color value
 * @param foreground - Foreground color value
 * @param options - Validation options (exceptions)
 */
export function validatePairing(
	background: string,
	foreground: string,
	options: ValidationOptions = {}
): ValidationResult {
	const bgZone = getZone(background);
	const fgZone = getZone(foreground);

	// Handle unparseable colors
	if (!bgZone || !fgZone) {
		return {
			valid: false,
			backgroundZone: bgZone ?? 'dark',
			foregroundZone: fgZone ?? 'light',
			reason: 'unparseable-color',
			suggestion: 'Use hex (#rrggbb) or oklch(L C H) format'
		};
	}

	// Check if same zone (violation)
	if (bgZone === fgZone) {
		// Check exceptions
		if (options.isDisabled) {
			return {
				valid: true,
				backgroundZone: bgZone,
				foregroundZone: fgZone,
				reason: 'disabled-exception'
			};
		}

		if (options.isDecorative) {
			return {
				valid: true,
				backgroundZone: bgZone,
				foregroundZone: fgZone,
				reason: 'decorative-exception'
			};
		}

		if (options.isLargeText) {
			return {
				valid: true,
				backgroundZone: bgZone,
				foregroundZone: fgZone,
				reason: 'large-text-exception'
			};
		}

		// Same zone violation
		return {
			valid: false,
			backgroundZone: bgZone,
			foregroundZone: fgZone,
			reason: 'same-zone-violation',
			suggestion: `Move ${bgZone === 'light' ? 'foreground' : 'background'} to ${bgZone === 'light' ? 'dark' : 'light'} zone`
		};
	}

	// Opposite zones - valid
	return {
		valid: true,
		backgroundZone: bgZone,
		foregroundZone: fgZone,
		reason: 'opposite-zones'
	};
}

// =============================================================================
// BRAND VALIDATION
// =============================================================================

/**
 * Common semantic pairings to validate in a brand.
 */
const SEMANTIC_PAIRINGS = [
	{ bg: 'background', fg: 'text', name: 'Background + Text' },
	{ bg: 'background', fg: 'textMuted', name: 'Background + Text Muted' },
	{ bg: 'surface', fg: 'text', name: 'Surface + Text' },
	{ bg: 'surface', fg: 'textMuted', name: 'Surface + Text Muted' },
	{ bg: 'surfaceElevated', fg: 'text', name: 'Surface Elevated + Text' },
	{ bg: 'primary', fg: 'text', name: 'Primary + Text', isAccent: true }
];

/**
 * Validate color contrast in a brand configuration.
 * Checks common semantic pairings for zone compliance.
 *
 * @param brand - Brand configuration with colors
 */
export function validateBrandContrast(brand: {
	id: string;
	colors: Record<string, string | undefined>;
}): ContrastReport {
	const issues: ContrastIssue[] = [];
	let checkedCount = 0;

	for (const pairing of SEMANTIC_PAIRINGS) {
		const bgValue = brand.colors[pairing.bg];
		const fgValue = brand.colors[pairing.fg];

		// Skip if colors not defined
		if (!bgValue || !fgValue) continue;

		checkedCount++;

		const result = validatePairing(bgValue, fgValue);

		if (!result.valid && result.reason === 'same-zone-violation') {
			issues.push({
				pairing: pairing.name,
				background: {
					token: pairing.bg,
					value: bgValue,
					zone: result.backgroundZone
				},
				foreground: {
					token: pairing.fg,
					value: fgValue,
					zone: result.foregroundZone
				},
				severity: 'error',
				message: `Both colors in ${result.backgroundZone} zone. ${result.suggestion}`
			});
		}
	}

	return {
		brandId: brand.id,
		valid: issues.length === 0,
		issues,
		checkedPairings: checkedCount
	};
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

/**
 * Get zone label for display.
 */
export function getZoneLabel(zone: Zone): string {
	return zone === 'light' ? 'Light Zone (L ≥ 0.5)' : 'Dark Zone (L < 0.5)';
}

/**
 * Get CSS class suffix for zone.
 */
export function getZoneClass(zone: Zone): string {
	return `zone-${zone}`;
}

/**
 * Check if color is near zone threshold.
 * Colors with L between 0.4 and 0.6 are considered "threshold-adjacent"
 * and may need manual contrast checking.
 */
export function isNearThreshold(color: string): boolean {
	const lightness = getLightness(color);
	if (lightness === null) return false;
	return lightness >= 0.4 && lightness <= 0.6;
}

/**
 * Suggest a safer color for better contrast.
 * Moves color further from threshold.
 */
export function suggestSaferColor(color: string, targetZone: Zone): string | null {
	const lightness = getLightness(color);
	if (lightness === null) return null;

	// Target lightness values for safety margin
	const targetL = targetZone === 'light' ? 0.85 : 0.25;

	// Return suggestion as OKLCH (can be converted)
	return `oklch(${targetL.toFixed(2)} 0 0) /* Move to L=${targetL} for better contrast */`;
}

// =============================================================================
// TAILWIND HELPERS
// =============================================================================

/**
 * Map Tailwind shade to zone.
 * 50-400 = light zone, 500-900 = dark zone
 */
export function tailwindShadeToZone(shade: number): Zone {
	return shade <= 400 ? 'light' : 'dark';
}

/**
 * Validate Tailwind class combination.
 */
export function validateTailwindPairing(bgClass: string, textClass: string): ValidationResult {
	// Extract shade numbers
	const bgMatch = bgClass.match(/-(\d+)(?:\/|$)/);
	const textMatch = textClass.match(/-(\d+)(?:\/|$)/);

	if (!bgMatch || !textMatch) {
		return {
			valid: false,
			backgroundZone: 'dark',
			foregroundZone: 'light',
			reason: 'unparseable-tailwind',
			suggestion: 'Use standard Tailwind shade classes (50-900)'
		};
	}

	const bgShade = parseInt(bgMatch[1]);
	const textShade = parseInt(textMatch[1]);

	const bgZone = tailwindShadeToZone(bgShade);
	const textZone = tailwindShadeToZone(textShade);

	if (bgZone === textZone) {
		return {
			valid: false,
			backgroundZone: bgZone,
			foregroundZone: textZone,
			reason: 'same-zone-violation',
			suggestion: `Use ${bgZone === 'light' ? '500-900' : '50-400'} shade for text`
		};
	}

	return {
		valid: true,
		backgroundZone: bgZone,
		foregroundZone: textZone,
		reason: 'opposite-zones'
	};
}
