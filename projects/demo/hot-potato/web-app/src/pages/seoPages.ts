import { homeSeo } from './home/seo.ts';
import { restaurantJsonLd } from './home/structuredData.ts';

export const seoPages = [
  {
    seo: homeSeo,
    structuredData: restaurantJsonLd
  }
];
