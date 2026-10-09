import type { Handle } from '@sveltejs/kit';

import { mediaOrigin, site } from '$lib/content';

// Fills the app.html placeholders at prerender time (fully static site). Everything it
// injects comes from content/site.yml — the shell has no literals of its own.
const langForPath = (p: string): 'en' | 'ru' | 'th' => {
	if (p === '/ru' || p.startsWith('/ru/')) return 'ru';
	if (p === '/th' || p.startsWith('/th/')) return 'th';
	return 'en';
};

// The script a locale actually needs, on top of Latin. English needs nothing extra.
const FONT_FOR: Record<'en' | 'ru' | 'th', string> = {
	en: '',
	ru: 'nunito-cyrillic.woff2',
	th: 'noto-thai.woff2'
};

const preloadFor = (lang: 'en' | 'ru' | 'th'): string => {
	const file = FONT_FOR[lang];
	if (!file) return '';
	return `<link rel="preload" href="/fonts/${file}" as="font" type="font/woff2" crossorigin="anonymous" />`;
};

// A cross-origin image costs a DNS lookup plus a TLS handshake before its first byte.
// Announced here it starts while the HTML is still parsing. Empty string when media is
// local, which is the default — no wasted connection to our own origin.
const mediaPreconnect = (): string => {
	const origin = mediaOrigin();
	if (!origin) return '';
	return (
		`<link rel="preconnect" href="${origin}" crossorigin="anonymous" />` +
		`<link rel="dns-prefetch" href="${origin}" />`
	);
};

export const handle: Handle = async ({ event, resolve }) => {
	const lang = langForPath(event.url.pathname);
	return resolve(event, {
		transformPageChunk: ({ html }) =>
			html
				.replace('%wanflo.lang%', lang)
				// Browser-chrome colour, light and dark. The inline theme script swaps to the
				// dark one before first paint when that is the visitor's stored choice.
				.replace('%wanflo.themeLight%', site.brand.themeColor.light)
				.replace('%wanflo.themeDark%', site.brand.themeColor.dark)
				// Thai / Cyrillic webfont, preloaded only on the pages that render that script.
				.replace('%wanflo.fontPreload%', preloadFor(lang))
				// Empty unless site.yml names a media CDN.
				.replace('%wanflo.mediaPreconnect%', mediaPreconnect())
	});
};
