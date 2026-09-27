// Pure placement math for EventPreviewPopover — DOM-free so it unit-tests in isolation
// (the component only measures the card + viewport and delegates the geometry here). Given
// an anchor rect (the clicked event), the card size, and the viewport, it returns a fixed
// top/left that prefers BELOW the anchor, FLIPS above when there's no room, and CLAMPS to
// the viewport so the card never overflows. No anchor → centered (host opened it context-
// free, e.g. from a dashboard list with no element rect).
import type { PreviewAnchor } from '../types.js';

export interface Size {
	width: number;
	height: number;
}

export interface Placement {
	top: number;
	left: number;
}

function clamp(value: number, min: number, max: number): number {
	// max can fall below min on a viewport smaller than the card — keep `min` (top/left
	// edge visible) rather than letting Math.min win and pushing the card off-screen.
	return Math.max(min, Math.min(value, Math.max(min, max)));
}

export function placePopover(
	anchor: PreviewAnchor | undefined,
	card: Size,
	viewport: Size,
	margin = 8
): Placement {
	if (!anchor) {
		return {
			top: clamp((viewport.height - card.height) / 2, margin, viewport.height - card.height - margin),
			left: clamp((viewport.width - card.width) / 2, margin, viewport.width - card.width - margin)
		};
	}
	let top = anchor.y + anchor.height + margin; // prefer just below the event
	// Flip above only when there isn't room below AND there IS room above.
	const overflowsBelow = top + card.height > viewport.height - margin;
	const roomAbove = anchor.y - card.height - margin >= margin;
	if (overflowsBelow && roomAbove) {
		top = anchor.y - card.height - margin;
	}
	return {
		top: clamp(top, margin, viewport.height - card.height - margin),
		left: clamp(anchor.x, margin, viewport.width - card.width - margin)
	};
}
