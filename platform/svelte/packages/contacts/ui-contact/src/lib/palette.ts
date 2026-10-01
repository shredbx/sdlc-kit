// Deterministic per-category color (Decision 2026-06-07: no backend / no dictionary color
// field). A stable hash of the category code selects a hue from a fixed, brand-harmonious
// palette, so the same category always dots the same color across rows and badges — without
// storing any color. Upgradeable later to a dictionary `color` field without changing callers.

const PALETTE = [
	'#0d4f4f', // deep ocean teal (brand primary)
	'#c8a851', // warm gold (brand accent)
	'#3b6ea5', // muted blue
	'#8a5a44', // terracotta
	'#5b8c5a', // sage green
	'#9c5b8b', // plum
	'#c0703a', // amber
	'#5a6b7b' // slate
];

/** Stable color for a category code (FNV-ish rolling hash → palette index). */
export function categoryColor(code: string): string {
	let h = 0;
	for (let i = 0; i < code.length; i++) {
		h = (h * 31 + code.charCodeAt(i)) >>> 0;
	}
	return PALETTE[h % PALETTE.length];
}
