import type { PageSeo, ResolvedSeo, SiteSeoDefaults } from './types.ts';

export function resolveSeo(page: PageSeo, site: SiteSeoDefaults): ResolvedSeo {
  const origin = new URL(site.origin);
  if (origin.protocol !== 'https:') {
    throw new Error(`SEO site origin must use HTTPS: ${site.origin}`);
  }

  return {
    title: page.title,
    description: page.description,
    canonicalUrl: new URL(page.path, origin).toString(),
    imageUrl: new URL(page.image?.path ?? site.defaultShareImage.path, origin).toString(),
    imageAlt: page.image?.alt ?? site.defaultShareImage.alt,
    robots: page.robots ?? 'index,follow'
  };
}
