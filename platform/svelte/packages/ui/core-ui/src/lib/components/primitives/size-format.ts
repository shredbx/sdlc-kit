/**
 * size-format.ts — display formatting for SizeInput.
 *
 * `comma` mode renders thousands separators (reusing the money formatter) while
 * the bound value stays a plain number string. Any other mode returns the value
 * untouched. Empty stays empty — never inject a placeholder 0.
 */
import { withCommas } from './money';

export type SizeFormat = 'comma' | 'plain';

export function formatSizeValue(value: string, format?: SizeFormat): string {
	if (!value) return '';
	return format === 'comma' ? withCommas(value) : value;
}
