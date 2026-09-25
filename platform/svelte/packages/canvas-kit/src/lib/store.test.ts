// TDD RED — editor-state reducers (FDD4.PD.TEST_RED).
// Scenarios: MC-SC-01 (add element + select), MC-SC-02 (reorder z-order),
//            MC-SC-03 (edit a property in the inspector).
//
// CASE ENUMERATION
//   TC-ST-01 success  addLayer appends a text layer to the page (MC-SC-01)
//   TC-ST-02 success  addLayer selects the new layer (MC-SC-01)
//   TC-ST-03 edge     addLayer appends at the end (front of z-stack)
//   TC-ST-04 failure  addLayer to an unknown page id is a no-op
//   TC-ST-05 success  reorderLayer moves a layer to the front (MC-SC-02)
//   TC-ST-06 regress  reorderLayer preserves every layer's properties
//   TC-ST-07 failure  reorderLayer with an out-of-bounds index is a no-op
//   TC-ST-08 success  updateLayer sets font_size (MC-SC-03)
//   TC-ST-09 edge     updateLayer merges (other properties preserved)
//   TC-ST-10 failure  updateLayer with an unknown id is a no-op
//   TC-ST-11 regress  updateLayer does not mutate the input state
//   TC-ST-12 success  resizePage sets the page width + height (Resize dialog)
//   TC-ST-13 failure  resizePage with an unknown page id is a no-op
//   TC-ST-14 failure  resizePage with non-positive / NaN dims is a no-op
//   TC-ST-15 regress  resizePage does not mutate the input state
//   TC-ST-16 success  deleteLayer removes the target layer
//   TC-ST-17 failure  deleteLayer with an unknown id is a no-op
//   TC-ST-18 failure  deleteLayer is a no-op for a `system` layer
//   TC-ST-19 regress  deleteLayer does not mutate the input state
//   TC-ST-20 success  deleteLayer clears the selection when the deleted layer was selected
//   TC-ST-21 edge     deleteLayer keeps the selection when a DIFFERENT layer is deleted

import { describe, expect, it } from 'vitest';
import {
	addLayer,
	attachSource,
	bindLayer,
	clearBindingOverride,
	createBackgroundLayer,
	createDocument,
	createWatermarkDocument,
	deleteLayer,
	detachSource,
	ensureBackgroundLayers,
	isDefaultBackground,
	nextSourceAlias,
	reorderLayer,
	resizePage,
	selectLayer,
	setBackground,
	setBindingOverride,
	setDocumentMeta,
	setPageDuration,
	setSourceSnapshot,
	unbindLayer,
	updateLayer
} from './store.js';
import type { EditorState } from './store.js';
import { DEFAULT_WATERMARK_PARAMS } from './types/watermark.js';
import { DEFAULT_PREVIEW_BACKDROP } from './types/preview-backdrop.js';
import type { Document, SourceRef } from './types/document.js';
import type { Binding } from './types/binding.js';
import { makeImageLayer, makeTextLayer, withBindings } from './fixtures.js';

const doc = (...layerIds: string[]): Document => ({
	id: 'doc1',
	name: 'Untitled',
	pages: [
		{
			id: 'p1',
			width: 1080,
			height: 1350,
			layers: layerIds.map((id) => makeImageLayer({ id, name: id }))
		}
	]
});

const state = (...layerIds: string[]): EditorState => ({
	document: doc(...layerIds),
	selectedLayerId: null
});

describe('addLayer (MC-SC-01)', () => {
	it('TC-ST-01 appends a new text layer to the active page', () => {
		const out = addLayer(state(), 'p1', makeTextLayer({ id: 'txt1' }));
		const layers = out.document.pages[0].layers;
		expect(layers.map((l) => l.id)).toContain('txt1');
	});

	it('TC-ST-02 selects the newly added layer', () => {
		const out = addLayer(state(), 'p1', makeTextLayer({ id: 'txt1' }));
		expect(out.selectedLayerId).toBe('txt1');
	});

	it('TC-ST-03 appends at the end of the z-stack (front-most)', () => {
		const out = addLayer(state('a', 'b'), 'p1', makeTextLayer({ id: 'txt1' }));
		const layers = out.document.pages[0].layers;
		expect(layers[layers.length - 1].id).toBe('txt1');
	});

	it('TC-ST-04 is a no-op for an unknown page id', () => {
		const before = state('a');
		const out = addLayer(before, 'nope', makeTextLayer({ id: 'txt1' }));
		expect(out.document.pages[0].layers).toHaveLength(1);
	});
});

describe('reorderLayer (MC-SC-02)', () => {
	it('TC-ST-05 moves a layer to the front of the z-order', () => {
		const out = reorderLayer(state('a', 'b', 'c'), 'p1', 2, 0);
		expect(out.document.pages[0].layers.map((l) => l.id)).toEqual(['c', 'a', 'b']);
	});

	it('TC-ST-06 preserves every layer property when reordering', () => {
		const before = state('a', 'b', 'c');
		const out = reorderLayer(before, 'p1', 2, 0);
		expect(out.document.pages[0].layers.find((l) => l.id === 'a')).toEqual(
			before.document.pages[0].layers.find((l) => l.id === 'a')
		);
	});

	it('TC-ST-07 is a no-op for an out-of-bounds index', () => {
		const out = reorderLayer(state('a', 'b'), 'p1', 9, 0);
		expect(out.document.pages[0].layers.map((l) => l.id)).toEqual(['a', 'b']);
	});
});

describe('updateLayer (MC-SC-03)', () => {
	it('TC-ST-08 sets a property (font_size) on the target layer', () => {
		const s = { document: doc('a'), selectedLayerId: 'a' };
		const out = updateLayer(s, 'a', { font_size: 96 });
		expect(out.document.pages[0].layers[0].font_size).toBe(96);
	});

	it('TC-ST-09 merges the patch, preserving other properties', () => {
		const s: EditorState = {
			document: {
				id: 'd',
				name: 'n',
				pages: [{ id: 'p1', width: 1, height: 1, layers: [makeTextLayer({ id: 'a', content: 'keep' })] }]
			},
			selectedLayerId: 'a'
		};
		const out = updateLayer(s, 'a', { font_size: 40 });
		expect(out.document.pages[0].layers[0].content).toBe('keep');
		expect(out.document.pages[0].layers[0].font_size).toBe(40);
	});

	it('TC-ST-10 is a no-op for an unknown layer id', () => {
		const before = state('a');
		const out = updateLayer(before, 'ghost', { font_size: 12 });
		expect(out.document.pages[0].layers[0].font_size).toBeUndefined();
	});

	it('TC-ST-11 does not mutate the input state', () => {
		const before = state('a');
		updateLayer(before, 'a', { font_size: 50 });
		expect(before.document.pages[0].layers[0].font_size).toBeUndefined();
	});
});

describe('selectLayer', () => {
	it('clears the selection when passed null', () => {
		const out = selectLayer({ document: doc('a'), selectedLayerId: 'a' }, null);
		expect(out.selectedLayerId).toBeNull();
	});
});

describe('resizePage', () => {
	it('TC-ST-12 sets the page width and height', () => {
		const out = resizePage(state('a'), 'p1', 1080, 1080);
		expect(out.document.pages[0].width).toBe(1080);
		expect(out.document.pages[0].height).toBe(1080);
	});

	it('TC-ST-13 is a no-op for an unknown page id', () => {
		const out = resizePage(state('a'), 'nope', 1920, 1080);
		expect(out.document.pages[0].width).toBe(1080);
		expect(out.document.pages[0].height).toBe(1350);
	});

	it('TC-ST-14 is a no-op for non-positive or NaN dimensions', () => {
		const before = state('a');
		expect(resizePage(before, 'p1', 0, 1080).document.pages[0].height).toBe(1350);
		expect(resizePage(before, 'p1', -10, 1080).document.pages[0].width).toBe(1080);
		expect(resizePage(before, 'p1', 1080, Number.NaN).document.pages[0].width).toBe(1080);
	});

	it('TC-ST-15 does not mutate the input state', () => {
		const before = state('a');
		resizePage(before, 'p1', 800, 800);
		expect(before.document.pages[0].width).toBe(1080);
		expect(before.document.pages[0].height).toBe(1350);
	});
});

describe('deleteLayer', () => {
	it('TC-ST-16 removes the target layer from its page', () => {
		const out = deleteLayer(state('a', 'b', 'c'), 'b');
		expect(out.document.pages[0].layers.map((l) => l.id)).toEqual(['a', 'c']);
	});

	it('TC-ST-17 is a no-op for an unknown layer id', () => {
		const out = deleteLayer(state('a', 'b'), 'ghost');
		expect(out.document.pages[0].layers.map((l) => l.id)).toEqual(['a', 'b']);
	});

	it('TC-ST-18 is a no-op for a system layer', () => {
		const s: EditorState = {
			document: {
				id: 'd',
				name: 'n',
				pages: [{ id: 'p1', width: 1, height: 1, layers: [makeImageLayer({ id: 'bg', system: true })] }]
			},
			selectedLayerId: null
		};
		const out = deleteLayer(s, 'bg');
		expect(out.document.pages[0].layers.map((l) => l.id)).toEqual(['bg']);
	});

	it('TC-ST-19 does not mutate the input state', () => {
		const before = state('a', 'b');
		deleteLayer(before, 'a');
		expect(before.document.pages[0].layers.map((l) => l.id)).toEqual(['a', 'b']);
	});

	it('TC-ST-20 clears the selection when the deleted layer was selected', () => {
		const s = { document: doc('a', 'b'), selectedLayerId: 'b' };
		const out = deleteLayer(s, 'b');
		expect(out.selectedLayerId).toBeNull();
	});

	it('TC-ST-21 keeps the selection when a different layer is deleted', () => {
		const s = { document: doc('a', 'b'), selectedLayerId: 'a' };
		const out = deleteLayer(s, 'b');
		expect(out.selectedLayerId).toBe('a');
	});
});

describe('setPageDuration (timeline Page-duration control)', () => {
	it('TC-ST-22 sets the page duration in ms', () => {
		const out = setPageDuration(state('a'), 'p1', 7000);
		expect(out.document.pages[0].duration).toBe(7000);
	});

	it('TC-ST-23 is a no-op for an unknown page id', () => {
		const out = setPageDuration(state('a'), 'nope', 7000);
		expect(out.document.pages[0].duration).toBeUndefined();
	});

	it('TC-ST-24 is a no-op for non-positive / NaN durations', () => {
		const before = state('a');
		expect(setPageDuration(before, 'p1', 0)).toBe(before);
		expect(setPageDuration(before, 'p1', -1)).toBe(before);
		expect(setPageDuration(before, 'p1', Number.NaN)).toBe(before);
	});

	it('TC-ST-25 does not mutate the input state', () => {
		const before = state('a');
		setPageDuration(before, 'p1', 3000);
		expect(before.document.pages[0].duration).toBeUndefined();
	});
});

describe('createDocument', () => {
	it('TC-ST-26 seeds a single page with the 5s default duration (§12.7 / D19)', () => {
		const d = createDocument('Untitled', 1080, 1080);
		expect(d.pages).toHaveLength(1);
		expect(d.pages[0].duration).toBe(5000);
		// The only seeded layer is the system background (user directive 2026-06-06).
		expect(d.pages[0].layers.filter((l) => !l.system)).toEqual([]);
	});
});

describe('createWatermarkDocument (#0298)', () => {
	it('TC-ST-26b seeds a fully TRANSPARENT system background (the overlay starts clear)', () => {
		const d = createWatermarkDocument('Untitled watermark', 1600, 1200);
		const bg = d.pages[0].layers.find((l) => l.system);
		// alpha = 1 - transparency, so transparency:1 = fully transparent.
		expect(bg?.fill?.transparency).toBe(1);
	});

	it('TC-ST-26c is otherwise identical to createDocument (one page, 5s, bg-only)', () => {
		const d = createWatermarkDocument('Untitled watermark', 1600, 1200);
		expect(d.pages).toHaveLength(1);
		expect(d.pages[0].duration).toBe(5000);
		expect(d.pages[0].layers.filter((l) => !l.system)).toEqual([]);
		expect(d.pages[0].width).toBe(1600);
		expect(d.pages[0].height).toBe(1200);
	});

	it('TC-ST-26d does NOT mutate createDocument’s opaque default (no shared fill object)', () => {
		createWatermarkDocument('wm', 800, 600);
		const plain = createDocument('doc', 800, 600);
		const bg = plain.pages[0].layers.find((l) => l.system);
		expect(bg?.fill?.transparency).toBe(0);
	});
});

describe('setDocumentMeta (Settings title + kind; persistence)', () => {
	it('TC-ST-27 renames the document', () => {
		const out = setDocumentMeta(state('a'), { name: 'Listing — Hua Hin' });
		expect(out.document.name).toBe('Listing — Hua Hin');
	});

	it('TC-ST-28 sets the template flag (kind = template)', () => {
		const out = setDocumentMeta(state('a'), { isTemplate: true });
		expect(out.document.isTemplate).toBe(true);
	});

	it('TC-ST-29 ignores a blank / whitespace-only name (title stays non-blank)', () => {
		const out = setDocumentMeta(setDocumentMeta(state('a'), { name: 'Keep' }), { name: '   ' });
		expect(out.document.name).toBe('Keep');
	});

	it('TC-ST-30 patches only the keys present (name change keeps isTemplate)', () => {
		const seeded = setDocumentMeta(state('a'), { isTemplate: true });
		const out = setDocumentMeta(seeded, { name: 'Renamed' });
		expect(out.document.name).toBe('Renamed');
		expect(out.document.isTemplate).toBe(true);
	});

	it('TC-ST-31 does not mutate the input state', () => {
		const before = state('a');
		setDocumentMeta(before, { name: 'X', isTemplate: true });
		expect(before.document.name).toBe('Untitled');
		expect(before.document.isTemplate).toBeUndefined();
	});

	it('TC-ST-31b sets/clears the animated flag (R3), patched independently of name + kind', () => {
		const on = setDocumentMeta(state('a'), { animated: true });
		expect(on.document.animated).toBe(true);
		const off = setDocumentMeta(on, { animated: false });
		expect(off.document.animated).toBe(false);
	});

	// --- Watermark params (Slice A2.2 · #0298) — partial merge over the defaults ---
	it('TC-ST-31c seeds the full default params from a partial watermark patch (first edit)', () => {
		const out = setDocumentMeta(state('a'), { watermark: { opacity: 80 } });
		expect(out.document.watermark).toEqual({
			...DEFAULT_WATERMARK_PARAMS(),
			opacity: 80
		});
	});

	it('TC-ST-31d merges a later patch over the existing params (other knobs kept)', () => {
		const seeded = setDocumentMeta(state('a'), { watermark: { opacity: 80 } });
		const out = setDocumentMeta(seeded, { watermark: { tiling: 'grid', rotation: 45 } });
		expect(out.document.watermark).toEqual({
			...DEFAULT_WATERMARK_PARAMS(),
			opacity: 80,
			tiling: 'grid',
			rotation: 45
		});
	});

	it('TC-ST-31e leaves watermark untouched when not in the patch', () => {
		const seeded = setDocumentMeta(state('a'), { watermark: { theme: 'light' } });
		const out = setDocumentMeta(seeded, { name: 'Renamed' });
		expect(out.document.watermark?.theme).toBe('light');
		expect(out.document.name).toBe('Renamed');
	});

	it('TC-ST-31f does not mutate the input document watermark', () => {
		const seeded = setDocumentMeta(state('a'), { watermark: { opacity: 80 } });
		setDocumentMeta(seeded, { watermark: { opacity: 10 } });
		expect(seeded.document.watermark?.opacity).toBe(80);
	});

	// --- Preview backdrop (Slice A2.3 · #0298) — partial merge over the defaults.
	//     PREVIEW-ONLY: persists in the doc but is rendered as stage chrome, so it is
	//     structurally excluded from the export path (resolveFrame reads only layers). ---
	it('TC-ST-31g seeds the default (none) from a partial backdrop patch (first edit)', () => {
		const out = setDocumentMeta(state('a'), { previewBackdrop: { kind: 'color', color: '#101010' } });
		expect(out.document.previewBackdrop).toEqual({
			...DEFAULT_PREVIEW_BACKDROP(),
			kind: 'color',
			color: '#101010'
		});
	});

	it('TC-ST-31h merges a later patch over the existing backdrop (other face kept)', () => {
		const seeded = setDocumentMeta(state('a'), { previewBackdrop: { kind: 'color', color: '#101010' } });
		// Switch the kind to image but keep the cached colour from the earlier patch.
		const out = setDocumentMeta(seeded, { previewBackdrop: { kind: 'image', imageSrc: 'cdn://photo' } });
		expect(out.document.previewBackdrop).toEqual({
			kind: 'image',
			color: '#101010',
			imageSrc: 'cdn://photo'
		});
	});

	it('TC-ST-31i leaves the backdrop untouched when not in the patch', () => {
		const seeded = setDocumentMeta(state('a'), { previewBackdrop: { kind: 'color', color: '#abcabc' } });
		const out = setDocumentMeta(seeded, { name: 'Renamed' });
		expect(out.document.previewBackdrop?.kind).toBe('color');
		expect(out.document.previewBackdrop?.color).toBe('#abcabc');
		expect(out.document.name).toBe('Renamed');
	});

	it('TC-ST-31j does not mutate the input document backdrop', () => {
		const seeded = setDocumentMeta(state('a'), { previewBackdrop: { kind: 'color', color: '#111111' } });
		setDocumentMeta(seeded, { previewBackdrop: { color: '#999999' } });
		expect(seeded.document.previewBackdrop?.color).toBe('#111111');
	});
});

// --- Sources & data-binding (D20/D21 — MC-SC-06 attach, MC-SC-15 override+restore,
//     MC-SC-16 unlink-preserves, MC-SC-17 re-link) -----------------------------------
//   TC-ST-32..34  attachSource: add / replace-by-alias / append-new-alias
//   TC-ST-35..36  detachSource: remove-by-alias / unknown-no-op
//   TC-ST-37..38  setSourceSnapshot: fill / unknown-no-op
//   TC-ST-39..43  bindLayer: add / re-link-replace / unknown-no-op / multi-property / immutable
//   TC-ST-44..47  unbindLayer: remove / keep-value-as-static / drop-empty-array / no-op
//   TC-ST-48..49  setBindingOverride: set / no-op-when-unbound
//   TC-ST-50..51  clearBindingOverride: restore / no-op-when-no-override

const ref = (alias = 'primary', refId = 'prop-1'): SourceRef => ({
	id: `s-${alias}`,
	kind: 'property',
	refId,
	alias
});

const bind = (property: string, token: string, extra: Partial<Binding> = {}): Binding => ({
	property,
	sourceAlias: 'primary',
	token,
	...extra
});

/** A state with one text layer 'txt1' carrying the given bindings. */
const boundState = (...bindings: Binding[]): EditorState => ({
	document: {
		id: 'doc1',
		name: 'Untitled',
		pages: [
			{
				id: 'p1',
				width: 1080,
				height: 1350,
				layers: [withBindings(makeTextLayer({ id: 'txt1' }), bindings)]
			}
		]
	},
	selectedLayerId: null
});

const layerOf = (s: EditorState) => s.document.pages[0].layers.find((l) => l.id === 'txt1')!;

describe('attachSource (MC-SC-06)', () => {
	it('TC-ST-32 attaches a source ref under its alias', () => {
		const out = attachSource(state('a'), ref('primary'));
		expect(out.document.sources).toEqual([ref('primary')]);
	});

	it('TC-ST-33 replaces the ref when re-attaching the same alias', () => {
		const out = attachSource(attachSource(state('a'), ref('primary', 'prop-1')), ref('primary', 'prop-9'));
		expect(out.document.sources).toHaveLength(1);
		expect(out.document.sources?.[0].refId).toBe('prop-9');
	});

	it('TC-ST-33b refuses a BUILT-IN alias — a doc attach can never shadow an injected source (#0286)', () => {
		const s = state('a');
		const out = attachSource(s, ref('branding'));
		expect(out).toBe(s); // unchanged state, by reference (pure refusal)
		expect(out.document.sources ?? []).toHaveLength(0);
	});

	it('TC-ST-33c refuses a BUILT-IN kind under a normal alias — never persistable as a doc source (#0286)', () => {
		const s = state('a');
		const out = attachSource(s, { ...ref('primary'), kind: 'branding' });
		expect(out).toBe(s);
		expect(out.document.sources ?? []).toHaveLength(0);
	});

	it('TC-ST-34 appends a ref under a new alias', () => {
		const out = attachSource(attachSource(state('a'), ref('primary')), ref('secondary'));
		expect(out.document.sources?.map((s) => s.alias)).toEqual(['primary', 'secondary']);
	});
});

describe('detachSource', () => {
	it('TC-ST-35 removes the attached source by alias', () => {
		const out = detachSource(attachSource(state('a'), ref('primary')), 'primary');
		expect(out.document.sources).toEqual([]);
	});

	it('TC-ST-36 is a no-op for an unknown alias', () => {
		const before = attachSource(state('a'), ref('primary'));
		expect(detachSource(before, 'nope')).toBe(before);
	});

	it('TC-ST-52 leaves dangling bindings in place (fallback covers a deleted source)', () => {
		const seeded = attachSource(boundState(bind('content', 'title')), ref('primary'));
		const out = detachSource(seeded, 'primary');
		expect(out.document.sources).toEqual([]);
		expect(layerOf(out).bindings).toEqual([bind('content', 'title')]);
	});
});

describe('setSourceSnapshot (live re-resolve on open)', () => {
	it('TC-ST-37 fills the snapshot on the matching ref', () => {
		const out = setSourceSnapshot(attachSource(state('a'), ref('primary')), 'primary', { title: 'Villa' });
		expect(out.document.sources?.[0].snapshot).toEqual({ title: 'Villa' });
	});

	it('TC-ST-38 is a no-op for an unknown alias', () => {
		const before = attachSource(state('a'), ref('primary'));
		expect(setSourceSnapshot(before, 'nope', { title: 'x' })).toBe(before);
	});
});

describe('bindLayer (MC-SC-07 link / MC-SC-17 re-link)', () => {
	it('TC-ST-39 adds a binding to a layer', () => {
		const out = bindLayer(boundState(), 'txt1', bind('content', 'title'));
		expect(layerOf(out).bindings).toEqual([bind('content', 'title')]);
	});

	it('TC-ST-40 replaces the binding for the same property (re-link to a new field)', () => {
		const out = bindLayer(boundState(bind('content', 'title')), 'txt1', bind('content', 'price', { format: 'currency:THB' }));
		expect(layerOf(out).bindings).toEqual([bind('content', 'price', { format: 'currency:THB' })]);
	});

	it('TC-ST-41 is a no-op for an unknown layer id', () => {
		const before = boundState();
		expect(bindLayer(before, 'nope', bind('content', 'title'))).toBe(before);
	});

	it('TC-ST-42 keeps a binding on a different property (multi-bind)', () => {
		const out = bindLayer(boundState(bind('content', 'title')), 'txt1', bind('src', 'coverImage.url'));
		expect(layerOf(out).bindings?.map((b) => b.property)).toEqual(['content', 'src']);
	});

	it('TC-ST-43 does not mutate the input state', () => {
		const before = boundState();
		bindLayer(before, 'txt1', bind('content', 'title'));
		expect(layerOf(before).bindings).toEqual([]);
	});
});

describe('unbindLayer (MC-SC-16 unlink preserves the current value)', () => {
	it('TC-ST-44 removes the binding for the property', () => {
		const out = unbindLayer(boundState(bind('content', 'title')), 'txt1', 'content');
		expect(layerOf(out).bindings).toBeUndefined();
	});

	it('TC-ST-45 writes the supplied value as the new static property (nothing changes visually)', () => {
		const out = unbindLayer(boundState(bind('content', 'title')), 'txt1', 'content', 'Sunset Ridge Villa');
		expect(layerOf(out).content).toBe('Sunset Ridge Villa');
		expect(layerOf(out).bindings).toBeUndefined();
	});

	it('TC-ST-46 keeps other bindings and drops only the targeted one', () => {
		const out = unbindLayer(boundState(bind('content', 'title'), bind('src', 'coverImage.url')), 'txt1', 'content');
		expect(layerOf(out).bindings?.map((b) => b.property)).toEqual(['src']);
	});

	it('TC-ST-47 is a no-op when the property is not bound', () => {
		const before = boundState(bind('content', 'title'));
		expect(unbindLayer(before, 'txt1', 'src')).toBe(before);
	});
});

describe('setBindingOverride / clearBindingOverride (MC-SC-15 override + Restore)', () => {
	it('TC-ST-48 sets a manual override on the binding', () => {
		const out = setBindingOverride(boundState(bind('content', 'title')), 'txt1', 'content', 'Tuned copy');
		expect(layerOf(out).bindings?.[0].override).toBe('Tuned copy');
	});

	it('TC-ST-49 is a no-op when the property is not bound', () => {
		const before = boundState(bind('content', 'title'));
		expect(setBindingOverride(before, 'txt1', 'src', 'x')).toBe(before);
	});

	it('TC-ST-50 clears the override (Restore → live value)', () => {
		const seeded = setBindingOverride(boundState(bind('content', 'title')), 'txt1', 'content', 'Tuned copy');
		const out = clearBindingOverride(seeded, 'txt1', 'content');
		expect(layerOf(out).bindings?.[0].override).toBeUndefined();
	});

	it('TC-ST-51 is a no-op when there is no override to clear', () => {
		const before = boundState(bind('content', 'title'));
		expect(clearBindingOverride(before, 'txt1', 'content')).toBe(before);
	});

	it('TC-ST-53 unlink keeps the overridden value when the UI passes it (override → static)', () => {
		// The UI resolves override-first, so the value handed to unbindLayer is the override.
		const seeded = setBindingOverride(boundState(bind('content', 'title')), 'txt1', 'content', 'Tuned copy');
		const out = unbindLayer(seeded, 'txt1', 'content', 'Tuned copy');
		expect(layerOf(out).bindings).toBeUndefined();
		expect(layerOf(out).content).toBe('Tuned copy');
	});
});

// --- nextSourceAlias (G6a multi-source attach) ----------------------------
//   TC-AL-01 success  no sources → 'primary' (the Active/preview anchor)
//   TC-AL-02 success  walks the ladder in order (primary → secondary → …)
//   TC-AL-03 edge     skips aliases already in use (gaps, any order)
//   TC-AL-04 edge     ladder exhausted (6 used) → 'source-7'
//   TC-AL-05 failure  mid-list detach + re-attach never returns an in-use alias (no overwrite)
describe('nextSourceAlias', () => {
	it('TC-AL-01: returns primary when nothing is attached', () => {
		expect(nextSourceAlias([])).toBe('primary');
	});

	it('TC-AL-02: walks the ladder in order', () => {
		expect(nextSourceAlias(['primary'])).toBe('secondary');
		expect(nextSourceAlias(['primary', 'secondary'])).toBe('tertiary');
	});

	it('TC-AL-03: skips aliases already in use regardless of order', () => {
		// 'primary' detached, 'secondary' kept → the next free ladder slot is 'primary'.
		expect(nextSourceAlias(['secondary'])).toBe('primary');
	});

	it('TC-AL-04: falls back to source-N once the ladder is exhausted', () => {
		const ladder = ['primary', 'secondary', 'tertiary', 'quaternary', 'quinary', 'senary'];
		expect(nextSourceAlias(ladder)).toBe('source-7');
	});

	it('TC-AL-05: never returns an alias already in use after a mid-list detach (no silent overwrite)', () => {
		// Ladder full + source-7, source-8 attached, then source-7 detached. A count-based
		// `source-${len+1}` would return the still-attached 'source-8' and overwrite it.
		const ladder = ['primary', 'secondary', 'tertiary', 'quaternary', 'quinary', 'senary'];
		const used = [...ladder, 'source-8']; // 7 entries; 'source-7' was detached
		const next = nextSourceAlias(used);
		expect(used).not.toContain(next);
		expect(next).toBe('source-7');
	});
});

// ── Background layer (user directive 2026-06-06; land-canvas parity) ────────────
//   TC-ST-54 success  createBackgroundLayer = shape/system/locked, white fill, bg_mode color, page-sized
//   TC-ST-55 success  ensureBackgroundLayers prepends a default background to a page without one
//   TC-ST-56 regress  ensureBackgroundLayers is idempotent (same state object when complete)
//   TC-ST-57 edge     ensureBackgroundLayers moves an out-of-position background back to index 0
//   TC-ST-58 success  setBackground color: fill + bg_mode, strips src/mime_type + the src binding
//   TC-ST-59 success  setBackground image: type image + src + bg_mode image (binding left to bindLayer)
//   TC-ST-60 success  setBackground map: type image + bg_mode map
//   TC-ST-61 regress  setBackground preserves id/system/locked/geometry; unknown page = no-op
//   TC-ST-62 success  isDefaultBackground true ONLY for the untouched white color background
//   TC-ST-63 failure  reorderLayer never moves the background nor displaces it from index 0
//   TC-ST-64 success  resizePage pins the background to the new page size (land-canvas parity)

describe('background layer (land-canvas parity)', () => {
	const bgState = (...layerIds: string[]): EditorState => {
		const s = state(...layerIds);
		return ensureBackgroundLayers(s);
	};

	it('TC-ST-54 createBackgroundLayer builds the canonical system background', () => {
		const bg = createBackgroundLayer('p1', 1080, 1350);
		expect(bg.id).toBe('bg-p1');
		expect(bg.type).toBe('shape');
		expect(bg.shape_type).toBe('rectangle');
		expect(bg.system).toBe(true);
		expect(bg.locked).toBe(true);
		expect(bg.bg_mode).toBe('color');
		expect(bg.fill?.color).toBe('#ffffff');
		expect([bg.x, bg.y, bg.width, bg.height]).toEqual([0, 0, 1080, 1350]);
	});

	it('TC-ST-55 ensureBackgroundLayers prepends a default background when missing', () => {
		const out = ensureBackgroundLayers(state('a', 'b'));
		const layers = out.document.pages[0].layers;
		expect(layers).toHaveLength(3);
		expect(layers[0].system).toBe(true);
		expect(layers[0].bg_mode).toBe('color');
		expect(layers.map((l) => l.id).slice(1)).toEqual(['a', 'b']);
	});

	it('TC-ST-56 ensureBackgroundLayers returns the SAME state when already complete', () => {
		const once = ensureBackgroundLayers(state('a'));
		expect(ensureBackgroundLayers(once)).toBe(once);
	});

	it('TC-ST-57 ensureBackgroundLayers moves an out-of-position background to index 0', () => {
		const s = state('a');
		s.document.pages[0].layers.push(createBackgroundLayer('p1', 1080, 1350));
		const out = ensureBackgroundLayers(s);
		const layers = out.document.pages[0].layers;
		expect(layers[0].system).toBe(true);
		expect(layers.map((l) => l.id)).toEqual(['bg-p1', 'a']);
	});

	it('TC-ST-58 setBackground color sets the fill and strips src + its binding', () => {
		let s = bgState('a');
		s = setBackground(s, 'p1', { mode: 'image', src: 'https://x/img.jpg' });
		s = bindLayer(s, 'bg-p1', { property: 'src', sourceAlias: 'primary', token: 'coverImage.url' });
		const out = setBackground(s, 'p1', { mode: 'color', fill: { color: '#102030', transparency: 0 } });
		const bg = out.document.pages[0].layers[0];
		expect(bg.bg_mode).toBe('color');
		expect(bg.type).toBe('shape');
		expect(bg.fill?.color).toBe('#102030');
		expect(bg.src).toBeUndefined();
		expect(bg.bindings ?? []).toHaveLength(0);
	});

	it('TC-ST-59 setBackground image sets type/src/bg_mode', () => {
		const out = setBackground(bgState('a'), 'p1', { mode: 'image', src: 'https://x/img.jpg' });
		const bg = out.document.pages[0].layers[0];
		expect(bg.type).toBe('image');
		expect(bg.bg_mode).toBe('image');
		expect(bg.src).toBe('https://x/img.jpg');
	});

	it('TC-ST-60 setBackground map sets bg_mode map on an image-typed background', () => {
		const out = setBackground(bgState('a'), 'p1', { mode: 'map', src: 'https://maps/static.png' });
		const bg = out.document.pages[0].layers[0];
		expect(bg.type).toBe('image');
		expect(bg.bg_mode).toBe('map');
	});

	it('TC-ST-61 setBackground preserves identity + geometry; unknown page is a no-op', () => {
		const before = bgState('a');
		const out = setBackground(before, 'p1', { mode: 'image', src: 'x' });
		const bg = out.document.pages[0].layers[0];
		expect([bg.id, bg.system, bg.locked]).toEqual(['bg-p1', true, true]);
		expect([bg.x, bg.y, bg.width, bg.height]).toEqual([0, 0, 1080, 1350]);
		expect(setBackground(before, 'nope', { mode: 'image', src: 'x' })).toBe(before);
	});

	it('TC-ST-62 isDefaultBackground is true only for the untouched white color background', () => {
		const fresh = bgState('a');
		expect(isDefaultBackground(fresh.document.pages[0].layers[0])).toBe(true);
		const colored = setBackground(fresh, 'p1', { mode: 'color', fill: { color: '#102030', transparency: 0 } });
		expect(isDefaultBackground(colored.document.pages[0].layers[0])).toBe(false);
		const imaged = setBackground(fresh, 'p1', { mode: 'image', src: 'x' });
		expect(isDefaultBackground(imaged.document.pages[0].layers[0])).toBe(false);
	});

	it('TC-ST-63 reorderLayer refuses to move or displace the background', () => {
		const s = bgState('a', 'b');
		expect(reorderLayer(s, 'p1', 0, 2)).toBe(s); // moving the bg itself
		expect(reorderLayer(s, 'p1', 2, 0)).toBe(s); // displacing it from index 0
		// normal reorders above the background still work
		const moved = reorderLayer(s, 'p1', 2, 1);
		expect(moved.document.pages[0].layers.map((l) => l.id)).toEqual(['bg-p1', 'b', 'a']);
	});

	it('TC-ST-64 resizePage pins the background to the new page size', () => {
		const out = resizePage(bgState('a'), 'p1', 1920, 1080);
		const bg = out.document.pages[0].layers[0];
		expect([bg.width, bg.height]).toEqual([1920, 1080]);
	});

	it('TC-ST-65 createDocument seeds the background layer in its page', () => {
		const d = createDocument('Untitled', 1080, 1080);
		expect(d.pages[0].layers).toHaveLength(1);
		expect(d.pages[0].layers[0].system).toBe(true);
		expect(isDefaultBackground(d.pages[0].layers[0])).toBe(true);
	});

	it('TC-ST-66 an image↔map mode switch drops the stale src binding (no lying discriminator)', () => {
		let s = bgState('a');
		s = setBackground(s, 'p1', { mode: 'image', src: 'https://x/photo.jpg' });
		s = bindLayer(s, 'bg-p1', { property: 'src', sourceAlias: 'primary', token: 'images.gallery.0.url' });
		// photo binding must NOT survive under bg_mode 'map'…
		const toMap = setBackground(s, 'p1', { mode: 'map' });
		expect(toMap.document.pages[0].layers[0].bindings ?? []).toHaveLength(0);
		// …but a SAME-mode re-apply keeps it (the caller rebinds right after).
		const sameMode = setBackground(s, 'p1', { mode: 'image', src: 'https://x/other.jpg' });
		expect(sameMode.document.pages[0].layers[0].bindings).toHaveLength(1);
	});

	it('TC-ST-67 ensureBackgroundLayers normalizes EVERY page of a multi-page document', () => {
		const s = state('a');
		s.document.pages.push({ id: 'p2', width: 800, height: 600, layers: [makeImageLayer({ id: 'z' })] });
		const out = ensureBackgroundLayers(s);
		for (const page of out.document.pages) {
			expect(page.layers[0].system).toBe(true);
			expect([page.layers[0].width, page.layers[0].height]).toEqual([page.width, page.height]);
		}
	});
});
