import { LOCALES, localePath, site } from '$lib/content';

export const prerender = true;

const LASTMOD = '2026-08-24';

// The public surface, and only the public surface. /packages and the admin sitemap are
// retired to internal reference (SC7c) — nothing here may advertise them into a 404.
const PAGES = ['', 'website', 'mobile', 'contacts', 'cookies'] as const;

const alternatesFor = (path: string) =>
	LOCALES.map(
		(l) =>
			`\n    <xhtml:link rel="alternate" hreflang="${l}" href="${site.url}${localePath(l, path)}"/>`
	).join('') +
	`\n    <xhtml:link rel="alternate" hreflang="x-default" href="${site.url}${localePath('en', path)}"/>`;

export function GET() {
	const urls = LOCALES.flatMap((l) =>
		PAGES.map((p) => ({ loc: `${site.url}${localePath(l, p)}`, alt: alternatesFor(p) }))
	);

	const body = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">
${urls
	.map(
		(u) => `  <url>
    <loc>${u.loc}</loc>${u.alt}
    <lastmod>${LASTMOD}</lastmod>
  </url>`
	)
	.join('\n')}
</urlset>
`;
	return new Response(body, {
		headers: { 'Content-Type': 'application/xml; charset=utf-8' }
	});
}
