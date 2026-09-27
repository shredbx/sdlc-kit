// Named fixture builders for resolver tests. The repo's TS convention is inline
// vitest with co-located builders (matches @sbx/units, land-canvas) — typed,
// reusable, never raw magic literals scattered through the tests.

import type { Layer } from './types/layer.js';
import type { AnimationTrack } from './types/animation.js';
import type { Binding } from './types/binding.js';
import type { SourceSnapshot, FormatterRegistry } from './types/source.js';

/** A minimal valid text layer; override any field. */
export function makeTextLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 't1',
		type: 'text',
		name: 'Text',
		visible: true,
		locked: false,
		x: 0,
		y: 0,
		content: '',
		opacity: 1,
		...overrides
	};
}

/** A minimal valid image layer; override any field. */
export function makeImageLayer(overrides: Partial<Layer> = {}): Layer {
	return {
		id: 'i1',
		type: 'image',
		name: 'Image',
		visible: true,
		locked: false,
		x: 0,
		y: 0,
		width: 100,
		height: 100,
		src: '',
		opacity: 1,
		...overrides
	};
}

/** Attach animation tracks to a layer. */
export function withAnimations(layer: Layer, animations: AnimationTrack[]): Layer {
	return { ...layer, animations };
}

/** Attach bindings to a layer. */
export function withBindings(layer: Layer, bindings: Binding[]): Layer {
	return { ...layer, bindings };
}

/** A representative BR-property snapshot (nested → exercises dotted-token paths). */
export const propertySnapshot: SourceSnapshot = {
	title: 'Sunset Ridge Villa',
	price: 28_500_000,
	/** land size in m² — exercises an area formatter (1600 m² = 1 rai). */
	landSize: 1600,
	coverImage: { url: 'https://r2.example/bestie-realestate/cover-1.jpg' },
	rooms: { bedrooms: 4, bathrooms: 5 }
};

/** Test formatter registry — proves resolveBindings routes by hint name AND merges
 *  per-binding formatOptions over the descriptor defaults, with NO dependency on
 *  @sbx/units (the real area/currency descriptors are wired in BR, slice 4). The
 *  resolver coerces the raw token to a number before calling format(). */
export const testFormatters: FormatterRegistry = {
	// currency — `arg` is the ISO code (orthogonal to the knobs). The `notation` knob
	// exercises the merge: default 'full' reproduces the prior flat output ('฿28,500,000'),
	// a binding override 'short' switches to compact ('฿28.5M').
	currency: {
		name: 'currency',
		sample: 2_500_000,
		options: [{ key: 'notation', label: 'Notation', choices: ['full', 'short'], default: 'full' }],
		format: (value: number, opts, arg) => {
			const sym = arg === 'THB' ? '฿' : '';
			const num =
				opts.notation === 'short'
					? value.toLocaleString('en-US', { notation: 'compact', maximumFractionDigits: 1 })
					: value.toLocaleString('en-US');
			return `${sym}${num}`;
		}
	},
	// area — unit comes from the legacy hint `arg` ('area:rai' → 'rai'); no injected unit
	// default, so a binding with no formatOptions renders identically to before ('1 rai').
	// 1 rai = 1600 m² (mirrors @sbx/units fromSqm for the test only).
	area: {
		name: 'area',
		sample: 1600,
		options: [],
		format: (value: number, _opts, arg) => (arg === 'rai' ? `${value / 1600} rai` : `${value} m²`)
	},
	// upper — a STRING-valued formatter (INC-A): valueType 'string' makes the resolver pass the
	// RAW token string to format() (not Number(raw) → NaN). Proves the string path end-to-end.
	upper: {
		name: 'upper',
		valueType: 'string',
		sample: 'sample',
		options: [],
		format: (value) => String(value).toUpperCase()
	}
};
