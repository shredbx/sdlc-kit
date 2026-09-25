// Canonical map markers — ONE source of truth for the two location glyphs used across every
// map surface (viewers + pickers), so the PROPERTY pin never drifts into several different
// looks (task 2607-115 consistency pass).
//
//   PROPERTY LOCATION → always the red teardrop with a white centre circle, anchored at its
//                       bottom TIP so the point marks the exact spot (never the glyph centre).
//   AREA LOCATION     → a soft dashed halo (NEVER a pin), captioned with the area name it
//                       represents, centred on the area centroid.
//
// Every consumer (maplibre-viewer, google-viewer, LocationPicker, PolygonPicker, and any
// static-map builder) derives its marker from here — no inline pin SVGs, colours, or dots.

/** Property-location red — the one map-pin red used everywhere. */
export const PIN_RED = '#EA4335';
/** Approximate-area tone (neutral slate) — distinct from the exact red and the plot blue. */
export const AREA_TONE_RGB = '71,85,105';

// Canonical property-location pin as an SVG string: red teardrop + white centre circle + white
// casing. viewBox is 0 0 24 32; the TIP is at (12, 32), so DOM consumers anchor bottom-centre
// and Google consumers set the icon anchor to (width/2, height).
export function locationPinSvg(width = 28, height = 38): string {
	return (
		`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 24 32" aria-hidden="true">` +
		`<path d="M12 0C5.7 0 .6 5.1.6 11.4.6 20 12 32 12 32s11.4-12 11.4-20.6C23.4 5.1 18.3 0 12 0z" ` +
		`fill="${PIN_RED}" stroke="#fff" stroke-width="1.5"/>` +
		`<circle cx="12" cy="11.4" r="4.2" fill="#fff"/></svg>`
	);
}

// A data: URI of the canonical pin — for Google Maps, whose Marker icon takes a URL. Pair with
// an anchor at (width/2, height) so the tip sits on the coordinate.
export function locationPinDataUri(width = 28, height = 38): string {
	return 'data:image/svg+xml;charset=UTF-8,' + encodeURIComponent(locationPinSvg(width, height));
}

// Canonical property-location pin as a ready-to-mount DOM element (for MapLibre / HTML overlays).
// Anchor it at 'bottom' so the tip marks the spot.
export function locationPinElement(): HTMLDivElement {
	const el = document.createElement('div');
	el.setAttribute('data-marker', 'exact');
	el.style.cssText = 'display:block;line-height:0;filter:drop-shadow(0 2px 3px rgba(0,0,0,0.35));';
	el.innerHTML = locationPinSvg();
	return el;
}

// Canonical AREA marker as a DOM element: a soft dashed halo centred on the area centroid,
// captioned with the area NAME below it. Anchor CENTRE — the halo centres on the point; the
// caption is positioned out of flow so it never shifts the anchor.
export function areaMarkerElement(label?: string): HTMLDivElement {
	const el = document.createElement('div');
	el.setAttribute('data-marker', 'approximate');
	el.style.cssText = 'position:relative;width:64px;height:64px;';
	// A translucent slate disc with a SOLID slate ring + a white casing + drop shadow, and a
	// solid centre dot as the focal point. Deliberately stronger than the old faint dashed
	// halo, which blended into a busy terrain base (2607-118) — this reads on any map.
	const halo =
		`<div style="position:absolute;inset:0;border-radius:50%;box-sizing:border-box;` +
		`background:rgba(${AREA_TONE_RGB},0.22);border:3px solid rgba(${AREA_TONE_RGB},0.95);` +
		`box-shadow:0 0 0 2px rgba(255,255,255,0.9),0 2px 6px rgba(0,0,0,0.35);"></div>`;
	const dot =
		`<div style="position:absolute;top:50%;left:50%;width:12px;height:12px;` +
		`transform:translate(-50%,-50%);border-radius:50%;background:rgb(${AREA_TONE_RGB});` +
		`border:2px solid #fff;box-shadow:0 1px 2px rgba(0,0,0,0.4);"></div>`;
	const caption = label
		? `<span style="position:absolute;top:100%;left:50%;transform:translateX(-50%);margin-top:6px;` +
			`white-space:nowrap;font:600 11px/1.2 system-ui,-apple-system,sans-serif;color:#fff;` +
			`background:rgba(${AREA_TONE_RGB},0.95);padding:3px 8px;border-radius:9px;` +
			`box-shadow:0 1px 3px rgba(0,0,0,0.35);">${escapeHtml(label)}</span>`
		: '';
	el.innerHTML = halo + dot + caption;
	return el;
}

// ── Catalog board markers (task 2607-133) ────────────────────────────────────────────
// The listings map plots MANY properties at once, so it needs a marker that carries a VALUE
// (the price) rather than just a position. Same source of truth as the two glyphs above — a
// board marker is a third canonical glyph here, never an inline pill at a call site.
//
//   PRICE PILL  → a white capsule + a downward tail whose tip marks the point. Used for an
//                 exact property location AND for an area group (which adds a count chip).
//   ACTIVE      → the pill fills BR gold. Identification never rests on the fill alone: the
//                 casing darkens and the weight steps up too, so the state survives a
//                 colour-blind read and a low-contrast basemap.
//
// Colours are literals here for the same reason the pin red is: this package is consumed by
// several apps and cannot read a host app's CSS custom properties from a detached DOM node.

/** Board pill palette — resting. Teal casing (not a hairline grey): a white pill on a pale
 *  basemap is ~1.13:1, far under the 3:1 needed to identify a control, so the CASING is what
 *  makes the marker findable. #1A7A7A against pale land is ~4.5:1. */
const PILL_BG = '#FFFFFF';
const PILL_CASING = '#1A7A7A';
const PILL_FG = '#0D4F4F';
/** Board pill palette — active/selected. BR gold, with a near-black teal casing (~11:1). */
const PILL_ACTIVE_BG = '#C8A851';
const PILL_ACTIVE_CASING = '#083838';
const PILL_ACTIVE_FG = '#083838';

export interface PricePillOptions {
	/** The pill's text — a formatted price ("฿14.2M", "฿180K/mo") or any short label. */
	label: string;
	/** Group size. >1 renders a count chip after the label; 1/undefined renders the label alone. */
	count?: number;
	/** Selected state — gold fill, darker casing, heavier text, lifted shadow. */
	active?: boolean;
}

// Canonical BOARD marker: a price capsule with a tail whose TIP marks the coordinate. Anchor
// 'bottom' so the tip — not the capsule centre — sits on the point, exactly like the teardrop
// pin. The tail is two stacked triangles (casing behind fill) so it flips colour with the pill
// instead of leaving a white wedge under a gold capsule.
export function pricePillElement(opts: PricePillOptions): HTMLDivElement {
	const active = opts.active === true;
	const bg = active ? PILL_ACTIVE_BG : PILL_BG;
	const casing = active ? PILL_ACTIVE_CASING : PILL_CASING;
	const fg = active ? PILL_ACTIVE_FG : PILL_FG;
	const weight = active ? 700 : 600;
	// Two-part shadow (key + ambient), tinted with the brand teal rather than neutral black so
	// the marker sits on the map instead of floating in a grey haze.
	const shadow = active
		? '0 2px 4px rgba(13,79,79,0.12),0 8px 20px rgba(13,79,79,0.16)'
		: '0 1px 2px rgba(13,79,79,0.10),0 2px 6px rgba(13,79,79,0.10)';

	// The ROOT must NOT set `position` — MapLibre positions a marker via the
	// `.maplibregl-marker { position:absolute }` CLASS it adds to this element, and an inline
	// `position` here would override that class (inline beats a class selector) and drop the
	// marker into normal document flow (all markers then pile at the container's top-left). The
	// relative context the tails need lives on the INNER wrapper instead.
	const el = document.createElement('div');
	el.setAttribute('data-marker', 'price');
	if (active) el.setAttribute('data-active', 'true');
	el.style.cssText = 'display:block;line-height:0;';

	// Count chip — an area group's inventory ("7 homes here"), NOT a proximity cluster bubble.
	// aria-hidden because the marker's own aria-label already spells the count out in words.
	const chip =
		opts.count && opts.count > 1
			? `<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;` +
				`min-width:18px;height:18px;padding:0 5px;margin-left:6px;margin-right:-3px;border-radius:9px;` +
				`background:${active ? PILL_ACTIVE_FG : PILL_FG};color:#fff;font:700 11px/1 Inter,system-ui,sans-serif;">` +
				`${escapeHtml(String(opts.count))}</span>`
			: '';

	// Inter (not the brand serif): a display face's hairlines and high-contrast numerals break up
	// at 13px over varied terrain. tabular-nums keeps ฿ figures aligned across stacked pills.
	const pill =
		`<span style="display:inline-flex;align-items:center;box-sizing:border-box;min-height:28px;` +
		`padding:4px 10px;border-radius:999px;background:${bg};border:1px solid ${casing};` +
		`box-shadow:${shadow};color:${fg};white-space:nowrap;` +
		`font:${weight} 13px/1.15 Inter,system-ui,-apple-system,sans-serif;letter-spacing:0.01em;` +
		`font-variant-numeric:tabular-nums;">${escapeHtml(opts.label)}${chip}</span>`;

	// Tail: casing triangle 1px larger, behind the fill triangle. Both hang below the capsule.
	const tailCasing =
		`<span style="position:absolute;top:100%;left:50%;width:0;height:0;margin-left:-7px;` +
		`border-left:7px solid transparent;border-right:7px solid transparent;` +
		`border-top:8px solid ${casing};"></span>`;
	const tailFill =
		`<span style="position:absolute;top:100%;left:50%;width:0;height:0;margin-left:-6px;` +
		`border-left:6px solid transparent;border-right:6px solid transparent;` +
		`border-top:7px solid ${bg};"></span>`;

	// Inner wrapper owns the relative context for the two tail triangles; the root stays
	// position-free so MapLibre's absolute placement works (see the root comment above).
	el.innerHTML =
		`<div style="position:relative;display:inline-block;line-height:0;">${pill}${tailCasing}${tailFill}</div>`;
	return el;
}

/** Canonical CLUSTER glyph — the SAME white capsule as the price pill (one marker vocabulary
 *  on the board — the earlier dark count-circle read as a second, unexplained species of
 *  marker), labelled "N listings". No tail and anchored CENTER: a cluster stands at its
 *  members' mean position, not at a real place, so a precision tip would over-promise. */
export function clusterPillElement(count: number, active = false): HTMLDivElement {
	const bg = active ? PILL_ACTIVE_BG : PILL_BG;
	const casing = active ? PILL_ACTIVE_CASING : PILL_CASING;
	const fg = active ? PILL_ACTIVE_FG : PILL_FG;
	const el = document.createElement('div');
	el.setAttribute('data-marker', 'cluster');
	if (active) el.setAttribute('data-active', 'true');
	el.style.cssText = 'display:block;line-height:0;';
	el.innerHTML =
		`<div style="display:inline-flex;align-items:center;box-sizing:border-box;min-height:28px;` +
		`padding:4px 12px;border-radius:999px;background:${bg};border:1px solid ${casing};` +
		`box-shadow:0 1px 2px rgba(13,79,79,0.10),0 2px 6px rgba(13,79,79,0.10);color:${fg};` +
		`white-space:nowrap;font:${active ? 700 : 600} 13px/1.15 Inter,system-ui,-apple-system,sans-serif;` +
		`letter-spacing:0.01em;font-variant-numeric:tabular-nums;">` +
		`${escapeHtml(String(count))}&nbsp;listings</div>`;
	return el;
}

/** Estimated on-screen box of a board marker BEFORE it exists in the DOM — feeds the
 *  rectangle-collision declutter (a pill is ~4× wider than tall, so collision needs the real
 *  footprint, not a radius). 13px Inter averages ≈7.3px/character; the constants mirror the
 *  paddings/borders/chip metrics in {@link pricePillElement}. An estimate only — ±10% is fine,
 *  the declutter adds its own gap. */
export function estimatePillSize(label: string, count?: number): { w: number; h: number } {
	const chip = count && count > 1 ? 6 + Math.max(18, String(count).length * 8 + 10) - 3 : 0;
	return {
		w: Math.round(2 + 20 + label.length * 7.3 + chip),
		h: 28 + 8 // capsule + tail
	};
}

function escapeHtml(s: string): string {
	return s.replace(
		/[&<>"]/g,
		(c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c] ?? c
	);
}
