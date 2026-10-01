// Media Canvas field-type model (INC-B). A layer property is edited in exactly one of three
// kinds; the Inspector reads this to lock a linked field (so a manual edit can't silently
// override + stale the link) and to label the field's type.

import type { Layer } from './types/layer.js';

/** The editing kind of a layer property:
 *  • 'text'     — a free static value the user types directly.
 *  • 'linked'   — driven by a source binding: LOCKED from direct typing (edit display via the
 *                 Format knobs, or convert to text by unlinking — which keeps the resolved value).
 *  • 'template' — driven by a text template that interpolates source tokens (INC-C composer). */
export type CanvasFieldKind = 'text' | 'linked' | 'template';

/** Derive the field kind of a layer property: a 'content' template → 'template' (INC-C); else a
 *  binding → 'linked'; else 'text'. Template wins over a binding (mutually exclusive by design).
 *  Pure: reads only the layer, so the Inspector + tests share one definition. */
export function fieldKindFor(layer: Layer, property: string): CanvasFieldKind {
	if (property === 'content' && layer.content_template && layer.content_template.length > 0) return 'template';
	const binding = layer.bindings?.find((b) => b.property === property);
	if (!binding) return 'text';
	return 'linked';
}
