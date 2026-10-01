// Shared drag-and-drop contract for bind-to-canvas (G8, extended in D-1). Draggable
// producers carry their payload under a kind-specific MIME; CanvasStage reads it on drop
// and applies the drop matrix:
//   image payload  →  over an image layer: RELINK its src · anywhere else: place a NEW
//                     bound image layer at the drop point.
//   field payload  →  over a text-bindable layer (text/callout): REBIND its content ·
//                     anywhere else: place a NEW bound text layer at the drop point.
//   preset payload →  place the preset at the drop point, whatever is under it (a new
//                     component is never a replace).
// During dragover only `dataTransfer.types` is readable (HTML5 dnd), so the MIME constant
// doubles as the kind discriminator for the replace-target highlight. One constant +
// payload type per kind so producers and the consumer never drift on a magic string.
import type { FieldDescriptor } from '@sbx/canvas-kit';

/** Custom MIME for the image-bind drag payload (kept off text/* so unrelated drops are inert). */
export const IMAGE_BIND_MIME = 'application/x-sbx-imagebind';

/** The dragged image's positional token + the source alias it belongs to. */
export interface ImageBindDrag {
	token: string;
	alias: string;
}

/** Custom MIME for the field-bind drag payload (a Texts-panel chip). */
export const FIELD_BIND_MIME = 'application/x-sbx-fieldbind';

/** The dragged field descriptor (token + label + format/fallback travel whole, so the
 *  drop side can bind without a registry lookup) + its source alias (undefined = the
 *  Active source, same default as the chip-click path). */
export interface FieldBindDrag {
	field: FieldDescriptor;
	alias?: string;
}

/** Custom MIME for the palette-preset drag payload (a ComponentCard — D-2). */
export const PRESET_MIME = 'application/x-sbx-preset';

/** The dragged palette preset id — the drop side materialises it at the drop point
 *  (same factory as click-insert, which keeps placing at the page centre). A preset
 *  is a NEW component, never a replace target: no highlight, place-only. */
export interface PresetDrag {
	presetId: string;
}

/** Producer helper — fill the DataTransfer with a preset payload (copy-drag), so the
 *  panels that render ComponentCards never hand-roll the MIME/serialisation. */
export function startPresetDrag(event: DragEvent, presetId: string): void {
	if (!event.dataTransfer) return;
	const payload: PresetDrag = { presetId };
	event.dataTransfer.setData(PRESET_MIME, JSON.stringify(payload));
	event.dataTransfer.effectAllowed = 'copy';
}
