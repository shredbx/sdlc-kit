import { getMediaConfig } from './config';
import { MEDIA_SIZES, type MediaSize } from './sizes';

export function mediaUrl(path: string, size: MediaSize): string {
	const { baseUrl } = getMediaConfig();
	const width = MEDIA_SIZES[size];
	const cleanPath = path.startsWith('/') ? path.slice(1) : path;
	const origin = new URL(baseUrl).origin;
	const pathPrefix = new URL(baseUrl).pathname.replace(/\/$/, '');
	return `${origin}/cdn-cgi/image/width=${width},format=auto${pathPrefix}/${cleanPath}`;
}
