export const MEDIA_SIZES = {
	thumb: 120,
	'card-sm': 320,
	'card-lg': 640,
	hero: 1280,
	full: 1920
} as const;

export type MediaSize = keyof typeof MEDIA_SIZES;
