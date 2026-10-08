import { restaurant } from '../../config/restaurant.ts';
import { site } from '../../config/site.ts';
import { resolveSeo } from '../../lib/seo/resolve.ts';
import { homeSeo } from './seo.ts';
import { links } from './content.ts';

const resolvedSeo = resolveSeo(homeSeo, site);

export const restaurantJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'Restaurant',
  name: site.name,
  url: site.origin,
  image: resolvedSeo.imageUrl,
  telephone: restaurant.telephone,
  address: {
    '@type': 'PostalAddress',
    streetAddress: restaurant.streetAddress,
    addressLocality: restaurant.locality,
    addressRegion: restaurant.region,
    postalCode: restaurant.postalCode,
    addressCountry: restaurant.country
  },
  openingHours: restaurant.openingHours,
  servesCuisine: restaurant.servesCuisine,
  priceRange: restaurant.priceRange,
  hasMenu: links.grab.href
};
