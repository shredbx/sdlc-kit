export interface Link {
  label: string;
  href: string;
  external?: boolean;
}

export interface ImageRef {
  src: string;
  alt: string;
}

export interface MenuItem {
  name: string;
  image: ImageRef;
  priceLabel: string;
}

export interface Step {
  number: string;
  title: string;
  body: string;
}
