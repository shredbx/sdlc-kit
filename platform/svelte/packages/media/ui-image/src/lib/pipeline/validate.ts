// Magic byte detection + file size validation.
// Runs synchronously on the raw File/ArrayBuffer before any async processing.

import { ALLOWED_TYPES_DEFAULT } from './types';

const SIGNATURES: Array<{ format: string; bytes: number[]; offset?: number; extra?: { bytes: number[]; offset: number } }> = [
	{ format: 'image/jpeg', bytes: [0xFF, 0xD8, 0xFF] },
	{ format: 'image/png', bytes: [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A] },
	{ format: 'image/webp', bytes: [0x52, 0x49, 0x46, 0x46], extra: { bytes: [0x57, 0x45, 0x42, 0x50], offset: 8 } },
	{ format: 'image/avif', bytes: [0x66, 0x74, 0x79, 0x70], offset: 4 },
	{ format: 'image/heic', bytes: [0x66, 0x74, 0x79, 0x70], offset: 4 },
];

export function detectFormat(header: Uint8Array): string | null {
	for (const sig of SIGNATURES) {
		const offset = sig.offset ?? 0;
		if (header.length < offset + sig.bytes.length) continue;

		const match = sig.bytes.every((b, i) => header[offset + i] === b);
		if (!match) continue;

		if (sig.extra) {
			const extraMatch = sig.extra.bytes.every((b, i) => header[sig.extra!.offset + i] === b);
			if (!extraMatch) continue;
		}

		// HEIC vs AVIF: both use ftyp box — disambiguate by brand
		if (sig.format === 'image/avif' && header.length >= 12) {
			const brand = String.fromCharCode(...header.slice(8, 12));
			if (brand === 'avif' || brand === 'avis' || brand === 'mif1') return 'image/avif';
			if (brand === 'heic' || brand === 'heix' || brand === 'heim') return 'image/heic';
			return 'image/heic';
		}

		return sig.format;
	}
	return null;
}

export function validateMagicBytes(header: Uint8Array, allowedTypes: string[]): string | null {
	if (header.length === 0) return 'Empty file (0 bytes)';
	const detected = detectFormat(header);
	if (!detected) return 'Unrecognized file type — not a valid image';
	if (!allowedTypes.includes(detected)) return `File type ${detected} is not allowed`;
	return null;
}

export function validateFileSize(size: number, maxBytes: number): string | null {
	if (size > maxBytes) {
		const maxMB = (maxBytes / 1_000_000).toFixed(1);
		return `File size ${(size / 1_000_000).toFixed(1)}MB exceeds maximum ${maxMB}MB`;
	}
	return null;
}

export async function readFileHeader(file: File, bytes = 16): Promise<Uint8Array> {
	const slice = file.slice(0, bytes);
	const buffer = await slice.arrayBuffer();
	return new Uint8Array(buffer);
}

export function isHEIC(format: string | null): boolean {
	return format === 'image/heic' || format === 'image/heif';
}
