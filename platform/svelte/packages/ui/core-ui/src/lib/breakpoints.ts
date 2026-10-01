/**
 * Responsive breakpoints — single source of truth for the design system.
 *
 * Each value is the **min viewport width (px) at which the wider layout begins**.
 * This matches the existing component convention where mobile is `≤768` and
 * desktop is `≥769`, so the helpers below emit the exact same media strings the
 * components already hand-wrote (no 1px drift on migration).
 *
 *   BREAKPOINTS.desktop = 769  → mediaUp('desktop')   === '(min-width: 769px)'
 *                              → mediaDown('desktop')  === '(max-width: 768px)'
 *
 * Why a module and not CSS custom properties: this package has no
 * `postcss-custom-media`, so `@media (max-width: var(--bp))` is NOT supported.
 * Breakpoints therefore live here as named constants:
 *   - JS reads the constant directly (`window.innerWidth`, `matchMedia`).
 *   - CSS keys off a static state class the component derives from the name
 *     (e.g. `sidebarOverlayBreakpoint="wide"` → `.overlay-wide`), so the only
 *     literal in a `@media` rule is documented with a `/* = BREAKPOINTS.x *​/`
 *     comment pointing back here.
 *
 * Consumers pass the **name**, never the number (e.g. `breakpoint="wide"`).
 * Adding a new breakpoint = one entry here + (if it drives layout) one additive
 * CSS band in the consuming component.
 */
export const BREAKPOINTS = {
	/** First desktop pixel. Mobile/drawer layout is everything below this (≤768). */
	desktop: 769,
	/** Full-desktop pixel. Below this a docked sidebar overlays content instead of
	 *  pushing it (the BR admin overlay regime — Wave 2 / B2). */
	wide: 1280,
} as const;

export type BreakpointName = keyof typeof BREAKPOINTS;

/** `(min-width: <bp>px)` — the named layout and wider. */
export const mediaUp = (bp: BreakpointName): string => `(min-width: ${BREAKPOINTS[bp]}px)`;

/** `(max-width: <bp-1>px)` — strictly narrower than the named layout. */
export const mediaDown = (bp: BreakpointName): string => `(max-width: ${BREAKPOINTS[bp] - 1}px)`;

/** `(min-width: <min>px) and (max-width: <maxExclusive-1>px)` — a single band. */
export const mediaBetween = (min: BreakpointName, maxExclusive: BreakpointName): string =>
	`${mediaUp(min)} and ${mediaDown(maxExclusive)}`;

/** True when the viewport is at least the named breakpoint. SSR-safe (0 width). */
export function matchesUp(
	bp: BreakpointName,
	width: number = typeof window !== 'undefined' ? window.innerWidth : 0
): boolean {
	return width >= BREAKPOINTS[bp];
}
