import { describe, it, expect } from 'vitest';
import { dragStartPayload } from './dragPayload';
import type { ImageData } from '@sbx/core-ui/types';

// 2607-100 SC06 guard: without dragText, the payload set must byte-match the
// behavior GalleryManager shipped with (text/plain = reorder index, or the image
// id for a source-only pane) — plus the new dragText override path.

const img = {
	id: 'img-1',
	url: 'https://media.example.com/g/img-1.webp',
	width: 0,
	height: 0,
	alt_text: 'Pool',
	purpose: 'gallery',
	sort_order: 0
} as ImageData;

function flavors(p: NonNullable<ReturnType<typeof dragStartPayload>>): Record<string, string> {
	return Object.fromEntries(p.flavors);
}

describe('dragStartPayload', () => {
	// SUCCESS — the two legacy modes, unchanged (SC06)
	it('reorder mode without dragText stamps the index (legacy byte-match)', () => {
		const p = dragStartPayload({ canReorder: true, dragSource: false, image: img, index: 3 });
		expect(p).not.toBeNull();
		expect(p!.effectAllowed).toBe('move');
		expect(flavors(p!)).toEqual({
			'text/plain': '3',
			'application/x-image-id': 'img-1'
		});
	});

	it('source-only mode without dragText stamps the image id (legacy byte-match)', () => {
		const p = dragStartPayload({ canReorder: false, dragSource: true, image: img, index: 0 });
		expect(p).not.toBeNull();
		expect(p!.effectAllowed).toBe('copy');
		expect(flavors(p!)).toEqual({
			'application/x-image-id': 'img-1',
			'text/plain': 'img-1'
		});
	});

	// SUCCESS — dragText override in both modes
	it('reorder mode with dragText stamps the consumer payload, keeps the id flavor', () => {
		const p = dragStartPayload({
			canReorder: true,
			dragSource: false,
			dragText: (i) => `\n\n![${i.alt_text}](${i.url})\n\n`,
			image: img,
			index: 3
		});
		expect(p!.effectAllowed).toBe('move');
		expect(flavors(p!)).toEqual({
			'text/plain': '\n\n![Pool](https://media.example.com/g/img-1.webp)\n\n',
			'application/x-image-id': 'img-1'
		});
	});

	it('source-only mode with dragText stamps the consumer payload, keeps the id flavor', () => {
		const p = dragStartPayload({
			canReorder: false,
			dragSource: true,
			dragText: () => 'X',
			image: img,
			index: 0
		});
		expect(p!.effectAllowed).toBe('copy');
		expect(flavors(p!)).toEqual({
			'application/x-image-id': 'img-1',
			'text/plain': 'X'
		});
	});

	// EDGE
	it('dragText returning an empty string is stamped as-is (deterministic no-op drop)', () => {
		const p = dragStartPayload({
			canReorder: true,
			dragSource: false,
			dragText: () => '',
			image: img,
			index: 0
		});
		expect(flavors(p!)['text/plain']).toBe('');
	});

	it('canReorder wins when both modes are set (reorder branch, index/dragText payload)', () => {
		const p = dragStartPayload({ canReorder: true, dragSource: true, image: img, index: 5 });
		expect(p!.effectAllowed).toBe('move');
		expect(flavors(p!)['text/plain']).toBe('5');
	});

	// FAILURE — not draggable
	it('returns null when neither canReorder nor dragSource is set', () => {
		expect(dragStartPayload({ canReorder: false, dragSource: false, image: img, index: 0 })).toBeNull();
	});

	it('returns null even with dragText when the tile is not draggable', () => {
		expect(
			dragStartPayload({
				canReorder: false,
				dragSource: false,
				dragText: () => 'X',
				image: img,
				index: 0
			})
		).toBeNull();
	});
});
