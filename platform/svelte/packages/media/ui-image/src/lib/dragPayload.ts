import type { ImageData } from '@sbx/core-ui/types';

/** The dataTransfer stamp for a tile drag-start, or null when the tile is not
 *  draggable. Extracted from GalleryManager.handleDragStart so the payload
 *  contract is unit-testable: existing consumers (no dragText) must byte-match
 *  the shipped behavior — text/plain = reorder index, or the image id for a
 *  source-only pane — while a consumer-supplied dragText overrides ONLY the
 *  text/plain flavor. `application/x-image-id` is always stamped: cross-pane
 *  targets and cover slots key off it, and a text drop target simply ignores it. */
export interface DragStartPayload {
	effectAllowed: 'move' | 'copy';
	flavors: [format: string, data: string][];
}

export function dragStartPayload(input: {
	canReorder: boolean;
	dragSource: boolean;
	dragText?: (image: ImageData) => string;
	image: ImageData;
	index: number;
}): DragStartPayload | null {
	const { canReorder, dragSource, dragText, image, index } = input;
	if (!canReorder && !dragSource) return null;
	if (canReorder) {
		return {
			effectAllowed: 'move',
			flavors: [
				['text/plain', dragText ? dragText(image) : String(index)],
				['application/x-image-id', image.id]
			]
		};
	}
	return {
		effectAllowed: 'copy',
		flavors: [
			['application/x-image-id', image.id],
			['text/plain', dragText ? dragText(image) : image.id]
		]
	};
}
