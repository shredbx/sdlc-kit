import { getMediaConfig } from './config';
import { MEDIA_SIZES, type MediaSize } from './sizes';

const SRCSET_WIDTHS = [320, 640, 1280, 1920];

export type ImgFit = 'thumb' | 'card' | 'showcase' | 'hero' | 'full';

const FIT_SIZES: Record<ImgFit, string> = {
	thumb: '120px',
	card: '(max-width: 768px) 100vw, 320px',
	showcase: '(max-width: 768px) 100vw, 640px',
	hero: '(max-width: 768px) 100vw, 1280px',
	full: '100vw'
};

export function mediaSrcset(url: string): string {
	if (!url) return '';
	if (url.endsWith('.svg')) return '';
	let baseUrl: string;
	try {
		baseUrl = getMediaConfig().baseUrl;
	} catch {
		return '';
	}
	if (!url.startsWith(baseUrl)) return '';
	const origin = new URL(baseUrl).origin;
	const pathPrefix = new URL(baseUrl).pathname.replace(/\/$/, '');
	const path = url.slice(baseUrl.length);
	return SRCSET_WIDTHS
		.map((w) => `${origin}/cdn-cgi/image/width=${w},format=auto${pathPrefix}${path} ${w}w`)
		.join(', ');
}

export function imgSizes(fit: ImgFit): string {
	return FIT_SIZES[fit];
}
