// MapLibre BOARD adapter (task 2607-133) — the multi-marker PUBLIC map behind the property
// catalog's Map view and the homepage map section. Sibling of maplibre-viewer.ts: same free
// OpenFreeMap basemap, same lazy `import('maplibre-gl')` so a page ships none of the WebGL
// engine until a board actually mounts, same fatal-vs-tile error split. What differs is the
// job — this one plots MANY points, DECLUTTERS them so none ever overlap, and owns a selection.
//
// Two glyphs render (both from the canonical markers.ts):
//   • a PRICE PILL for a single marker (an area group or an exact pin)
//   • a COUNT BUBBLE when several markers collide at the current zoom — the Google-style
//     "N here" that expands (zooms to its members) on click and dissolves back into pills as
//     you zoom in and they move apart in pixel space.
//
// The decluster MATH is pure + unit-tested in cluster.ts; this file is the DOM/GL glue (no
// WebGL under vitest), so it isn't unit-tested. It deliberately draws no plot/legend/vertex
// handles — that is the single-property viewer's job (Decision #0303).
import { applyBrandWaterWash, resolveViewerStyle } from '../maplibre.js';
import { pricePillElement, clusterPillElement, estimatePillSize } from '../markers.js';
import { declusterByPixel, type PixelPoint } from '../cluster.js';
import type { MapBoardAdapter, MapBoardFrame, MapBoardMountOpts, MapMarkerPoint } from '../types.js';

type MaplibreModule = typeof import('maplibre-gl');
type MaplibreMap = import('maplibre-gl').Map;
type MaplibreMarker = import('maplibre-gl').Marker;

/** Camera used when a board has no points at all (Thailand, country view). */
const FALLBACK_CENTER = { lat: 13.0, lng: 101.0 };
const FALLBACK_ZOOM = 5;

/** Never zoom past this when framing markers — fitting a single point (or several that share an
 *  area centroid, geometrically identical) yields a zero-area bounds, and MapLibre answers that
 *  by zooming to maximum: a street-level view of an empty field. BR's shared-centroid data makes
 *  that the common case, not an edge case. */
const FIT_MAX_ZOOM = 13;

/** Deepest zoom a cluster expansion may reach — and therefore the zoom at which "can these
 *  members ever separate?" is judged. Past this an area-based map shows doorstep-level streets
 *  for listings that only have an area, which over-promises precision. */
const EXPAND_MAX_ZOOM = 16;

/** Minimum center-to-center gap (px) two markers may have before they merge into a bubble.
 *  ~54px clears a ~28px pill with breathing room. */
const COLLIDE_RADIUS = 54;

/** Stacking order — selected above hovered above resting. */
const Z_RESTING = 1;
const Z_HOVER = 10;
const Z_ACTIVE = 20;

/** A rendered item after declutter: a lone marker, or a merged cluster of several. */
type DisplayItem =
	| { kind: 'point'; key: string; lat: number; lng: number; point: MapMarkerPoint }
	| { kind: 'cluster'; key: string; lat: number; lng: number; count: number; memberKeys: string[] };

interface Entry {
	marker: MaplibreMarker;
	item: DisplayItem;
	el: HTMLElement;
}

export function createMaplibreBoardAdapter(): MapBoardAdapter {
	let map: MaplibreMap | null = null;
	let ml: MaplibreModule | null = null;
	let entries = new Map<string, Entry>();
	let sourcePoints: readonly MapMarkerPoint[] = [];
	let sourceByKey = new Map<string, MapMarkerPoint>();
	let activeKey: string | null = null;
	let frame: MapBoardFrame | null = null;
	let interactive = true;
	let fitPadding: MapBoardMountOpts['fitPadding'];
	let activePanPadding: MapBoardMountOpts['activePanPadding'];
	let onSelect: MapBoardMountOpts['onSelect'];
	let onClusterSelect: MapBoardMountOpts['onClusterSelect'];
	let generation = 0;

	// ── Declutter ────────────────────────────────────────────────────────────────────────
	// Project every source marker to the current screen, merge overlaps (cluster.ts), and turn
	// each resulting cell back into a display item anchored at its members' mean lng/lat.
	function computeDisplay(): DisplayItem[] {
		const m = map;
		if (!m) return [];
		const projected: PixelPoint[] = sourcePoints.map((p) => {
			const pt = m.project([p.lng, p.lat]);
			const size = estimatePillSize(p.label, p.count);
			return { key: p.key, x: pt.x, y: pt.y, count: p.count ?? 1, w: size.w, h: size.h };
		});
		const cells = declusterByPixel(projected, COLLIDE_RADIUS);
		const items: DisplayItem[] = [];
		for (const cell of cells) {
			if (cell.memberKeys.length === 1) {
				const src = sourceByKey.get(cell.memberKeys[0]);
				if (src) items.push({ kind: 'point', key: src.key, lat: src.lat, lng: src.lng, point: src });
				continue;
			}
			let la = 0;
			let lo = 0;
			let n = 0;
			for (const k of cell.memberKeys) {
				const s = sourceByKey.get(k);
				if (!s) continue;
				la += s.lat;
				lo += s.lng;
				n++;
			}
			if (n === 0) continue;
			items.push({
				kind: 'cluster',
				key: cell.key,
				lat: la / n,
				lng: lo / n,
				count: cell.count,
				memberKeys: cell.memberKeys
			});
		}
		return items;
	}

	// A cluster click has exactly two honest outcomes. If zooming can separate the members, zoom
	// (fitBounds) — the Google behaviour. If it CANNOT — the members share (or nearly share) one
	// coordinate, the normal case for area-based listings — hand the members to the host via
	// onClusterSelect so it opens their listings. A click must never change nothing.
	//
	// "Can it split?" is decided in PIXELS, not by degenerate bounds: pixel gaps scale by
	// 2^Δzoom, so if even the WIDEST member pair would still sit inside the collide radius at
	// the deepest zoom we fit to, zooming is futile no matter how non-zero the bounds are.
	function expandCluster(clusterKey: string, memberKeys: string[]): void {
		const m = map;
		if (!m || !ml) return;
		const pts = memberKeys.map((k) => sourceByKey.get(k)).filter(Boolean) as MapMarkerPoint[];
		if (pts.length === 0) return;

		// An AMBIENT board never moves its own camera — a zoom the visitor has no controls to
		// undo. The host decides what a stack means there (usually: open the full map).
		if (!interactive) {
			onClusterSelect?.(clusterKey, memberKeys);
			return;
		}

		let maxPx = 0;
		const projected = pts.map((p) => m.project([p.lng, p.lat]));
		for (let i = 0; i < projected.length; i++) {
			for (let j = i + 1; j < projected.length; j++) {
				const dx = projected[i].x - projected[j].x;
				const dy = projected[i].y - projected[j].y;
				maxPx = Math.max(maxPx, Math.hypot(dx, dy));
			}
		}
		const splittable = maxPx * 2 ** (EXPAND_MAX_ZOOM - m.getZoom()) >= COLLIDE_RADIUS;

		if (splittable) {
			const bounds = new ml.LngLatBounds();
			for (const p of pts) bounds.extend([p.lng, p.lat]);
			m.fitBounds(bounds, { padding: fitPadding ?? padDefault(), maxZoom: EXPAND_MAX_ZOOM, duration: 300 });
			return;
		}
		// Unsplittable: center on the spot and let the host reveal the listings. Without a host
		// handler, fall back to the old zoom nudge rather than swallowing the click entirely.
		if (onClusterSelect) {
			m.easeTo({ center: [pts[0].lng, pts[0].lat], duration: 300 });
			onClusterSelect(clusterKey, memberKeys);
		} else {
			m.easeTo({ center: [pts[0].lng, pts[0].lat], zoom: Math.min(m.getZoom() + 2, 18), duration: 300 });
		}
	}

	// ── Marker DOM ───────────────────────────────────────────────────────────────────────
	// Visible keyboard focus — markers are detached DOM with inline styles, so no app stylesheet
	// can reach them; the ring is applied by hand, and only for :focus-visible so a mouse click
	// doesn't leave one behind. Near-black teal reads on the white pill, the gold active pill,
	// and every basemap tone.
	function addFocusRing(el: HTMLElement): void {
		el.addEventListener('focus', () => {
			let keyboard = true;
			try {
				keyboard = el.matches(':focus-visible');
			} catch {
				// Selector unsupported → show the ring for every focus; visible beats invisible.
			}
			if (!keyboard) return;
			el.style.outline = '2px solid #083838';
			el.style.outlineOffset = '2px';
			el.style.borderRadius = '999px';
		});
		el.addEventListener('blur', () => {
			el.style.outline = '';
			el.style.outlineOffset = '';
		});
	}

	// Hover/focus lift — a TRANSLATE, never a scale: the tail tip marks the coordinate and
	// scaling would slide it off the spot the marker points at. The lift is applied to the
	// INNER wrapper, NEVER the root: MapLibre positions the marker by writing
	// `style.transform = translate(x,y)` onto the ROOT, so touching the root's transform
	// erases the marker's placement and dumps it at the container origin (the exact bug that
	// piled every pill at the top-left corner). Root = MapLibre's; inner = ours. Shared by
	// BOTH capsule kinds (price pill + cluster pill) — same vocabulary, same affordance.
	function addLift(el: HTMLElement, isActive: () => boolean): void {
		const inner = () => el.firstElementChild as HTMLElement | null;
		const raise = () => {
			if (isActive()) return;
			el.style.zIndex = String(Z_HOVER);
			const w = inner();
			if (w) w.style.transform = 'translateY(-2px)';
		};
		const lower = () => {
			if (isActive()) return;
			el.style.zIndex = String(Z_RESTING);
			const w = inner();
			if (w) w.style.transform = '';
		};
		el.addEventListener('pointerenter', raise);
		el.addEventListener('pointerleave', lower);
		el.addEventListener('focus', raise);
		el.addEventListener('blur', lower);
	}

	function buildElement(item: DisplayItem): HTMLElement {
		if (item.kind === 'cluster') {
			const el = clusterPillElement(item.count, item.key === activeKey);
			el.setAttribute('role', 'button');
			el.setAttribute('tabindex', '0');
			el.setAttribute('aria-label', `${item.count} listings here. Opens them.`);
			el.style.cursor = 'pointer';
			el.style.zIndex = String(Z_RESTING);
			addFocusRing(el);
			addLift(el, () => item.key === activeKey);
			const go = (ev: Event) => {
				ev.stopPropagation();
				expandCluster(item.key, item.memberKeys);
			};
			el.addEventListener('click', go);
			el.addEventListener('keydown', (ev) => {
				const e = ev as KeyboardEvent;
				if (e.key === 'Enter' || e.key === ' ') {
					e.preventDefault();
					go(e);
				}
			});
			return el;
		}

		const p = item.point;
		const el = pricePillElement({ label: p.label, count: p.count, active: p.key === activeKey });
		el.setAttribute('role', 'button');
		el.setAttribute('tabindex', '0');
		el.setAttribute('aria-label', p.ariaLabel);
		el.style.cursor = 'pointer';
		el.style.zIndex = String(p.key === activeKey ? Z_ACTIVE : Z_RESTING);
		addFocusRing(el);
		const activate = (ev: Event) => {
			ev.stopPropagation();
			onSelect?.(p.key);
		};
		el.addEventListener('click', activate);
		el.addEventListener('keydown', (ev) => {
			const e = ev as KeyboardEvent;
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault();
				activate(e);
			}
		});
		addLift(el, () => p.key === activeKey);
		return el;
	}

	function padDefault() {
		return { top: 48, right: 48, bottom: 48, left: 48 };
	}

	// Repaint an existing marker's glyph in place (active/count/label changed) WITHOUT recreating
	// the marker — the root element keeps its click/hover listeners; only the inner visual and the
	// active affordances (data-active, z-index) are swapped.
	function repaint(entry: Entry): void {
		const item = entry.item;
		const active = item.key === activeKey;
		const fresh =
			item.kind === 'cluster'
				? clusterPillElement(item.count, active)
				: pricePillElement({ label: item.point.label, count: item.point.count, active });
		entry.el.innerHTML = fresh.innerHTML;
		if (active) entry.el.setAttribute('data-active', 'true');
		else entry.el.removeAttribute('data-active');
		entry.el.style.zIndex = String(active ? Z_ACTIVE : Z_RESTING);
		// Lift goes on the INNER wrapper — the root's transform is MapLibre's positioning and
		// must never be written by us (see buildElement).
		const w = entry.el.firstElementChild as HTMLElement | null;
		if (w) w.style.transform = active && item.kind === 'point' ? 'translateY(-2px)' : '';
	}

	// ── Reconcile ────────────────────────────────────────────────────────────────────────
	// Recompute the display set and reconcile BY KEY — mutate survivors, add new, remove gone.
	// Never clear-and-rebuild: it flickers, drops the selection, and churns listeners. Every
	// removal calls marker.remove(), which is what unregisters the map-level listeners MapLibre
	// bound in addTo(); dropping the reference without it leaks a per-frame `move` handler.
	function render(): void {
		if (!map || !ml) return;
		const items = computeDisplay();
		const seen = new Set<string>();
		for (const item of items) {
			seen.add(item.key);
			const existing = entries.get(item.key);
			if (existing) {
				existing.item = item;
				existing.marker.setLngLat([item.lng, item.lat]);
				repaint(existing);
				continue;
			}
			const el = buildElement(item);
			const marker = new ml.Marker({ element: el, anchor: item.kind === 'cluster' ? 'center' : 'bottom' })
				.setLngLat([item.lng, item.lat])
				.addTo(map);
			entries.set(item.key, { marker, item, el });
		}
		for (const [key, entry] of entries) {
			if (seen.has(key)) continue;
			entry.marker.remove();
			entries.delete(key);
		}
		if (activeKey && !seen.has(activeKey)) activeKey = null;
	}

	// Frame the camera: an explicit host frame (the density core) wins; otherwise all points.
	function fitCamera(duration = 0): void {
		if (!map || !ml) return;
		const bounds = new ml.LngLatBounds();
		if (frame) {
			bounds.extend([frame.sw.lng, frame.sw.lat]);
			bounds.extend([frame.ne.lng, frame.ne.lat]);
		} else {
			if (sourcePoints.length === 0) return;
			for (const p of sourcePoints) bounds.extend([p.lng, p.lat]);
		}
		map.fitBounds(bounds, { padding: fitPadding ?? padDefault(), maxZoom: FIT_MAX_ZOOM, duration });
	}

	// A newly-selected marker must stay clear of the host chrome docked over the map's bottom
	// edge (the listings rail). If its tip would sit under that band, ease the camera up just
	// enough — never a full recenter, which would yank the map on every selection.
	function panActiveClear(): void {
		const m = map;
		if (!m || !activeKey || !activePanPadding) return;
		const src = sourceByKey.get(activeKey);
		if (!src) return;
		const pt = m.project([src.lng, src.lat]);
		const h = m.getContainer().clientHeight;
		const limit = h - activePanPadding.bottom - 16;
		if (pt.y <= limit && pt.y >= 48) return; // already clear (and not off the top)
		const shift = pt.y > limit ? pt.y - limit : pt.y - 48;
		m.easeTo({ center: m.unproject([m.getContainer().clientWidth / 2, h / 2 + shift]), duration: 300 });
	}

	function reindex(next: readonly MapMarkerPoint[]): void {
		sourcePoints = next;
		sourceByKey = new Map(next.map((p) => [p.key, p]));
	}

	function teardown(): void {
		generation++;
		for (const entry of entries.values()) entry.marker.remove();
		entries = new Map();
		map?.remove();
		map = null;
		ml = null;
		activeKey = null;
		sourcePoints = [];
		sourceByKey = new Map();
	}

	return {
		async mount(opts: MapBoardMountOpts) {
			const myGen = ++generation;
			fitPadding = opts.fitPadding;
			activePanPadding = opts.activePanPadding;
			frame = opts.frame ?? null;
			interactive = opts.interactive !== false;
			onSelect = opts.onSelect;
			onClusterSelect = opts.onClusterSelect;
			reindex(opts.points);

			let ns: MaplibreModule;
			try {
				ns = await import('maplibre-gl');
				await import('maplibre-gl/dist/maplibre-gl.css');
			} catch {
				opts.onError?.();
				return;
			}
			if (myGen !== generation) return; // destroyed mid-load
			const lib = ('default' in ns ? ns.default : ns) as MaplibreModule;
			ml = lib;

			const first = opts.points[0];
			const m = new lib.Map({
				container: opts.container,
				style: resolveViewerStyle(),
				center: opts.center
					? [opts.center.lng, opts.center.lat]
					: first
						? [first.lng, first.lat]
						: [FALLBACK_CENTER.lng, FALLBACK_CENTER.lat],
				zoom: opts.zoom ?? FALLBACK_ZOOM,
				attributionControl: false
			});
			map = m;
			// Attribution goes TOP-left (compact ⓘ): the board's host docks a listings rail over
			// the map's bottom edge, and the license text must never sit under it.
			m.addControl(new lib.AttributionControl({ compact: true }), 'top-left');
			if (opts.interactive === false) {
				// Ambient board (homepage band): no zoom chrome, no camera gestures — the band is
				// a picture with clickable pills, never a scroll-trap between page sections.
				m.dragPan.disable();
				m.scrollZoom.disable();
				m.doubleClickZoom.disable();
				m.boxZoom.disable();
				m.keyboard.disable();
				m.touchZoomRotate.disable();
				m.dragRotate.disable();
			} else {
				// Zoom only — no compass. Rotation on a listings map breaks the marker↔rail spatial
				// link and is a state a visitor cannot easily undo.
				m.addControl(new lib.NavigationControl({ showCompass: false }), 'top-right');
				m.dragRotate.disable();
				m.touchZoomRotate.disableRotation();
			}

			let loaded = false;
			m.on('error', (e) => {
				// Always name the real failure in the console — a silent "Live map unavailable"
				// is undiagnosable and invites wrong guesses about providers/keys.
				console.error('[ui-map board]', (e as { error?: Error }).error ?? e);
				// A TILE error (event carries tile/sourceId) is non-fatal — the map still renders
				// with that tile missing. Only a style/engine failure before first load kills the
				// board and shows the fallback.
				const anyE = e as { tile?: unknown; sourceId?: string };
				if (!loaded && !anyE.tile && !anyE.sourceId) {
					opts.onError?.();
					teardown();
				}
			});
			// A click that reaches the canvas missed every marker → clear the selection.
			m.on('click', () => opts.onDeselect?.());
			// Re-decluster after every camera settle — a zoom change moves points in pixel space,
			// so clusters must reform. moveend (not move) keeps it to once per gesture.
			m.on('moveend', () => {
				if (myGen === generation && loaded) render();
			});

			m.on('load', () => {
				if (myGen !== generation) return;
				loaded = true;
				// Sit the basemap into the brand before the first frame the visitor reads.
				applyBrandWaterWash(m);
				if (opts.fitToPoints !== false) fitCamera();
				render();
			});
		},

		setPoints(next: readonly MapMarkerPoint[]) {
			reindex(next);
			render();
		},

		setActive(key: string | null) {
			if (activeKey === key) return;
			activeKey = key;
			// Active state changes a marker's glyph — re-render so the pill picks up the gold fill.
			render();
			if (key) panActiveClear();
		},

		setFrame(next) {
			frame = next;
			fitCamera(400);
		},

		fit() {
			fitCamera(400);
		},

		destroy() {
			teardown();
		}
	};
}
