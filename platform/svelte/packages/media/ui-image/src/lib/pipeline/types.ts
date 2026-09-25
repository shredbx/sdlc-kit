// Image pipeline types — shared across all consumers.
// Pipeline is purpose-agnostic: the consuming app defines named configs.

export interface PipelineConfig {
	maxBytes: number;
	maxDimension: number;
	quality: number;
	allowedTypes?: string[];
	// Optional output MIME forcing. Default behaviour (when unset) is:
	//   PNG → PNG, everything else → WebP. Consumers that always want WebP
	//   (e.g. property photos with no transparency) set `outputFormat:
	//   'image/webp'` to stop PNG sources landing in storage.
	outputFormat?: string;
}

export interface ProcessedImage {
	blob: Blob;
	format: string;
	width: number;
	height: number;
	originalSize: number;
	processedSize: number;
}

export interface PipelineError {
	step: string;
	message: string;
	fatal: boolean;
}

export interface PipelineResult {
	ok: boolean;
	image?: ProcessedImage;
	errors: PipelineError[];
	warnings: PipelineError[];
}

export const ALLOWED_TYPES_DEFAULT = [
	'image/jpeg',
	'image/png',
	'image/webp',
	'image/heic',
	'image/heif',
	'image/avif'
];

// ImageOwner is the owning entity's collection name (plural, matching the
// table/route) and the first segment of the owner-scoped object key
// (Decision #0271, object-storage-key-governance OSK-001). Mirrors the Go
// pkg/image ImageOwner enum. 'system' is the reserved namespace for ownerless
// brand assets (OSK-002) — those carry no ownerId.
export type ImageOwner = 'properties' | 'agents' | 'contacts' | 'pages' | 'system';

// ImageFacet is the role of an asset within its owner — the {facet} segment of
// the owner-scoped key (Decision #0271). Mirrors the Go pkg/image ImageFacet enum.
// There is NO 'general' catch-all: an unowned/role-less asset is a modeling gap.
// 'logo' is the displayed site brand mark under the reserved system/logo/ prefix
// (OSK-002) — DISTINCT from 'watermark' (an overlay the R2 worker consumes, never
// shown). Mirrored in the Go enum + the facet dictionary (Decision #0271 parity).
export type ImageFacet = 'cover' | 'gallery' | 'avatar' | 'photo' | 'qr' | 'hero' | 'watermark' | 'logo';
