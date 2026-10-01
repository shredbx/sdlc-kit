// Upload processed image to the API via SvelteKit proxy route.
// The proxy forwards auth cookies — no token handling needed here.

import type { ImageFacet, ImageOwner, ProcessedImage } from './types';

// UploadOptions carries the owner-scoped key inputs the API requires
// (Decision #0271, object-storage-key-governance OSK-001/002):
//   - owner   — owning entity collection ('properties', 'agents', …) or 'system'
//               for ownerless brand assets.
//   - ownerId — the owning entity UUID. REQUIRED for every owner except 'system'
//               (system assets have no owner-id segment).
//   - facet   — the asset role within the owner ('cover', 'gallery', 'photo', …).
// The resulting key is {owner}/{ownerId}/{facet}/{objectId}.{ext} (or
// system/{facet}/{objectId}.{ext}). There is no 'purpose' field anymore.
export interface UploadOptions {
	owner: ImageOwner;
	ownerId?: string;
	facet: ImageFacet;
	altText?: string;
	endpoint?: string;
}

export interface UploadResult {
	ok: boolean;
	image?: {
		id: string;
		url: string;
		width: number;
		height: number;
		format: string;
		size: number;
	};
	error?: string;
}

export async function uploadImage(processed: ProcessedImage, options: UploadOptions): Promise<UploadResult> {
	const endpoint = options.endpoint ?? '/api/images';
	const ext = extensionFromFormat(processed.format);
	const filename = `upload.${ext}`;

	const form = new FormData();
	form.append('file', processed.blob, filename);
	form.append('owner', options.owner);
	// owner_id is omitted for the reserved 'system' namespace (no owner-id segment).
	if (options.ownerId) {
		form.append('owner_id', options.ownerId);
	}
	form.append('facet', options.facet);
	if (options.altText) {
		form.append('alt_text', options.altText);
	}

	const res = await fetch(endpoint, {
		method: 'POST',
		headers: { 'X-Requested-With': 'XMLHttpRequest' },
		body: form,
		credentials: 'include'
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		return { ok: false, error: body.error ?? `Upload failed: ${res.status}` };
	}

	const data = await res.json();
	return {
		ok: true,
		image: {
			id: data.id,
			url: data.url,
			width: data.width,
			height: data.height,
			format: data.format,
			size: data.size
		}
	};
}

function extensionFromFormat(format: string): string {
	switch (format) {
		case 'image/jpeg': return 'jpg';
		case 'image/png': return 'png';
		case 'image/webp': return 'webp';
		case 'image/avif': return 'avif';
		default: return 'bin';
	}
}
