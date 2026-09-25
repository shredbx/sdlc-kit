// Pure decision logic extracted from LocationPicker.svelte so the touched-flip
// + emit-or-swallow rules are unit-testable without rendering a Svelte component.
// The .svelte component must delegate to these helpers — they ARE the contract.

import type { AdminFrameQuery, LngLat } from './types.js';
import { clampCoords, coordsEqual, isNullIsland, isValidCoords } from './coords.js';

export interface PinState {
	center: LngLat;
	dirty: boolean;
}

/**
 * Compute the picker's initial pin state from the lat/lng props the consumer
 * passed in. Returns `dirty: false` (placeholder visible, no onchange should
 * fire on first idle) when the props are missing, invalid, or null-island.
 *
 * The lat/lng parameters are typed `unknown` because Svelte 5 `$props()` is
 * runtime-lax — a consumer could pass a string from a query param, null from
 * a form, etc. The internal `typeof === 'number'` narrowing keeps the
 * remainder of the body safe.
 */
export function initialPinState(
	lat: unknown,
	lng: unknown,
	defaultCenter: LngLat
): PinState {
	if (typeof lat !== 'number' || typeof lng !== 'number') {
		return { center: defaultCenter, dirty: false };
	}
	const raw = { lat, lng };
	if (!isValidCoords(raw) || isNullIsland(raw)) {
		return { center: defaultCenter, dirty: false };
	}
	return { center: clampCoords(raw), dirty: true };
}

/**
 * Returns true iff the picker's center has actually changed. The previous
 * `&& touched` guard in LocationPicker.svelte:91 mis-handled the mount-time
 * case — when the adapter's first idle event fires with the same coords the
 * map was asked to render at, the picker would slip past the guard and flip
 * dirty=true. This helper drops `touched` entirely: identical coords always
 * mean "nothing changed", regardless of state. The dirty flip is the
 * caller's responsibility (set it after emit, not before the compare).
 */
export function coordsChanged(prev: LngLat, next: LngLat): boolean {
	return !coordsEqual(prev, next);
}

/**
 * Assemble the geocodable query string for the DEEPEST non-empty admin level
 * in the cascade, building UP from sub-district to country. Returns `null`
 * when nothing is filled (no frame should be attempted).
 *
 * The deepest filled level anchors the string and every coarser level that is
 * also filled is appended for disambiguation, so a Thai sub-district like
 *   { subDistrict: 'Baan Tai', district: 'Koh Phangan',
 *     province: 'Surat Thani', country: 'Thailand' }
 * geocodes as "Baan Tai, Koh Phangan, Surat Thani, Thailand" — but a form with
 * only the province set yields "Surat Thani, Thailand". Whitespace-only values
 * are treated as empty. This is the LocationPicker's empty-state framing
 * contract; the component must delegate here.
 */
export function deepestAdminQuery(frame: AdminFrameQuery | undefined): string | null {
	if (!frame) return null;
	// Coarsest → finest, so the assembled parts read finest → coarsest when we
	// walk this list and prepend each filled level.
	const ordered = [frame.country, frame.province, frame.district, frame.subDistrict];
	const parts: string[] = [];
	for (const level of ordered) {
		const v = typeof level === 'string' ? level.trim() : '';
		if (v) parts.unshift(v);
	}
	return parts.length > 0 ? parts.join(', ') : null;
}

// ── Zoom-ladder framing ──────────────────────────────────────────────────

/**
 * Admin cascade levels, finest → coarsest. `location` is the placed-pin level
 * (the deepest, max-zoom frame). `country` is the coarsest fallback. The order
 * here is the AUTHORITY for "deepest filled level" — `deepestFilledLevel` walks
 * it finest-first and returns the first level present in the frame.
 */
export type AdminZoomLevel = 'location' | 'subDistrict' | 'district' | 'province' | 'country';

/**
 * A full ladder of Google zoom integers per admin level. `targetZoom` reads
 * this; the component supplies `DEFAULT_ADMIN_ZOOM` (or a consumer override).
 */
export type ZoomLadder = Record<AdminZoomLevel, number>;

/**
 * User-confirmed framing ladder (Google zoom integers): the camera tightens as
 * the agent fills deeper admin levels, and a placed pin frames at the max
 * (`location` = 16). `country` (6) is the coarsest fallback when only the
 * country is filled. These are the defaults the LocationPicker passes to
 * `targetZoom`; a consumer may override.
 */
export const DEFAULT_ADMIN_ZOOM: ZoomLadder = {
	location: 16,
	subDistrict: 13,
	district: 11,
	province: 9,
	country: 6
};

/**
 * The deepest non-empty admin level in a frame, finest → coarsest. Returns
 * `null` when nothing is filled. Whitespace-only values are treated as empty
 * (mirrors `deepestAdminQuery`). `location` is never returned here — that level
 * is owned by the pin, surfaced via `targetZoom(..., pinned=true)`.
 */
export function deepestFilledLevel(
	frame: AdminFrameQuery | undefined
): Exclude<AdminZoomLevel, 'location'> | null {
	if (!frame) return null;
	const ordered: Array<[Exclude<AdminZoomLevel, 'location'>, unknown]> = [
		['subDistrict', frame.subDistrict],
		['district', frame.district],
		['province', frame.province],
		['country', frame.country]
	];
	for (const [level, value] of ordered) {
		const v = typeof value === 'string' ? value.trim() : '';
		if (v) return level;
	}
	return null;
}

/**
 * The Google zoom integer the camera should frame at:
 *  - a placed pin (`pinned`) → `ladder.location` (max zoom, 16) — the pin owns
 *    the frame regardless of which admin levels are filled.
 *  - otherwise the deepest filled admin level → its ladder zoom.
 *  - nothing filled and no pin → `null` (no opinion — leave the camera where
 *    it is / on the persisted view).
 *
 * Pure: the component delegates BOTH the empty-state framing and the on-commit
 * framing to this single function so the ladder lives in one tested place.
 */
export function targetZoom(
	frame: AdminFrameQuery | undefined,
	pinned: boolean,
	ladder: ZoomLadder = DEFAULT_ADMIN_ZOOM
): number | null {
	if (pinned) return ladder.location;
	const level = deepestFilledLevel(frame);
	if (!level) return null;
	return ladder[level];
}
