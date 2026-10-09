export interface PageSeo {
  title: string;
  description: string;
  path: string;
  openGraphTitle?: string;
  openGraphDescription?: string;
  image?: {
    path: string;
    alt: string;
  };
  robots?: string;
}

export interface SiteSeoDefaults {
  name: string;
  origin: string;
  defaultShareImage: {
    path: string;
    alt: string;
  };
}

export interface ResolvedSeo {
  title: string;
  description: string;
  canonicalUrl: string;
  imageUrl: string;
  imageAlt: string;
  robots: string;
}
