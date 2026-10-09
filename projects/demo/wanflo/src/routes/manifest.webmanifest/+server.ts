import { asset, site, translator } from '$lib/content';

export const prerender = true;

// Web app manifest. It is what Android uses when someone adds the site to their home
// screen — without it they get a screenshot of the page as the icon and the raw URL as the
// name. The 192/512 icons already existed (tools/cut-assets.sh generates them) and were
// referenced by nothing at all; this is what they were for.
//
// Generated from content/site.yml like everything else, so a rename or a palette change is
// still one edit in one file.
export function GET() {
	const t = translator('en');
	const body = {
		name: site.name,
		short_name: site.name,
		description: t('home.meta.description'),
		start_url: '/',
		scope: '/',
		display: 'standalone',
		background_color: site.brand.themeColor.light,
		theme_color: site.brand.themeColor.light,
		icons: [
			{ src: asset('/brand/icon-192.png'), sizes: '192x192', type: 'image/png' },
			{ src: asset('/brand/icon-512.png'), sizes: '512x512', type: 'image/png' },
			// `maskable` tells Android it may crop to its own shape. Safe here because the
			// icons are full-bleed navy with the mark inset 22% — nothing important is near
			// the edge. A transparent-cornered icon would come out with black corners.
			{ src: asset('/brand/icon-512.png'), sizes: '512x512', type: 'image/png', purpose: 'maskable' }
		]
	};

	return new Response(JSON.stringify(body, null, '\t'), {
		headers: { 'Content-Type': 'application/manifest+json; charset=utf-8' }
	});
}
