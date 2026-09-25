// Enum/list display formatter (INC-D). A reusable, string-valued FormatterDescriptor that maps
// a raw enum code to a marketing-chosen ALIAS — e.g. transactionType 'sale' → "For Sale" — so
// any consumer (Media Canvas today; the text-template composer + post/content tools next) renders
// enums the way the business wants, not the raw code. Generic by construction: the kit owns the
// machinery, the consumer owns the alias data + which sets exist (modular-architecture-first).

import type { FormatterDescriptor } from './types/source.js';
import { capitalizeTokens } from './text-case.js';

/** Marketing aliases: enum-SET name → (raw code → display alias). The set name is the format
 *  hint's arg ('enum:transactionType' → 'transactionType'). Consumer-owned + MUTABLE: the
 *  descriptor reads it at format time, so a later persistence layer can swap marketing's saved
 *  edits in without rebuilding the descriptor (the "persisted later" half of the MVP). */
export type EnumAliasMap = Record<string, Record<string, string>>;

/** Proper Title Case ('for sale' → 'For Sale', 'FOR SALE' → 'For Sale') — capitalize each
 *  word AND lowercase the rest, via the shared {@link capitalizeTokens} primitive. */
function titleCase(value: string): string {
	return capitalizeTokens(value, { lowerRest: true });
}

/** Humanize a raw enum code when no alias is set: separators → spaces, title-cased
 *  ('pool-villa' → 'Pool Villa', 'town_house' → 'Town House'). */
function humanizeCode(code: string): string {
	return titleCase(code.replace(/[-_]+/g, ' ').trim());
}

/** Restyle the resolved display per the 'case' knob; 'default' keeps the alias/humanized text. */
function applyCase(value: string, mode: string | undefined): string {
	switch (mode) {
		case 'upper':
			return value.toUpperCase();
		case 'lower':
			return value.toLowerCase();
		case 'title':
			return titleCase(value);
		default:
			return value;
	}
}

/**
 * Build a string-valued enum/list {@link FormatterDescriptor}. The format hint's `arg` names the
 * enum SET ('enum:transactionType' → 'transactionType'); the raw code is mapped to `aliases[set]`,
 * else humanized. A 'case' knob (default/upper/lower/title) restyles the result. `valueType:
 * 'string'` routes the raw token unchanged (INC-A) — never Number-coerced to NaN.
 *
 * @param aliases  consumer-owned alias map, read at format time (so it can be mutated/persisted).
 * @param sample   preview code for the Inspector knob previews (default 'sale').
 */
export function makeEnumFormatter(aliases: EnumAliasMap, sample = 'sale'): FormatterDescriptor {
	return {
		name: 'enum',
		valueType: 'string',
		sample,
		options: [
			{ key: 'case', label: 'Case', choices: ['default', 'upper', 'lower', 'title'], default: 'default' }
		],
		format: (value, opts, arg): string => {
			const code = String(value);
			const set = arg ?? '';
			const display = aliases[set]?.[code] ?? humanizeCode(code);
			return applyCase(display, opts.case);
		}
	};
}
