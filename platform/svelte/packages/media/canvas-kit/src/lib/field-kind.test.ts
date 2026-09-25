// TDD — @sbx/canvas-kit fieldKindFor (INC-B · the Media Canvas field-type model).
// A layer property is one of: 'text' (free static value), 'linked' (driven by a source
// binding — locked from direct typing), or 'template' (token-interpolating template — INC-C).
// The Inspector reads this to lock a linked field's editor + label its type.
//   TC-FK-01  an unbound property → 'text'
//   TC-FK-02  a bound property → 'linked'
//   TC-FK-03  a DIFFERENT property on a layer that binds 'content' stays 'text' (per-property)
import { describe, it, expect } from 'vitest';
import { makeTextLayer, withBindings } from './fixtures.js';
import { fieldKindFor } from './field-kind.js';

describe('fieldKindFor', () => {
	it('TC-FK-01 an unbound property is text', () => {
		expect(fieldKindFor(makeTextLayer(), 'content')).toBe('text');
	});

	it('TC-FK-02 a property driven by a binding is linked', () => {
		const layer = withBindings(makeTextLayer(), [
			{ property: 'content', sourceAlias: 'primary', token: 'title' }
		]);
		expect(fieldKindFor(layer, 'content')).toBe('linked');
	});

	it('TC-FK-03 the kind is per-property — an unbound sibling stays text', () => {
		const layer = withBindings(makeTextLayer(), [
			{ property: 'content', sourceAlias: 'primary', token: 'title' }
		]);
		expect(fieldKindFor(layer, 'src')).toBe('text');
	});

	it('TC-FK-04 a content_template makes content a template (winning over a binding)', () => {
		const base = makeTextLayer({ content_template: [{ type: 'token', token: 'title' }] });
		expect(fieldKindFor(base, 'content')).toBe('template');
		// Template wins even if a stale content binding lingers (they're mutually exclusive).
		const withBoth = withBindings(base, [{ property: 'content', sourceAlias: 'primary', token: 'title' }]);
		expect(fieldKindFor(withBoth, 'content')).toBe('template');
	});

	it('TC-FK-05 an empty content_template is not a template (falls through to text/linked)', () => {
		expect(fieldKindFor(makeTextLayer({ content_template: [] }), 'content')).toBe('text');
	});
});
