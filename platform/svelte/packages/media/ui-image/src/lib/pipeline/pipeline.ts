// Client-side image processing pipeline.
// Runs: validate → convert HEIC → resize + compress → return processed blob.
//
// Dependencies (must be installed by consuming app):
//   - heic2any: HEIC/HEIF → WebP conversion
//   - browser-image-compression: resize + compress in web worker
//
// The pipeline is config-driven. The consuming app defines configs per purpose
// (e.g. cover: 2048px, gallery: 4096px). This module is purpose-agnostic.

import type { PipelineConfig, PipelineResult, ProcessedImage, PipelineError } from './types';
import { ALLOWED_TYPES_DEFAULT } from './types';
import { readFileHeader, validateMagicBytes, validateFileSize, detectFormat, isHEIC } from './validate';

export async function processImage(file: File, config: PipelineConfig): Promise<PipelineResult> {
	const errors: PipelineError[] = [];
	const warnings: PipelineError[] = [];
	const originalSize = file.size;

	// STEP 1: Validate magic bytes
	const header = await readFileHeader(file);
	const allowedTypes = config.allowedTypes ?? ALLOWED_TYPES_DEFAULT;
	const magicErr = validateMagicBytes(header, allowedTypes);
	if (magicErr) {
		errors.push({ step: 'ValidateMagicBytes', message: magicErr, fatal: true });
		return { ok: false, errors, warnings };
	}

	// STEP 2: Validate file size (pre-processing — reject obviously oversized originals)
	const sizeErr = validateFileSize(file.size, config.maxBytes * 3);
	if (sizeErr) {
		errors.push({ step: 'ValidateFileSize', message: sizeErr, fatal: true });
		return { ok: false, errors, warnings };
	}

	let blob: Blob = file;
	let format = detectFormat(header) ?? file.type;

	// STEP 3: Convert HEIC/HEIF → WebP
	if (isHEIC(format)) {
		try {
			const heic2any = (await import('heic2any')).default;
			const converted = await heic2any({ blob: file, toType: 'image/webp', quality: config.quality });
			blob = Array.isArray(converted) ? converted[0] : converted;
			format = 'image/webp';
		} catch (e) {
			errors.push({
				step: 'ConvertHEIC',
				message: e instanceof Error ? e.message : 'HEIC conversion failed',
				fatal: true
			});
			return { ok: false, errors, warnings };
		}
	}

	// STEP 4: Resize + Compress (web worker via browser-image-compression)
	// Output format precedence:
	//   1. `config.outputFormat` if set — wins everything (e.g. BR forces WebP
	//      to stop PNG sources landing in R2 — see PipelineConfig docs).
	//   2. Otherwise: PNG-in stays PNG-out (preserves transparency for callers
	//      that depend on it), everything else → WebP (best size/quality).
	const outputFormat =
		config.outputFormat ?? (format === 'image/png' ? 'image/png' : 'image/webp');
	try {
		const imageCompression = (await import('browser-image-compression')).default;
		const compressed = await imageCompression(new File([blob], file.name, { type: format }), {
			maxWidthOrHeight: config.maxDimension,
			maxSizeMB: config.maxBytes / 1_000_000,
			useWebWorker: true,
			fileType: outputFormat,
			initialQuality: config.quality
		});
		blob = compressed;
		format = compressed.type;
	} catch (e) {
		warnings.push({
			step: 'ResizeCompress',
			message: e instanceof Error ? e.message : 'Compression failed — using original',
			fatal: false
		});
	}

	// STEP 5: Extract dimensions from processed blob
	const { width, height } = await extractDimensions(blob);

	// STEP 6: Final size check (post-processing)
	if (blob.size > config.maxBytes) {
		errors.push({
			step: 'FinalSizeCheck',
			message: `Processed image ${(blob.size / 1_000_000).toFixed(1)}MB still exceeds limit`,
			fatal: true
		});
		return { ok: false, errors, warnings };
	}

	const image: ProcessedImage = {
		blob,
		format,
		width,
		height,
		originalSize,
		processedSize: blob.size
	};

	return { ok: true, image, errors, warnings };
}

async function extractDimensions(blob: Blob): Promise<{ width: number; height: number }> {
	return new Promise((resolve) => {
		const url = URL.createObjectURL(blob);
		const img = new Image();
		img.onload = () => {
			resolve({ width: img.naturalWidth, height: img.naturalHeight });
			URL.revokeObjectURL(url);
		};
		img.onerror = () => {
			resolve({ width: 0, height: 0 });
			URL.revokeObjectURL(url);
		};
		img.src = url;
	});
}
