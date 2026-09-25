// TDD RED — @sbx/canvas-kit formatChoiceLabel (INC-0 · SC-INC0-1 · TC-INC0-1).
// The Inspector format-knob dropdowns must show the CHOICE NAME (humanized), not a
// formatted price preview — so an editor can tell which option is selected; the live
// formatted value stays on the result line. This pure helper produces the dropdown label.
//
// CASE ENUMERATION
//   TC-FCL-01 success  currency display choices title-case: symbol→Symbol, code→Code, none→None
//   TC-FCL-02 success  notation + decimals choices: full→Full, short→Short, off→Off, on→On
//   TC-FCL-03 success  area-unit + auto choices stay readable: sqm→Sqm, rai→Rai, auto→Auto
//   TC-FCL-04 edge     hyphenated enum codes title-case each part: freehold-leasehold→Freehold-Leasehold
//   TC-FCL-05 edge     empty string returns unchanged (never throws)
import { describe, it, expect } from 'vitest';
import type { FormatterDescriptor } from './types/source.js';
import { formatChoiceLabel, formatChoicePreview, formatKnobChoices } from './format-labels.js';

describe('formatChoiceLabel', () => {
	it('TC-FCL-01 title-cases currency display choices', () => {
		expect(formatChoiceLabel('symbol')).toBe('Symbol');
		expect(formatChoiceLabel('code')).toBe('Code');
		expect(formatChoiceLabel('none')).toBe('None');
	});

	it('TC-FCL-02 title-cases notation + decimals choices', () => {
		expect(formatChoiceLabel('full')).toBe('Full');
		expect(formatChoiceLabel('short')).toBe('Short');
		expect(formatChoiceLabel('off')).toBe('Off');
		expect(formatChoiceLabel('on')).toBe('On');
	});

	it('TC-FCL-03 keeps area-unit + auto choices readable', () => {
		expect(formatChoiceLabel('sqm')).toBe('Sqm');
		expect(formatChoiceLabel('rai')).toBe('Rai');
		expect(formatChoiceLabel('auto')).toBe('Auto');
	});

	it('TC-FCL-04 title-cases each part of a hyphenated enum code', () => {
		expect(formatChoiceLabel('freehold-leasehold')).toBe('Freehold-Leasehold');
	});

	it('TC-FCL-05 returns empty string unchanged (never throws)', () => {
		expect(formatChoiceLabel('')).toBe('');
	});
});

// INC-0b — isolated per-knob previews on a fixed sample. The OLD dropdown previews
// composited the binding's CURRENT options (every dropdown jumped as any knob changed);
// these helpers render each choice on a fixed sample with the OTHER knobs held at their
// defaults, so a knob's dropdown is stable + shows only its own effect. A synthetic
// descriptor controls the output so the contract — not a specific currency string — is tested.
//   wrap  — visibly distinct per choice ('123' vs '[123]')           → PREVIEW labels
//   noop  — choice ignored by format() so both previews collapse     → NAME fallback
//   scale — only differs at magnitude; carries its own previewSample → PREVIEW from previewSample
const knobDesc: FormatterDescriptor = {
	name: 'knob',
	sample: 123,
	options: [
		{ key: 'wrap', label: 'Wrap', choices: ['plain', 'bracket'], default: 'plain' },
		{ key: 'noop', label: 'No Effect', choices: ['x', 'y'], default: 'x' },
		{ key: 'scale', label: 'Scale', choices: ['full', 'short'], default: 'full', previewSample: 1000 }
	],
	format: (value, opts) => {
		const n = Number(value);
		const shown = opts.scale === 'short' ? `${n / 1000}k` : String(n);
		return opts.wrap === 'bracket' ? `[${shown}]` : shown;
	}
};

describe('formatChoicePreview', () => {
	it('TC-FCP-01 renders one choice with the OTHER knobs held at their defaults', () => {
		// wrap=bracket while noop + scale stay at default → only the wrap effect shows.
		expect(formatChoicePreview(knobDesc, 'wrap', 'bracket')).toBe('[123]');
		expect(formatChoicePreview(knobDesc, 'wrap', 'plain')).toBe('123');
	});

	it('TC-FCP-02 uses the option previewSample (not the descriptor sample) when set', () => {
		// scale carries previewSample 1000 so Short actually demonstrates the magnitude.
		expect(formatChoicePreview(knobDesc, 'scale', 'short')).toBe('1k');
		expect(formatChoicePreview(knobDesc, 'scale', 'full')).toBe('1000');
	});

	it('TC-FCP-03 is independent of a binding’s other saved knobs (no jumpiness)', () => {
		// The preview never reads current options — only descriptor defaults — so the wrap
		// dropdown is identical regardless of what scale/noop happen to be set to.
		expect(formatChoicePreview(knobDesc, 'wrap', 'bracket', 'ignored-arg')).toBe('[123]');
	});
});

describe('formatKnobChoices', () => {
	it('TC-FKC-01 labels distinct choices with their previews', () => {
		const wrap = knobDesc.options[0];
		expect(formatKnobChoices(knobDesc, wrap)).toEqual([
			{ value: 'plain', label: '123' },
			{ value: 'bracket', label: '[123]' }
		]);
	});

	it('TC-FKC-02 falls back to humanized NAMES when previews collapse to one string', () => {
		// noop is ignored by format() → both previews are '123' → names are clearer.
		const noop = knobDesc.options[1];
		expect(formatKnobChoices(knobDesc, noop)).toEqual([
			{ value: 'x', label: 'X' },
			{ value: 'y', label: 'Y' }
		]);
	});

	it('TC-FKC-03 previews a magnitude knob via its previewSample', () => {
		const scale = knobDesc.options[2];
		expect(formatKnobChoices(knobDesc, scale)).toEqual([
			{ value: 'full', label: '1000' },
			{ value: 'short', label: '1k' }
		]);
	});
});
