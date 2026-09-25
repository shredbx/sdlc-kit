// TDD RED — @sbx/canvas-kit pure resolvers (FDD4.PD.TEST_RED, slice 1).
// Every test MUST fail until resolve.ts is implemented (GREEN).
//
// CASE ENUMERATION
// resolveLayerAtTime (scenario MC-SC-10 — keyframe interpolation + easing)
//   TC-RA-01 success  two opacity keyframes, linear, midpoint → 0.5
//   TC-RA-02 success  ease-out at midpoint → > linear midpoint (curve applied)
//   TC-RA-03 success  numeric x interpolates at t=250 over [100→300] → 150
//   TC-RA-04 edge     t before first keyframe → clamps to first value
//   TC-RA-05 edge     t after last keyframe → clamps to last value
//   TC-RA-06 success  two tracks (x + opacity) both resolve at one t
//   TC-RA-07 edge     t exactly on a keyframe → that keyframe's value
//   TC-RA-08 edge     string-valued track → steps (holds previous), no interpolation
//   TC-RA-09 edge     layer with no animations → returned unchanged
//   TC-RA-10 edge     single-keyframe track → constant value
//   TC-RA-11 regress  input layer is never mutated
//
// resolveBindings (scenarios MC-SC-06 attach/active, MC-SC-07 bind+format,
//                   MC-SC-08 fallback/never-clip, MC-SC-09 placeholder)
//   TC-RB-01 success  content ← 'price' with currency:THB formatter → '฿28,500,000'
//   TC-RB-02 success  content ← 'title', no formatter → raw String value
//   TC-RB-03 success  image src ← nested 'coverImage.url' (dotted token)
//   TC-RB-04 edge     missing token + fallback → fallback text
//   TC-RB-05 edge     empty-string value + fallback → fallback text
//   TC-RB-06 edge     missing token + NO fallback → '' (never undefined → never clips)
//   TC-RB-07 success  content ← 'landSize' with area:rai formatter → '1 rai'
//   TC-RB-08 success  placeholder:true resolves identically to placeholder:false
//   TC-RB-09 edge     layer with no bindings → returned unchanged
//   TC-RB-10 success  multiple bindings on one layer (content + src) both apply
//   TC-RB-11 regress  input layer is never mutated
//   TC-RB-12 failure  dotted token with missing intermediate → fallback, no throw
//
//   override precedence — D20 / MC-SC-15 (override → token→formatter → fallback)
//   TC-RB-13 success  override set → used verbatim, wins over the live token value
//   TC-RB-14 success  override wins even when the token would resolve to fallback (missing)
//   TC-RB-15 success  override is NOT re-formatted (currency format present, override used as-is)
//   TC-RB-16 edge     override === '' (explicit blank) → '' (user's blank, not the fallback)
//   TC-RB-17 success  override undefined (Restore) → falls through to the live token value
//   TC-RB-18 success  image src override → replaces the bound image url verbatim
//
//   format-option merge — increment 2 (descriptor defaults + per-binding formatOptions)
//   TC-RB-19 success  binding.formatOptions overrides the descriptor default (override wins)
//   TC-RB-20 regress  no formatOptions → descriptor defaults (back-compat, byte-identical)
//   TC-RB-21 edge     unknown formatter name → raw String(value) passthrough

import { describe, expect, it } from 'vitest';
import { resolveBindings, resolveLayerAtTime, resolveTemplateContent } from './resolve.js';
import type { AnimationTrack } from './types/animation.js';
import type { Binding } from './types/binding.js';
import type { SourceSnapshot } from './types/source.js';
import type { Template } from '@sbx/text-template';
import {
	makeImageLayer,
	makeTextLayer,
	propertySnapshot,
	testFormatters,
	withAnimations,
	withBindings
} from './fixtures.js';

describe('resolveLayerAtTime (MC-SC-10)', () => {
	const opacityTrack = (easing: AnimationTrack['keyframes'][number]['easing']): AnimationTrack => ({
		property: 'opacity',
		keyframes: [
			{ time: 0, value: 0, easing },
			{ time: 1000, value: 1, easing }
		]
	});

	it('TC-RA-01 interpolates a numeric property linearly at the midpoint', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [opacityTrack('linear')]);
		const out = resolveLayerAtTime(layer, 500);
		expect(out.opacity).toBeCloseTo(0.5, 5);
	});

	it('TC-RA-02 applies an ease-out curve (faster early than linear)', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [opacityTrack('easeOut')]);
		const out = resolveLayerAtTime(layer, 500);
		expect(out.opacity).toBeGreaterThan(0.5);
	});

	it('TC-RA-03 interpolates x at t=250 over [100→300] → 150', () => {
		const layer = withAnimations(makeTextLayer({ x: 100 }), [
			{ property: 'x', keyframes: [
				{ time: 0, value: 100, easing: 'linear' },
				{ time: 1000, value: 300, easing: 'linear' }
			] }
		]);
		const out = resolveLayerAtTime(layer, 250);
		expect(out.x).toBeCloseTo(150, 5);
	});

	it('TC-RA-04 clamps to the first keyframe before the track starts', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [opacityTrack('linear')]);
		const out = resolveLayerAtTime(layer, -100);
		expect(out.opacity).toBe(0);
	});

	it('TC-RA-05 clamps to the last keyframe after the track ends', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [opacityTrack('linear')]);
		const out = resolveLayerAtTime(layer, 5000);
		expect(out.opacity).toBe(1);
	});

	it('TC-RA-06 resolves multiple tracks at one time', () => {
		const layer = withAnimations(makeTextLayer({ x: 0, opacity: 0 }), [
			{ property: 'x', keyframes: [
				{ time: 0, value: 0, easing: 'linear' },
				{ time: 1000, value: 200, easing: 'linear' }
			] },
			opacityTrack('linear')
		]);
		const out = resolveLayerAtTime(layer, 500);
		expect(out.x).toBeCloseTo(100, 5);
		expect(out.opacity).toBeCloseTo(0.5, 5);
	});

	it('TC-RA-07 returns the exact keyframe value when t lands on a keyframe', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [opacityTrack('linear')]);
		expect(resolveLayerAtTime(layer, 0).opacity).toBe(0);
		expect(resolveLayerAtTime(layer, 1000).opacity).toBe(1);
	});

	it('TC-RA-08 steps string-valued tracks (holds previous keyframe, no interpolation)', () => {
		const layer = withAnimations(makeTextLayer({ content: 'A' }), [
			{ property: 'content', keyframes: [
				{ time: 0, value: 'A', easing: 'linear' },
				{ time: 1000, value: 'B', easing: 'linear' }
			] }
		]);
		expect(resolveLayerAtTime(layer, 500).content).toBe('A');
		expect(resolveLayerAtTime(layer, 1000).content).toBe('B');
	});

	it('TC-RA-09 returns a layer with no animations unchanged', () => {
		const layer = makeTextLayer({ opacity: 0.42 });
		expect(resolveLayerAtTime(layer, 500)).toEqual(layer);
	});

	it('TC-RA-10 holds a single-keyframe track at a constant value', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [
			{ property: 'opacity', keyframes: [{ time: 500, value: 0.7, easing: 'linear' }] }
		]);
		expect(resolveLayerAtTime(layer, 0).opacity).toBe(0.7);
		expect(resolveLayerAtTime(layer, 9999).opacity).toBe(0.7);
	});

	it('TC-RA-11 does not mutate the input layer', () => {
		const layer = withAnimations(makeTextLayer({ opacity: 0 }), [opacityTrack('linear')]);
		resolveLayerAtTime(layer, 500);
		expect(layer.opacity).toBe(0);
	});
});

describe('resolveBindings (MC-SC-06 / MC-SC-07 / MC-SC-08 / MC-SC-09)', () => {
	const bind = (overrides: Partial<Binding>): Binding => ({
		property: 'content',
		sourceAlias: 'primary',
		token: 'title',
		...overrides
	});

	it('TC-RB-01 binds content to a field and applies the currency formatter', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'price', format: 'currency:THB' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('฿28,500,000');
	});

	it('TC-RB-02 binds content with no formatter as the raw String value', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'title' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('Sunset Ridge Villa');
	});

	it('TC-RB-03 binds an image src from a nested dotted token', () => {
		const layer = withBindings(makeImageLayer(), [bind({ property: 'src', token: 'coverImage.url' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.src).toBe('https://r2.example/bestie-realestate/cover-1.jpg');
	});

	it('TC-RB-04 uses the fallback when the token is missing on the active record', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'salePrice', fallback: 'POA' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('POA');
	});

	it('TC-RB-05 uses the fallback when the value is an empty string', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'title', fallback: '—' })]);
		const out = resolveBindings(layer, { title: '' }, testFormatters);
		expect(out.content).toBe('—');
	});

	it('TC-RB-06 yields an empty string (never undefined) when missing with no fallback', () => {
		const layer = withBindings(makeTextLayer({ content: 'seed' }), [bind({ token: 'salePrice' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('');
	});

	it('TC-RB-07 applies the area formatter (m² → rai)', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'landSize', format: 'area:rai' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('1 rai');
	});

	it('TC-RB-08 resolves a placeholder binding identically to a non-placeholder one', () => {
		const base = bind({ token: 'title' });
		const plain = resolveBindings(withBindings(makeTextLayer(), [base]), propertySnapshot, testFormatters);
		const placeholder = resolveBindings(
			withBindings(makeTextLayer(), [{ ...base, placeholder: true }]),
			propertySnapshot,
			testFormatters
		);
		expect(placeholder.content).toBe(plain.content);
	});

	it('TC-RB-09 returns a layer with no bindings unchanged', () => {
		const layer = makeTextLayer({ content: 'static' });
		expect(resolveBindings(layer, propertySnapshot, testFormatters)).toEqual(layer);
	});

	it('TC-RB-10 applies multiple bindings on one layer', () => {
		const layer = withBindings(makeImageLayer({ content: '' }), [
			bind({ property: 'content', token: 'title' }),
			bind({ property: 'src', token: 'coverImage.url' })
		]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('Sunset Ridge Villa');
		expect(out.src).toBe('https://r2.example/bestie-realestate/cover-1.jpg');
	});

	it('TC-RB-11 does not mutate the input layer', () => {
		const layer = withBindings(makeTextLayer({ content: 'seed' }), [bind({ token: 'title' })]);
		resolveBindings(layer, propertySnapshot, testFormatters);
		expect(layer.content).toBe('seed');
	});

	it('TC-RB-12 falls back (no throw) when a dotted token has a missing intermediate', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'agent.name', fallback: 'n/a' })]);
		expect(() => resolveBindings(layer, propertySnapshot, testFormatters)).not.toThrow();
		expect(resolveBindings(layer, propertySnapshot, testFormatters).content).toBe('n/a');
	});

	// Override precedence — D20 / MC-SC-15 (override → token→formatter → fallback).
	it('TC-RB-13 uses the override verbatim, winning over the live token value', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'title', override: 'Hand-tuned headline' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('Hand-tuned headline');
	});

	it('TC-RB-14 uses the override even when the token would resolve to the fallback', () => {
		const layer = withBindings(makeTextLayer(), [
			bind({ token: 'salePrice', fallback: 'POA', override: 'On request' })
		]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('On request');
	});

	it('TC-RB-15 does NOT re-format the override (currency format present, used as-is)', () => {
		const layer = withBindings(makeTextLayer(), [
			bind({ token: 'price', format: 'currency:THB', override: '฿29M (negotiable)' })
		]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('฿29M (negotiable)');
	});

	it('TC-RB-16 treats an explicit empty override as blank (not the fallback)', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'title', fallback: '—', override: '' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('');
	});

	it('TC-RB-17 falls through to the live token value when the override is undefined (Restore)', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'title', override: undefined })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('Sunset Ridge Villa');
	});

	it('TC-RB-18 replaces a bound image src with the override url verbatim', () => {
		const layer = withBindings(makeImageLayer(), [
			bind({ property: 'src', token: 'coverImage.url', override: 'https://r2.example/bestie-realestate/custom.jpg' })
		]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.src).toBe('https://r2.example/bestie-realestate/custom.jpg');
	});

	// Format-option merge — increment 2 (descriptor defaults + per-binding formatOptions).
	it('TC-RB-19 merges binding.formatOptions over the descriptor defaults (override wins)', () => {
		const layer = withBindings(makeTextLayer(), [
			bind({ token: 'price', format: 'currency:THB', formatOptions: { notation: 'short' } })
		]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		// The override reached descriptor.format → compact, and DIFFERS from the default render.
		expect(out.content).toBe(testFormatters.currency.format(28_500_000, { notation: 'short' }, 'THB'));
		expect(out.content).not.toBe(testFormatters.currency.format(28_500_000, { notation: 'full' }, 'THB'));
	});

	it('TC-RB-20 renders the descriptor defaults when the binding has no formatOptions (back-compat)', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'price', format: 'currency:THB' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		// Byte-identical to the descriptor's default-notation output AND to the pre-increment string.
		expect(out.content).toBe(testFormatters.currency.format(28_500_000, { notation: 'full' }, 'THB'));
		expect(out.content).toBe('฿28,500,000');
	});

	it('TC-RB-21 passes the raw String value through for an unknown formatter name', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'price', format: 'mystery:x' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('28500000');
	});

	// INC-A — value-type widening (SC-INCA-1): a valueType:'string' descriptor receives the RAW
	// token string, not Number(raw)=NaN. 'upper' uppercases, so a non-numeric title proves it.
	it('TC-RB-22 passes the RAW string to a valueType:string formatter (not Number-coerced)', () => {
		const layer = withBindings(makeTextLayer(), [bind({ token: 'title', format: 'upper' })]);
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('SUNSET RIDGE VILLA');
	});
});

// INC-C — content_template: a layer's content is interpolated from a template (text + tokens),
// resolved via the SAME token→formatter step as a binding (so currency/area/enum formatting works).
describe('resolveBindings — content_template (INC-C)', () => {
	it('TC-RB-23 resolves a template into content (literal text + formatted tokens)', () => {
		const layer = makeTextLayer({
			content_template: [
				{ type: 'text', text: 'Welcome to ' },
				{ type: 'token', token: 'title' },
				{ type: 'text', text: ' — ' },
				{ type: 'token', token: 'price', format: 'currency:THB' }
			]
		});
		const out = resolveBindings(layer, propertySnapshot, testFormatters);
		expect(out.content).toBe('Welcome to Sunset Ridge Villa — ฿28,500,000');
	});

	it('TC-RB-24 a missing template token resolves to empty (graceful, never "undefined")', () => {
		const layer = makeTextLayer({
			content_template: [
				{ type: 'text', text: 'A ' },
				{ type: 'token', token: 'nope' },
				{ type: 'text', text: ' B' }
			]
		});
		expect(resolveBindings(layer, propertySnapshot, testFormatters).content).toBe('A  B');
	});

	it('TC-RB-25 a template takes precedence over a content binding (mutually exclusive)', () => {
		const layer = withBindings(makeTextLayer({ content_template: [{ type: 'text', text: 'TPL' }] }), [
			{ property: 'content', sourceAlias: 'primary', token: 'title' }
		]);
		expect(resolveBindings(layer, propertySnapshot, testFormatters).content).toBe('TPL');
	});

	it('TC-RB-26 input layer is never mutated', () => {
		const layer = makeTextLayer({ content_template: [{ type: 'token', token: 'title' }] });
		const before = layer.content;
		resolveBindings(layer, propertySnapshot, testFormatters);
		expect(layer.content).toBe(before);
	});
});

// resolveTemplateContent — cross-source templates (refinement #1). A template token may carry an
// optional '<alias>.' prefix to pull from a SPECIFIC source; a bare token falls back to the default.
// This is the single resolver the artboard (resolveFrame), export, AND the Inspector's flatten-to-
// text path all share — so a multi-source template never diverges between them.
describe('resolveTemplateContent (cross-source templates)', () => {
	const byAlias = new Map<string, SourceSnapshot>([
		['primary', { title: 'Hillside Villa', price: 9_000_000 }],
		['secondary', { title: 'Wellness Biz', price: 28_500_000 }]
	]);

	it('TC-TC-01 routes each token to its OWN source by the <alias>. prefix', () => {
		const tpl: Template = [
			{ type: 'token', token: 'primary.title' },
			{ type: 'text', text: ' — ' },
			{ type: 'token', token: 'secondary.price', format: 'currency:THB' }
		];
		// title ← primary, price ← secondary (NOT primary's 9,000,000) — proves no single-source blanking.
		expect(resolveTemplateContent(tpl, byAlias, testFormatters, 'primary')).toBe('Hillside Villa — ฿28,500,000');
	});

	it('TC-TC-02 a BARE (unqualified) token resolves against the default alias', () => {
		const tpl: Template = [{ type: 'token', token: 'title' }];
		expect(resolveTemplateContent(tpl, byAlias, testFormatters, 'primary')).toBe('Hillside Villa');
	});

	it('TC-TC-03 an unknown-alias prefix is a field path on the default source (→ absent → "")', () => {
		// 'nope' is not a known source, so the WHOLE 'nope.title' is read as a field on 'primary'.
		const tpl: Template = [{ type: 'token', token: 'nope.title' }];
		expect(resolveTemplateContent(tpl, byAlias, testFormatters, 'primary')).toBe('');
	});
});
