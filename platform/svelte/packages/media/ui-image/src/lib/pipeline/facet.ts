// Per-facet client-side resize ceiling (long edge, px) — the image-size-policy
// applied BEFORE upload so the network payload stays bounded. The R2 image
// function also resizes server-side; this is the client cap.
//
//   avatar / photo / qr / logo  → 512   (headshots, QR originals, brand mark master)
//   cover / gallery / hero / watermark → 2048  (property photos, hero covers)
//
// A consumer may still override via ImagePicker's `maxDimension` prop; this is
// the default when none is supplied.

import type { ImageFacet } from './types';

const FACET_MAX_DIMENSION: Record<ImageFacet, number> = {
	avatar: 512,
	photo: 512,
	qr: 512,
	logo: 512,
	cover: 2048,
	gallery: 2048,
	hero: 2048,
	watermark: 2048
};

export function facetMaxDimension(facet: ImageFacet): number {
	return FACET_MAX_DIMENSION[facet] ?? 2048;
}
