// TDD — @sbx/canvas-kit resolveEditorFeatures (Slice A2.1 · Decision #0298).
// The watermark editor reuses @sbx/canvas-ui EditorShell via a generic `mode` prop
// ('design' | 'watermark'). This pure resolver maps the mode → a coherent feature
// bundle the shell threads to LeftRail / DocumentPanel / Inspector / TopToolbar.
// 'design' = the existing Media Canvas (full capability); 'watermark' = a static
// overlay-authoring surface (no animation/live-sources/components/map, +publish/copy).
//
//   TC-EM-01  'design' returns the FULL capability bundle (back-compat baseline)
//   TC-EM-02  'watermark' disables animation/liveSources/components/map/kindSelectable
//   TC-EM-03  'watermark' enables publish/copy/watermarkInspector/previewBackdrop
//   TC-EM-04  the two modes are exact mirror opposites on every flag (no accidental overlap)
//   TC-EM-05  the resolver is pure — same input, structurally equal output, no shared identity
import { describe, it, expect } from 'vitest';
import { resolveEditorFeatures, type EditorFeatures, type EditorMode } from './editor-mode.js';

const DESIGN: EditorFeatures = {
	animation: true,
	liveSources: true,
	components: true,
	map: true,
	kindSelectable: true,
	publish: false,
	copy: false,
	watermarkInspector: false,
	previewBackdrop: false,
	watermarkTiledPreview: false,
	documentTemplates: true,
	watermarkPhotoSizing: false
};

const WATERMARK: EditorFeatures = {
	animation: false,
	liveSources: false,
	components: false,
	map: false,
	kindSelectable: false,
	publish: true,
	copy: true,
	watermarkInspector: true,
	previewBackdrop: true,
	watermarkTiledPreview: true,
	documentTemplates: false,
	watermarkPhotoSizing: true
};

describe('resolveEditorFeatures', () => {
	it('TC-EM-01 design returns the full capability bundle (back-compat baseline)', () => {
		expect(resolveEditorFeatures('design')).toEqual(DESIGN);
	});

	it('TC-EM-02 watermark disables animation / live sources / components / map / kind selector', () => {
		const f = resolveEditorFeatures('watermark');
		expect(f.animation).toBe(false);
		expect(f.liveSources).toBe(false);
		expect(f.components).toBe(false);
		expect(f.map).toBe(false);
		expect(f.kindSelectable).toBe(false);
		expect(f.documentTemplates).toBe(false);
	});

	it('TC-EM-03 watermark enables publish / copy / watermark inspector / preview backdrop / tiled preview', () => {
		const f = resolveEditorFeatures('watermark');
		expect(f.publish).toBe(true);
		expect(f.copy).toBe(true);
		expect(f.watermarkInspector).toBe(true);
		expect(f.previewBackdrop).toBe(true);
		expect(f.watermarkTiledPreview).toBe(true);
		expect(f.watermarkPhotoSizing).toBe(true);
	});

	it('TC-EM-04 watermark resolves to the exact watermark bundle', () => {
		// Pins the whole watermark preset. (We deliberately do NOT assert the two modes
		// are per-flag opposites — that mirror is coincidental, not a contract: a future
		// flag could legitimately match in both modes.)
		expect(resolveEditorFeatures('watermark')).toEqual(WATERMARK);
	});

	it('TC-EM-05 is pure — equal value but a fresh object each call (no shared mutable identity)', () => {
		const a = resolveEditorFeatures('design');
		const b = resolveEditorFeatures('design');
		expect(a).toEqual(b);
		expect(a).not.toBe(b);
	});

	it('TC-EM-06 EditorMode covers exactly the two supported modes', () => {
		const modes: EditorMode[] = ['design', 'watermark'];
		for (const m of modes) expect(typeof resolveEditorFeatures(m)).toBe('object');
	});
});
