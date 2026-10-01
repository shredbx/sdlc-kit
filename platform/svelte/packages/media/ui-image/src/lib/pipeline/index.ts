// @sbx/ui-image/pipeline — client-side image processing pipeline.
//
// Usage (in consuming app):
//   import { processImage, uploadImage } from '@sbx/ui-image/pipeline';
//   const result = await processImage(file, { maxBytes: 10_000_000, maxDimension: 2048, quality: 0.85 });
//   if (result.ok) await uploadImage(result.image, { owner: 'properties', ownerId, facet: 'cover' });

export { processImage } from './pipeline';
export { uploadImage } from './upload';
export { detectFormat, validateMagicBytes, validateFileSize, readFileHeader, isHEIC } from './validate';
export { extractDropSources, sourceToFile, DropSourceFetchError } from './dropSource';
export type { DropSources } from './dropSource';
export { facetMaxDimension } from './facet';
export type { PipelineConfig, PipelineResult, ProcessedImage, PipelineError, ImageOwner, ImageFacet } from './types';
export type { UploadOptions, UploadResult } from './upload';
export { ALLOWED_TYPES_DEFAULT } from './types';
