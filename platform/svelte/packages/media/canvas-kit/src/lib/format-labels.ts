// Inspector format-knob choice rendering (S-FORMAT). A FormatterDescriptor exposes one or
// more knobs (FormatOption); the Inspector renders a <Select> per knob and must label each
// choice clearly. `formatKnobChoices` picks the strategy PER KNOB:
//   • PREVIEW (INC-0b) — the choice rendered on a small fixed SAMPLE with every OTHER knob
//     held at its DEFAULT, so the dropdown shows exactly what THIS choice does and stays
//     stable as the binding's other knobs change. This replaces the old preview that
//     composited the binding's *current* options (every dropdown jumped as any knob moved).
//     Used when a knob's choices render distinctly — e.g. currency display → ฿123 / 123 THB / 123.
//   • NAME (INC-0) — the humanized choice code, used when the previews don't distinguish the
//     choices at the sample (e.g. an area decimals knob whose auto/off both render "1,600 m²"
//     at the default sqm unit) — there the name ("Auto" / "Off") is the clearer label.
// The live, real-record value stays on the Inspector's ⟶ result line, never in these labels.
// Pure + presentation-only: no domain knowledge lives here.

import type { FormatterDescriptor, FormatOption } from './types/source.js';
import { descriptorDefaults } from './resolve.js';
import { capitalizeTokens } from './text-case.js';

/**
 * Humanize a formatter-knob choice value for an Inspector <Select> option label:
 * 'symbol' → 'Symbol', 'freehold-leasehold' → 'Freehold-Leasehold'. Capitalizes the first
 * letter of each whitespace/hyphen/underscore-separated part, preserving the separators +
 * any existing case (acronyms survive); '' → ''.
 */
export function formatChoiceLabel(value: string): string {
	return capitalizeTokens(value);
}

/**
 * Preview ONE knob's choice: the descriptor's `sample` (or the option's own `previewSample`)
 * formatted with `optionKey` set to `choice` and every OTHER knob held at its DEFAULT. So each
 * knob's preview is independent of the binding's other saved options (no jumpiness) and shows
 * precisely what `choice` does. `arg` is the formatter argument — the part after ':' in the
 * format hint ('currency:THB' → 'THB').
 */
export function formatChoicePreview(
	descriptor: FormatterDescriptor,
	optionKey: string,
	choice: string,
	arg?: string
): string {
	const option = descriptor.options.find((o) => o.key === optionKey);
	const sample = option?.previewSample ?? descriptor.sample;
	const opts = { ...descriptorDefaults(descriptor), [optionKey]: choice };
	return descriptor.format(sample, opts, arg);
}

/**
 * The Inspector <Select> options for one knob, as { value, label } pairs. Labels are PREVIEWS
 * ({@link formatChoicePreview}) when the knob's choices render distinctly at the sample, else
 * the humanized NAMES ({@link formatChoiceLabel}) when every preview collapses to one string
 * (a knob with no visible effect at the sample). One contract, both modes — so the caller never
 * special-cases a knob.
 */
export function formatKnobChoices(
	descriptor: FormatterDescriptor,
	option: FormatOption,
	arg?: string
): { value: string; label: string }[] {
	const previews = option.choices.map((c) => formatChoicePreview(descriptor, option.key, c, arg));
	const distinct = new Set(previews).size > 1;
	return option.choices.map((c, i) => ({
		value: c,
		label: distinct ? previews[i] : formatChoiceLabel(c)
	}));
}
