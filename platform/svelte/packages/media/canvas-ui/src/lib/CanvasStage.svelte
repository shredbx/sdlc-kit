<script lang="ts">
	// The artboard — a real <canvas> that draws the page's visible layers via the
	// kit renderer (resolveFrame → renderPage). Fit-to-viewport with a zoom % badge
	// and a dims badge. Clicking a layer selects it; a selected, non-locked,
	// non-system layer can be DRAGGED to move, RESIZED via 8 DOM-overlay handles
	// (Shift = aspect-lock) and CROPPED via 4 scissor handles on the crop-window
	// edges (Alt-drag on an edge pin = same gesture; dimmed ghost shows the
	// removed strips while dragging). The interaction MATH is the kit's pure
	// geometry (handleRects/resizeLayer/cropLayer, ported from land-canvas); this
	// component owns only the pointer-event binding. Screen px ÷ scale = artboard
	// px; positions round to integers (land-canvas parity). SSR-SAFE: chrome
	// renders on the server; all canvas/pointer access is deferred to
	// onMount/$effect (browser-only).
	import { onMount } from 'svelte';
	import {
		type Page,
		type Layer,
		type SourceRef,
		type FormatterRegistry,
		type Handle,
		type CropEdge,
		type ImageSourceResult,
		type FieldDescriptor,
		type MapStaticUrlBuilder,
		type PreviewBackdrop,
		resolveFrame,
		renderPage,
		renderCropGhost,
		handleRects,
		resizeLayer,
		cropLayer,
		rotationAtPointer,
		pointInLayer,
		tileFillRegion,
		HANDLE_CURSORS,
		IMAGE_FAILED
	} from '@sbx/canvas-kit';
	import { Icon } from '@sbx/core-ui/components/primitives';
	import {
		IMAGE_BIND_MIME,
		FIELD_BIND_MIME,
		PRESET_MIME,
		type ImageBindDrag,
		type FieldBindDrag,
		type PresetDrag
	} from './dnd.js';
	import { placementOf } from './palette.js';

	interface Props {
		page: Page;
		selectedLayerId?: string | null;
		sources?: SourceRef[];
		formatters?: FormatterRegistry;
		readonly?: boolean;
		/** Playhead time in ms — the frame the stage renders (slice-5a animation). */
		time?: number;
		/** R3 animated mode. true → render the playhead frame (motion); false → render
		 *  each layer's static base values (animation tracks ignored). Default true. */
		animated?: boolean;
		/** Maps a layer image src to the URL actually loaded. The kit/UI stay generic;
		 *  the consumer (BR) injects a same-origin proxy rewriter so the canvas is
		 *  export-clean (S-EXPORT). Default = identity (load the src as-is). */
		resolveImageSrc?: (src: string) => string;
		/** Build a keyless /api/map/static URL for a map layer (T4→T5 contract). When
		 *  supplied, resolveFrame bakes map layer `src` so the artboard loads and draws
		 *  the static tile. Without it, map layers render as blank placeholders. */
		buildMapUrl?: MapStaticUrlBuilder;
		/** PREVIEW-ONLY stage backdrop (Slice A2.3 · #0298) — rendered as STAGE CHROME
		 *  *behind* the artboard so the author can judge the transparent mark over a
		 *  colour or a picked image. NOT a layer (never selectable/movable, never in the
		 *  export). Omitted / kind:'none' → nothing renders (the artboard's own checker
		 *  shows). Only threaded in watermark mode (the shell gates it on the feature). The
		 *  image src routes through the SAME `resolveImageSrc` proxy as layer images
		 *  (#0298: never fetch a raw client URL). */
		previewBackdrop?: PreviewBackdrop;
		onselect?: (id: string | null) => void;
		onupdate?: (id: string, patch: Partial<Layer>) => void;
		/** Drag-to-bind (G8): a thumbnail dropped ON an image layer relinks its src. */
		onrebindimage?: (layerId: string, token: string, alias: string) => void;
		/** Drag-to-bind (G8): a thumbnail dropped anywhere else places a new bound image layer. */
		onplaceimage?: (token: string, x: number, y: number, alias: string) => void;
		/** Drag-to-bind (D-1): a field chip dropped ON a text-bindable layer rebinds its content. */
		onrebindfield?: (layerId: string, field: FieldDescriptor, alias?: string) => void;
		/** Drag-to-bind (D-1): a field chip dropped anywhere else places a new bound text layer. */
		onplacefield?: (field: FieldDescriptor, x: number, y: number, alias?: string) => void;
		/** Place a palette preset (drag-drop D-2, OR a placement-tool click/draw). When the
		 *  user click-DRAGS a box tool, w/h carry the drawn size (else the preset default). */
		onplacepreset?: (presetId: string, x: number, y: number, w?: number, h?: number) => void;
		/** The ARMED placement tool's preset id. When set, the cursor is a crosshair and a
		 *  canvas click (point tools) or click-drag (box tools) CREATES the preset where you
		 *  point — instead of selecting/moving. Disarmed by the shell after one placement. */
		activeTool?: string | null;
		/** Cancel the armed tool (e.g. a click on the empty margin around the page). */
		oncanceltool?: () => void;
	}

	let {
		page,
		selectedLayerId = null,
		sources = [],
		formatters = {},
		readonly = false,
		time = 0,
		animated = true,
		resolveImageSrc = (s) => s,
		buildMapUrl,
		previewBackdrop,
		onselect,
		onupdate,
		onrebindimage,
		onplaceimage,
		onrebindfield,
		onplacefield,
		onplacepreset,
		activeTool,
		oncanceltool
	}: Props = $props();

	let viewport = $state<HTMLDivElement | null>(null);
	let canvasEl = $state<HTMLCanvasElement | null>(null);
	let mounted = $state(false);
	let scale = $state(1);
	let viewportW = $state(0);
	let viewportH = $state(0);
	// #0299: the document IS the full-frame overlay, so the preview maps mark-bbox px → display px
	// at the document's native scale (ps = frame.w / page.width) and the frame takes the document's
	// aspect — the same 1:1 relationship the apply pipeline (Slice C) re-tiles at the photo's exact
	// dimensions. No fixed reference photo: what you author at this ratio is what apply stamps.

	// Handle hit-box size in artboard pixels (mirrors land-canvas HANDLE_SIZE).
	const HANDLE_SIZE = 8;

	const zoomPct = $derived(Math.round(scale * 100));

	// --- Drag state (move, resize, crop OR rotate) --------------------------
	// All coordinates are ARTBOARD pixels (screen px ÷ scale). startX/Y is where
	// the drag began; origX/Y/W/H is the layer geometry at drag start. A crop
	// drag (scissor handle / Alt-drag on an edge pin) also seeds the four crop
	// scalars so the kit math is independent of intermediate rounding. A rotate
	// drag needs only origX/Y/W/H (to find the box centre); it sets layer.rotation
	// to the pointer's bearing about that centre, so start/orig offsets aren't read.
	type DragMode = 'move' | 'resize' | 'crop' | 'rotate';
	let drag = $state<{
		mode: DragMode;
		layerId: string;
		handle?: Handle;
		edge?: CropEdge;
		startX: number;
		startY: number;
		origX: number;
		origY: number;
		origW: number;
		origH: number;
		origCrop?: { t: number; r: number; b: number; l: number };
	} | null>(null);

	// Map src FREEZE during a resize gesture — keyed by map layer id. A map's static
	// `src` is baked from its width/height (resolveFrame), so a live resize would
	// re-request a new tile every pointer delta → cache miss → blank flash. While a
	// map is being resized we draw its PRE-DRAG tile (scaled into the live box) and
	// let the crisp tile re-fetch once, on release (frozen entry cleared in onPointerUp).
	let frozenMapSrc = $state<Record<string, string>>({});

	// Placement-tool DRAW gesture (box tools) — from the pointerdown that armed the draw to
	// the live pointer. A point tool (pin) has no draw; a box tool with no drag = click =
	// default size. Rendered as a dashed preview rect while active.
	let toolDraw = $state<{ presetId: string; kind: 'point' | 'box'; startX: number; startY: number; x: number; y: number } | null>(null);

	/** Minimum drag (artboard px) before a box tool DRAWS a sized box; below this a release
	 *  is treated as a click → default size. */
	const TOOL_DRAW_MIN = 6;

	// Disarming the tool (Esc, re-click, margin-cancel, or post-placement) ABORTS any
	// in-flight draw. activeTool lives in the shell and toolDraw here, so without this a
	// draw begun before Esc would still place on release. Capture auto-releases on pointerup.
	$effect(() => {
		if (!activeTool && toolDraw) toolDraw = null;
	});

	// Resolved layers — bindings applied, then (when animated) animation interpolated at
	// the playhead `time`. animated=false → STATIC render: each layer's base values, with
	// its animation tracks ignored entirely (R3 animated-mode OFF). `buildMapUrl` is passed
	// so map layers have their static-map `src` baked before the imageFor cache loads them
	// (T4→T5 contract: without it, placed map layers render blank on the artboard).
	const resolved = $derived(resolveFrame(page, animated ? time : 0, sources, formatters, animated, buildMapUrl));

	// PREVIEW backdrop (Slice A2.3 · #0298) — stage chrome behind the artboard. Active
	// ONLY when a backdrop prop is supplied with a non-'none' kind AND it carries the
	// matching face (a colour, or an image src). The image src is routed through the SAME
	// hardened `resolveImageSrc` proxy the canvas uses for layer images (#0298: never a
	// raw client URL). null = nothing renders → the artboard's checker shows through. This
	// is purely a CSS layer; it never enters `resolved` / the canvas / the export.
	const backdrop = $derived.by(() => {
		const b = previewBackdrop;
		if (!b || b.kind === 'none') return null;
		if (b.kind === 'color') return b.color ? { color: b.color } : null;
		if (b.kind === 'image') return b.imageSrc ? { imageSrc: resolveImageSrc(b.imageSrc) } : null;
		return null;
	});

	// Fit the artboard inside the viewport, leaving a uniform inset on all sides. The
	// full-width TopToolbar now owns the top chrome at the editor level (R2), so the
	// canvas area no longer reserves a top zone for a floating cluster. These must match
	// the .stage padding below (the artboard centres in the padded content box; the fit
	// scale must target that same box).
	const INSET_X = 24;
	const INSET_TOP = 24;
	const INSET_BOTTOM = 24;
	function computeScale(): void {
		if (!viewport) return;
		const rect = viewport.getBoundingClientRect();
		viewportW = rect.width;
		viewportH = rect.height;
		const availW = Math.max(0, rect.width - INSET_X * 2);
		const availH = Math.max(0, rect.height - INSET_TOP - INSET_BOTTOM);
		if (availW <= 0 || availH <= 0) return;
		scale = Math.min(availW / page.width, availH / page.height, 1);
	}

	// --- Image-layer source cache -----------------------------------------
	// The kit renderer is HEADLESS: renderImageLayer draws from a preloaded cache and
	// never loads images itself (so @sbx/canvas-kit imports no DOM image API). The UI
	// owns loading. Each distinct src is fetched ONCE (keyed by src); a redraw is
	// scheduled on load so the image appears without a manual refresh, and a failed
	// load reports IMAGE_FAILED so the kit renders an explicit broken-image state
	// (I-2 — silent blanks are banned). Failure is pinned per resolved URL for the
	// component lifetime; re-opening the document retries. (Panel thumbs retry on
	// any grid remount — two retry lifetimes for the same URL is intentional.)
	//
	// EXPORT-CLEAN loading (S-EXPORT). The media CDN sends no Access-Control-Allow-
	// Origin, so a directly loaded cover taints the canvas → toBlob/toDataURL throw.
	// The consumer injects `resolveImageSrc` to route the src through a SAME-ORIGIN
	// proxy; we then set crossOrigin='anonymous' (harmless same-origin; future-proofs
	// a CDN-CORS switch) so the canvas stays untainted and exportable. With the
	// default identity resolver the behaviour is unchanged (on-screen only).
	// The cache is keyed by the RESOLVED url so a proxied + a raw draw never collide.
	const imageCache = new Map<string, { img: HTMLImageElement; ready: boolean; failed: boolean }>();

	function imageFor(src: string): ImageSourceResult {
		if (!src) return undefined;
		const url = resolveImageSrc(src);
		const hit = imageCache.get(url);
		if (hit) return hit.failed ? IMAGE_FAILED : hit.ready ? hit.img : undefined;
		const entry = { img: new Image(), ready: false, failed: false };
		imageCache.set(url, entry);
		entry.img.crossOrigin = 'anonymous';
		entry.img.onload = () => {
			entry.ready = true;
			scheduleRedraw();
		};
		entry.img.onerror = () => {
			// Signal failure + redraw so the kit paints the broken-image state.
			entry.failed = true;
			scheduleRedraw();
		};
		entry.img.src = url;
		return undefined; // not ready this frame
	}

	/** Redraw the artboard. Called on every image-cache load so a late-arriving photo (a layer
	 *  image OR the CSS preview backdrop) lands without a manual refresh. */
	function scheduleRedraw(): void {
		draw();
	}

	function draw(): void {
		if (!canvasEl) return;
		const ctx = canvasEl.getContext('2d');
		if (!ctx) return;
		const dpr = window.devicePixelRatio || 1;
		canvasEl.width = page.width * dpr;
		canvasEl.height = page.height * dpr;
		ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
		ctx.clearRect(0, 0, page.width, page.height);
		// Substitute frozen map tiles while a map is being resized (B2) — the live box
		// scales the last crisp bitmap instead of blanking on each per-delta re-request.
		const display =
			Object.keys(frozenMapSrc).length > 0
				? resolved.map((l) => (l.type === 'map' && frozenMapSrc[l.id] ? { ...l, src: frozenMapSrc[l.id] } : l))
				: resolved;
		renderPage(ctx, page, display, imageFor);
		// Dimmed ghost of the cropped-away strips while a crop gesture is live —
		// the user keeps visual reference of what the scissors are removing.
		if (drag?.mode === 'crop') {
			const ghost = resolved.find((l) => l.id === drag!.layerId);
			if (ghost) renderCropGhost(ctx, ghost, imageFor);
		}
	}

	onMount(() => {
		mounted = true;
		computeScale();
		draw();
		const ro = new ResizeObserver(() => {
			computeScale();
		});
		if (viewport) ro.observe(viewport);
		// Re-draw whenever a web font finishes loading (#3): a layer's Google font may land
		// AFTER the first paint (the canvas draws fallback otherwise — fonts load async). The
		// kit renderer reads the now-available face on the next draw. Best-effort; FontFaceSet.
		const onFontsDone = () => {
			if (mounted) draw();
		};
		const fontSet = typeof document !== 'undefined' ? document.fonts : undefined;
		fontSet?.addEventListener('loadingdone', onFontsDone);
		void fontSet?.ready.then(onFontsDone);
		return () => {
			ro.disconnect();
			fontSet?.removeEventListener('loadingdone', onFontsDone);
		};
	});

	// Re-draw when the resolved layers change (after mount). draw() also READS
	// `drag` (the crop-ghost branch), so this one effect re-runs on drag
	// start/end too — the ghost appears the moment the scissors are grabbed and
	// vanishes on release, no second effect needed (a duplicate would double
	// every full-canvas draw).
	$effect(() => {
		void resolved;
		if (mounted) draw();
	});

	// Force the crop cursor globally while a crop gesture is live — the scissor
	// tracks the moving edge, but rounding lag would flicker the cursor across
	// the canvas otherwise (same pattern as the Inspector's ScrubUnit).
	$effect(() => {
		if (drag?.mode !== 'crop' || !drag.edge) return;
		const cls = drag.edge === 'n' || drag.edge === 's' ? 'cv-crop-ns' : 'cv-crop-ew';
		document.body.classList.add(cls);
		return () => document.body.classList.remove(cls);
	});

	// Force the grabbing cursor globally while a rotate gesture is live (same
	// pattern as the crop cursor) — the grip moves with the pointer, so the
	// cursor must not flicker back to default over the rest of the canvas.
	$effect(() => {
		if (drag?.mode !== 'rotate') return;
		document.body.classList.add('cv-rotating');
		return () => document.body.classList.remove('cv-rotating');
	});

	// Re-fit the zoom when the page dimensions change (e.g. after a Resize). The
	// backing canvas redraws via the resolved effect, but the fit scale must be
	// recomputed too — otherwise the artboard keeps the old zoom and overflows or
	// underflows the viewport instead of re-fitting.
	$effect(() => {
		void page.width;
		void page.height;
		if (mounted) computeScale();
	});

	/** Convert a pointer OR drag event to artboard pixels (screen px ÷ scale). Structural
	 *  param so both PointerEvent and DragEvent (drag-to-bind, G8) can use it. */
	function toArtboard(event: { clientX: number; clientY: number }): { x: number; y: number } | null {
		if (!canvasEl) return null;
		const rect = canvasEl.getBoundingClientRect();
		return { x: (event.clientX - rect.left) / scale, y: (event.clientY - rect.top) / scale };
	}

	/** Hit-test in artboard pixels — rotation- AND tile-aware (kit geometry). A click on
	 *  any tiled COPY resolves to its source layer (copies are pure repeats, never their
	 *  own selectable layers), and a rotated layer is tested against its real footprint,
	 *  not a stale AABB. The fill region matches what renderPage drew, so selection and
	 *  render never drift. */
	function hitTest(layer: Layer, px: number, py: number): boolean {
		return pointInLayer(layer, px, py, layer.tile ? tileFillRegion(page, layer) : undefined);
	}

	/** Every non-hidden layer under the point, front→back (front = end of array). The
	 *  list drives both the plain pick (its head) and Alt-click drill-down (step through
	 *  it), so a layer hidden under a page-filling tiled mark is still reachable. */
	function pickLayers(px: number, py: number): Layer[] {
		const hits: Layer[] = [];
		for (let i = resolved.length - 1; i >= 0; i--) {
			const layer = resolved[i];
			if (layer.visible === false) continue;
			if (hitTest(layer, px, py)) hits.push(layer);
		}
		return hits;
	}

	/** Top-most non-hidden layer under the point, or null (front = end of array). */
	function pickLayer(px: number, py: number): Layer | null {
		return pickLayers(px, py)[0] ?? null;
	}

	/** The selected layer's base geometry (resolved copy carries x/y/w/h). */
	const selectedResolved = $derived(
		selectedLayerId ? (resolved.find((l) => l.id === selectedLayerId) ?? null) : null
	);

	/** A selected, non-locked, non-system layer is interactive (move + resize). */
	const interactiveSelected = $derived(
		!readonly && selectedResolved && !selectedResolved.locked && !selectedResolved.system
			? selectedResolved
			: null
	);

	// Fixed on-screen handle size (px) so handles stay grabbable at any zoom — the
	// hit math stays in artboard space; only the rendered box is a constant size.
	const HANDLE_SCREEN_SIZE = 10;

	/** The 8 resize handles for the selected layer, positioned for the DOM overlay.
	 *  Each handle's CENTRE is the scaled artboard handle point; the box is a fixed
	 *  screen size. Only shown for an interactive layer with a known box. */
	const handles = $derived.by(() => {
		const layer = interactiveSelected;
		if (!layer || layer.width == null || layer.height == null) return [];
		const half = HANDLE_SCREEN_SIZE / 2;
		return handleRects(layer, HANDLE_SIZE).map((r) => {
			// handleRects boxes are centred on the handle point → centre = r.x + r.w/2.
			const cx = (r.x + r.w / 2) * scale;
			const cy = (r.y + r.h / 2) * scale;
			return {
				handle: r.handle,
				left: cx - half,
				top: cy - half,
				size: HANDLE_SCREEN_SIZE,
				cursor: HANDLE_CURSORS[r.handle]
			};
		});
	});

	// --- Crop scissors (user directive 2026-06-07) --------------------------
	// One scissor per edge, sitting ON the live crop-window edge (direct
	// manipulation: after a crop it grabs the visible edge, and it tracks the
	// edge under the pointer during the drag). Offset along the edge from the
	// window midpoint so it never crowds the resize pin; an edge too short on
	// screen to fit pin + scissor drops its pair.
	const SCISSOR_SIZE = 18;
	const SCISSOR_OFFSET = 22; // centre offset along the edge from the window midpoint
	const SCISSOR_MIN_SIDE = 64; // screen px — below this the pair is dropped

	const scissors = $derived.by(() => {
		const layer = interactiveSelected;
		if (!layer || layer.width == null || layer.height == null) return [];
		const t = layer.crop_top ?? 0;
		const r = layer.crop_right ?? 0;
		const b = layer.crop_bottom ?? 0;
		const l = layer.crop_left ?? 0;
		// The crop WINDOW (box minus insets) in screen px.
		const wx = (layer.x + l) * scale;
		const wy = (layer.y + t) * scale;
		const ww = Math.max(0, layer.width - l - r) * scale;
		const wh = Math.max(0, layer.height - t - b) * scale;
		const half = SCISSOR_SIZE / 2;
		const out: { edge: CropEdge; left: number; top: number; cursor: string }[] = [];
		if (ww >= SCISSOR_MIN_SIDE) {
			out.push({ edge: 'n', left: wx + ww / 2 + SCISSOR_OFFSET - half, top: wy - half, cursor: 'ns-resize' });
			out.push({ edge: 's', left: wx + ww / 2 + SCISSOR_OFFSET - half, top: wy + wh - half, cursor: 'ns-resize' });
		}
		if (wh >= SCISSOR_MIN_SIDE) {
			out.push({ edge: 'w', left: wx - half, top: wy + wh / 2 + SCISSOR_OFFSET - half, cursor: 'ew-resize' });
			out.push({ edge: 'e', left: wx + ww - half, top: wy + wh / 2 + SCISSOR_OFFSET - half, cursor: 'ew-resize' });
		}
		return out;
	});

	// --- Rotation grip (user directive 2026-06-08; relocated 2026-06-13 #6) --
	// ONE grip that orbits the layer (Shift snaps 15°). Rotation has a single
	// degree of freedom, so one grip gives full 360° (unlike the four scissors,
	// which each crop a different edge). It rests just LEFT of the top-centre
	// pin, ON the top edge — NOT floating above it — so it stays in view when
	// the layer hugs the artboard top (the old upward offset was clipped by the
	// stage's overflow:hidden, #6). Mirror of the 'n' crop scissor on the right.
	// The whole overlay (box + handles + scissors + grip) rotates WITH the layer
	// via `selOverlay`, so the grip rides that transform glued to the content.
	const ROTATE_SIZE = 20; // grip chip diameter (screen px)
	const ROTATE_GAP = 22; // grip centre offset LEFT of the box top-centre (mirrors SCISSOR_OFFSET)

	const rotateGrip = $derived.by(() => {
		const layer = interactiveSelected;
		if (!layer || layer.width == null || layer.height == null) return null;
		const cx = (layer.x + layer.width / 2) * scale;
		const topY = layer.y * scale;
		// Rest just left of the top-centre pin, on the top edge — always within the artboard.
		return { gx: cx - ROTATE_GAP, gy: topY };
	});

	// Transform that rotates the selection overlay group to track the layer.
	// transform-origin is the layer's box CENTRE in scaled artboard px — the same
	// pivot the kit renderer uses (applyRotation), so chrome and content stay
	// aligned at any angle. Shown for ANY selected layer (the box rotates even
	// for locked/system layers); the interactive handles/scissors/grip only
	// render when interactiveSelected, so non-interactive layers show just a
	// rotated outline.
	const selOverlay = $derived.by(() => {
		if (!selectedResolved) return null;
		const w = selectedResolved.width ?? 0;
		const h = selectedResolved.height ?? 0;
		return {
			cx: (selectedResolved.x + w / 2) * scale,
			cy: (selectedResolved.y + h / 2) * scale,
			rotation: selectedResolved.rotation ?? 0
		};
	});

	// --- Pointer interaction (move / resize / crop) -------------------------

	/** Begin a crop drag for one edge — from its scissor handle, or Alt-drag on
	 *  the matching edge resize pin (the Figma-muscle-memory route). */
	function onCropPointerDown(event: PointerEvent, edge: CropEdge): void {
		if (event.button !== 0) return; // a context-menu mousedown must never start (and strand) a gesture
		const layer = interactiveSelected;
		if (!layer || layer.width == null || layer.height == null) return;
		const pt = toArtboard(event);
		if (!pt) return;
		event.preventDefault();
		event.stopPropagation();
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		drag = {
			mode: 'crop',
			layerId: layer.id,
			edge,
			startX: pt.x,
			startY: pt.y,
			origX: layer.x,
			origY: layer.y,
			origW: layer.width,
			origH: layer.height,
			origCrop: {
				t: layer.crop_top ?? 0,
				r: layer.crop_right ?? 0,
				b: layer.crop_bottom ?? 0,
				l: layer.crop_left ?? 0
			}
		};
	}

	/** Begin a rotate drag from the rotation grip. The gesture seeds only the
	 *  box centre (origX/Y/W/H); onPointerMove sets layer.rotation to the
	 *  pointer's bearing about that centre (Shift snaps to 15°). */
	function onRotatePointerDown(event: PointerEvent): void {
		if (event.button !== 0) return; // primary only — same stranded-gesture guard
		const layer = interactiveSelected;
		if (!layer || layer.width == null || layer.height == null) return;
		event.preventDefault();
		event.stopPropagation();
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		drag = {
			mode: 'rotate',
			layerId: layer.id,
			startX: 0,
			startY: 0,
			origX: layer.x,
			origY: layer.y,
			origW: layer.width,
			origH: layer.height
		};
	}

	/** Begin a resize drag from a handle (the handle div forwards this).
	 *  Alt on an EDGE pin re-routes to the crop gesture for that edge — same
	 *  drag, scissors semantics (corner pins stay pure resize). */
	function onHandlePointerDown(event: PointerEvent, handle: Handle): void {
		if (event.button !== 0) return; // primary only — same stranded-gesture guard as the canvas
		if (event.altKey && (handle === 'n' || handle === 'e' || handle === 's' || handle === 'w')) {
			onCropPointerDown(event, handle);
			return;
		}
		const layer = interactiveSelected;
		if (!layer || layer.width == null || layer.height == null) return;
		const pt = toArtboard(event);
		if (!pt) return;
		event.preventDefault();
		event.stopPropagation();
		(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
		drag = {
			mode: 'resize',
			layerId: layer.id,
			handle,
			startX: pt.x,
			startY: pt.y,
			origX: layer.x,
			origY: layer.y,
			origW: layer.width,
			origH: layer.height
		};
		// Freeze a map's current static tile for the duration of the resize (B2) — the
		// box scales the existing bitmap; the crisp re-fetch fires on release.
		if (layer.type === 'map') {
			const src = resolved.find((l) => l.id === layer.id)?.src;
			if (src) frozenMapSrc = { ...frozenMapSrc, [layer.id]: src };
		}
	}

	/** Pointerdown on the canvas: select the top-most layer and (if interactive)
	 *  begin a move drag. Empty space clears the selection. */
	function onCanvasPointerDown(event: PointerEvent): void {
		if (event.button !== 0) return;
		const pt = toArtboard(event);
		if (!pt) return;
		// Placement tool armed: begin the create gesture at the pointer — over the map,
		// empty canvas, or any layer. A point tool places on release at the down point; a
		// box tool draws a rect (or places default size on a click). Never selects/moves.
		if (activeTool && !readonly) {
			event.preventDefault();
			canvasEl?.setPointerCapture(event.pointerId);
			toolDraw = { presetId: activeTool, kind: placementOf(activeTool).kind, startX: pt.x, startY: pt.y, x: pt.x, y: pt.y };
			return;
		}
		// Front→back stack under the cursor. Plain click takes the top; Alt-click DRILLS
		// to the next layer beneath (wrapping) so a layer occluded by a page-filling tiled
		// mark — or anything overlapping — is selectable on the canvas, not only via the
		// Layers panel (real-editor "select-below"). Alt is read-only on selection; it
		// never starts a different gesture.
		const stack = pickLayers(pt.x, pt.y);
		if (stack.length === 0) {
			onselect?.(null);
			return;
		}
		let hit = stack[0];
		if (event.altKey && stack.length > 1) {
			const cur = stack.findIndex((l) => l.id === selectedLayerId);
			hit = stack[(cur + 1) % stack.length]; // next one beneath the current pick, wrapping
		}
		if (hit.id !== selectedLayerId) onselect?.(hit.id);
		// Locked / system / readonly layers select but never move.
		if (readonly || hit.locked || hit.system) return;
		event.preventDefault();
		canvasEl?.setPointerCapture(event.pointerId);
		drag = {
			mode: 'move',
			layerId: hit.id,
			startX: pt.x,
			startY: pt.y,
			origX: hit.x,
			origY: hit.y,
			origW: hit.width ?? 0,
			origH: hit.height ?? 0
		};
	}

	function onPointerMove(event: PointerEvent): void {
		// Placement-tool draw gesture (box tools) — track the live corner for the preview.
		if (toolDraw) {
			const p = toArtboard(event);
			if (p) toolDraw = { ...toolDraw, x: p.x, y: p.y };
			return;
		}
		if (!drag) return;
		const pt = toArtboard(event);
		if (!pt) return;
		const dx = pt.x - drag.startX;
		const dy = pt.y - drag.startY;
		if (drag.mode === 'move') {
			onupdate?.(drag.layerId, { x: Math.round(drag.origX + dx), y: Math.round(drag.origY + dy) });
		} else if (drag.mode === 'resize' && drag.handle) {
			// Pure resize math lives in the kit; resizeLayer reads only x/y/width/
			// height, so feed it the drag-start geometry — the result is then
			// independent of intermediate rounding.
			// KNOWN LIMITATION (S-ROTRESIZE, follow-up): dx/dy are ARTBOARD-axis
			// deltas, not the layer's rotated frame — so resizing a rotated layer
			// grows along screen axes, not the rotated edge. Pre-dates the rotation
			// grip (rotation was already settable via the Inspector); rotating the
			// chrome only made it visible. Fix = rotate the delta by −layer.rotation
			// before resizeLayer (its own slice + tests).
			const seed = { x: drag.origX, y: drag.origY, width: drag.origW, height: drag.origH } as Layer;
			// Free resize (Shift = aspect-lock) for EVERY layer, maps included. A map is a
			// croppable viewport: dragging an edge reveals more/less geography at constant
			// zoom (the static src re-requests at the new size). The src is FROZEN to the
			// last tile during the gesture (frozenMapSrc) so the box scales smoothly and the
			// crisp tile re-fetches once on release — no per-delta reload blank.
			const geo = resizeLayer(seed, drag.handle, dx, dy, event.shiftKey);
			onupdate?.(drag.layerId, geo);
		} else if (drag.mode === 'rotate') {
			// Bearing of the pointer about the box centre = the new rotation. Shift
			// snaps to 15°. The kit's rotationAtPointer normalises to [0, 360).
			const cx = drag.origX + drag.origW / 2;
			const cy = drag.origY + drag.origH / 2;
			onupdate?.(drag.layerId, { rotation: rotationAtPointer(cx, cy, pt.x, pt.y, event.shiftKey ? 15 : 0) });
		} else if (drag.mode === 'crop' && drag.edge && drag.origCrop) {
			// Same seed discipline for the crop math: gesture-start crop + size in,
			// one clamped scalar out ([0, side − oppositeCrop − 1] — never inverts).
			const seed = {
				width: drag.origW,
				height: drag.origH,
				crop_top: drag.origCrop.t,
				crop_right: drag.origCrop.r,
				crop_bottom: drag.origCrop.b,
				crop_left: drag.origCrop.l
			} as Layer;
			onupdate?.(drag.layerId, cropLayer(seed, drag.edge, dx, dy));
		}
	}

	function onPointerUp(event: PointerEvent): void {
		// Finalize a placement-tool gesture: a real drag on a box tool CREATES a sized box;
		// a click (point tool, or a box tool without a real drag) creates at default size.
		if (toolDraw) {
			try {
				canvasEl?.releasePointerCapture(event.pointerId);
			} catch {
				/* capture may already be released — ignore. */
			}
			const td = toolDraw;
			toolDraw = null;
			const w = Math.abs(td.x - td.startX);
			const h = Math.abs(td.y - td.startY);
			// A real draw = a box tool dragged past the threshold on EITHER axis (matches the
			// preview gate in `toolDrawBox` exactly, so what you see is what gets created).
			if (td.kind === 'box' && (w >= TOOL_DRAW_MIN || h >= TOOL_DRAW_MIN)) {
				onplacepreset?.(td.presetId, Math.min(td.x, td.startX) + w / 2, Math.min(td.y, td.startY) + h / 2, w, h);
			} else {
				onplacepreset?.(td.presetId, td.startX, td.startY);
			}
			return;
		}
		if (!drag) return;
		try {
			(event.currentTarget as HTMLElement).releasePointerCapture(event.pointerId);
		} catch {
			// capture may already be released — ignore.
		}
		// Thaw any frozen map tile — the final-size crisp src (resolved) takes over and
		// re-fetches once. Clear ALL (only ever one active resize) so a tile can never
		// leak frozen past its gesture.
		if (Object.keys(frozenMapSrc).length > 0) frozenMapSrc = {};
		drag = null;
	}

	// Selection outline rectangle, in scaled artboard pixels, for the selected layer.
	const selectionBox = $derived.by(() => {
		if (!selectedResolved) return null;
		return {
			left: selectedResolved.x * scale,
			top: selectedResolved.y * scale,
			width: (selectedResolved.width ?? 0) * scale,
			height: (selectedResolved.height ?? 0) * scale
		};
	});

	// --- Drag-to-bind/place (G8 + D-1 + D-2) — a Media/Map thumbnail, a Texts-panel field
	//     chip, or a palette ComponentCard dropped on the artboard. Reuses the pointer
	//     hit-test (toArtboard + pickLayer). Drop matrix (see dnd.ts): a same-kind layer
	//     under the cursor is the REPLACE target (image→relink src · field→rebind content)
	//     and shows the dashed highlight while hovered; anywhere else the drop PLACES at
	//     the point (presets are place-only — a new component never replaces).
	let dropTargetId = $state<string | null>(null);
	// The kind of the live drag while it hovers a REPLACE target — drives the
	// target box's label ("Replace image" / "Replace text") so a repoint reads
	// unmistakably as a replace, never an accidental new drop. null when the drag
	// would PLACE (empty artboard) or carries no replace-capable payload.
	let dropKind = $state<'image' | 'field' | null>(null);

	/** Parse a kind's payload off the drop event (data is only readable on drop). */
	function dragPayload<T>(event: DragEvent, mime: string): T | null {
		const raw = event.dataTransfer?.getData(mime);
		if (!raw) return null;
		try {
			return JSON.parse(raw) as T;
		} catch {
			return null;
		}
	}

	/** A field payload rebinds text-bindable layers — same predicate as the chip-click
	 *  flow (text + callout carry a bindable `content`). */
	function textBindable(layer: Layer): boolean {
		return layer.type === 'text' || layer.type === 'callout';
	}

	/** The payload kind of a live drag — from `dataTransfer.types` (the only thing
	 *  HTML5 dnd exposes during dragover), so the highlight can be kind-aware. */
	function dragKind(event: DragEvent): 'image' | 'field' | 'preset' | null {
		const types = event.dataTransfer?.types;
		if (!types) return null;
		if (types.includes(IMAGE_BIND_MIME)) return 'image';
		if (types.includes(FIELD_BIND_MIME)) return 'field';
		if (types.includes(PRESET_MIME)) return 'preset';
		return null;
	}

	/** The replace target under the cursor for a payload kind — a preset never
	 *  replaces (place-only), so it never rings a layer. The system BACKGROUND is
	 *  never a replace target either (it is full-bleed — every stray drop would
	 *  rebind it); backgrounds change only via their explicit affordances. */
	function replaceTarget(kind: 'image' | 'field' | 'preset', hit: Layer | null): string | null {
		if (!hit || hit.system) return null;
		if (kind === 'image' && hit.type === 'image') return hit.id;
		if (kind === 'field' && textBindable(hit)) return hit.id;
		return null;
	}

	function onStageDragOver(event: DragEvent): void {
		const kind = readonly ? null : dragKind(event);
		if (!kind) return;
		event.preventDefault(); // permit the drop
		event.dataTransfer!.dropEffect = 'copy';
		const pt = toArtboard(event);
		// Highlight ONLY a replace target (same-kind layer); a place-drop shows no ring.
		dropTargetId = replaceTarget(kind, pt ? pickLayer(pt.x, pt.y) : null);
		dropKind = dropTargetId && (kind === 'image' || kind === 'field') ? kind : null;
	}

	function onStageDragLeave(): void {
		dropTargetId = null;
		dropKind = null;
	}

	function onStageDrop(event: DragEvent): void {
		dropTargetId = null; // clear the drop highlight whatever happens next
		dropKind = null;
		if (readonly) return;
		const pt = toArtboard(event);
		if (!pt) return;
		// Shape-guard the parsed payloads: a foreign drag could carry our MIME with
		// valid-JSON-wrong-shape data — treat it as no payload, never throw downstream.
		const rawImage = dragPayload<ImageBindDrag>(event, IMAGE_BIND_MIME);
		const image =
			rawImage && typeof rawImage.token === 'string' && typeof rawImage.alias === 'string'
				? rawImage
				: null;
		const rawField = dragPayload<FieldBindDrag>(event, FIELD_BIND_MIME);
		const field = rawField && typeof rawField.field?.token === 'string' ? rawField : null;
		const rawPreset = dragPayload<PresetDrag>(event, PRESET_MIME);
		const preset = rawPreset && typeof rawPreset.presetId === 'string' ? rawPreset : null;
		if (!image && !field && !preset) return; // unrelated drop — leave the browser default alone
		event.preventDefault();
		// The background never catches a drop — it reads as empty artboard (place).
		const raw = pickLayer(pt.x, pt.y);
		const hit = raw?.system ? null : raw;
		if (image) {
			if (hit && hit.type === 'image') {
				onrebindimage?.(hit.id, image.token, image.alias); // relink the top image layer
			} else {
				onplaceimage?.(image.token, pt.x, pt.y, image.alias); // new image layer at the point
			}
		} else if (field) {
			if (hit && textBindable(hit)) {
				onrebindfield?.(hit.id, field.field, field.alias); // rebind the top text layer
			} else {
				onplacefield?.(field.field, pt.x, pt.y, field.alias); // new text layer at the point
			}
		} else if (preset) {
			onplacepreset?.(preset.presetId, pt.x, pt.y); // place-only — whatever is under it
		}
	}

	// Highlight box for the layer under a live drag (the relink/rebind target).
	const dropBox = $derived.by(() => {
		if (!dropTargetId) return null;
		const l = resolved.find((r) => r.id === dropTargetId);
		if (!l) return null;
		return {
			left: l.x * scale,
			top: l.y * scale,
			width: (l.width ?? 0) * scale,
			height: (l.height ?? 0) * scale
		};
	});

	// Dashed preview rect for an in-progress box-tool draw (scaled to screen px); null until
	// the drag passes the click/draw threshold (so a plain click shows no rect).
	const toolDrawBox = $derived.by(() => {
		if (!toolDraw || toolDraw.kind !== 'box') return null;
		const w = Math.abs(toolDraw.x - toolDraw.startX);
		const h = Math.abs(toolDraw.y - toolDraw.startY);
		if (w < TOOL_DRAW_MIN && h < TOOL_DRAW_MIN) return null;
		return {
			left: Math.min(toolDraw.x, toolDraw.startX) * scale,
			top: Math.min(toolDraw.y, toolDraw.startY) * scale,
			width: w * scale,
			height: h * scale
		};
	});

	/** Pointerdown on the bare margin AROUND the page: clear the selection and cancel any
	 *  armed tool (a deliberate click into empty space — the universal "deselect" gesture).
	 *  Guarded to the margin element itself so bubbled canvas/handle events never trigger it. */
	function onStageBackgroundPointerDown(event: PointerEvent): void {
		if (event.target !== event.currentTarget) return;
		if (activeTool) oncanceltool?.();
		onselect?.(null);
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="stage" bind:this={viewport} onpointerdown={onStageBackgroundPointerDown}>
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="stage__artboard"
		class:stage__artboard--dropping={!!dropTargetId}
		class:stage__artboard--backdrop={!!backdrop}
		style="width: {page.width * scale}px; height: {page.height * scale}px;"
		ondragover={onStageDragOver}
		ondragleave={onStageDragLeave}
		ondrop={onStageDrop}
	>
		{#if backdrop}
			<!-- PREVIEW-ONLY stage backdrop (#0298) — a colour fill OR a cover-fit picked image
			     rendered BEHIND the canvas so the transparent mark previews over it. Pure CSS
			     chrome: pointer-events none, never selectable, never in the layers / export. -->
			<div
				class="stage__backdrop"
				style={backdrop.color
					? `background-color: ${backdrop.color};`
					: `background-image: url('${backdrop.imageSrc}'); background-size: cover; background-position: center;`}
				aria-hidden="true"
			></div>
		{/if}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<canvas
			bind:this={canvasEl}
			class="stage__canvas"
			class:stage__canvas--dragging={drag?.mode === 'move'}
			class:stage__canvas--placing={!!activeTool && !readonly}
			style="width: {page.width * scale}px; height: {page.height * scale}px;"
			onpointerdown={onCanvasPointerDown}
			onpointermove={onPointerMove}
			onpointerup={onPointerUp}
			onpointercancel={onPointerUp}
			onlostpointercapture={onPointerUp}
		></canvas>
		{#if dropBox}
			<!-- Drag-to-bind: the same-kind layer under the cursor will be relinked/rebound on
			     drop. The label makes the REPLACE explicit so a repoint is never mistaken for a
			     new drop (user directive 2026-06-15). -->
			<div
				class="stage__drop"
				style="left: {dropBox.left}px; top: {dropBox.top}px; width: {dropBox.width}px; height: {dropBox.height}px;"
			>
				{#if dropKind}
					<span class="stage__drop-label">
						{dropKind === 'image' ? 'Replace image' : 'Replace text'}
					</span>
				{/if}
			</div>
		{/if}
		{#if toolDrawBox}
			<!-- Placement tool: the box being drawn (dashed) before it's created on release. -->
			<div
				class="stage__tooldraw"
				style="left: {toolDrawBox.left}px; top: {toolDrawBox.top}px; width: {toolDrawBox.width}px; height: {toolDrawBox.height}px;"
			></div>
		{/if}
		<!-- Selection overlay — the box, resize handles, crop scissors and rotation
		     grip, ALL rotated together about the layer's box centre (the same pivot
		     the kit renderer uses) so the chrome stays glued to rotated content.
		     pointer-events: none on the group; only the interactive chips opt back in. -->
		{#if selectionBox && selOverlay}
			<div
				class="stage__overlay"
				style="transform-origin: {selOverlay.cx}px {selOverlay.cy}px; transform: rotate({selOverlay.rotation}deg);"
			>
				<div
					class="stage__selection"
					style="left: {selectionBox.left}px; top: {selectionBox.top}px; width: {selectionBox.width}px; height: {selectionBox.height}px;"
				></div>
				<!-- Resize handles — DOM overlay over the artboard (positioned, accessible),
				     not painted on the canvas. Only for an interactive selected layer. -->
				{#each handles as h (h.handle)}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="stage__handle"
						style="left: {h.left}px; top: {h.top}px; width: {h.size}px; height: {h.size}px; cursor: {h.cursor};"
						onpointerdown={(e) => onHandlePointerDown(e, h.handle)}
						onpointermove={onPointerMove}
						onpointerup={onPointerUp}
						onpointercancel={onPointerUp}
						onlostpointercapture={onPointerUp}
					></div>
				{/each}
				<!-- Crop scissors — one per crop-window edge, offset from the resize pin.
				     Dragging cuts inward from 0 to the opposite edge; the dimmed ghost of
				     the removed strips draws on the canvas while the gesture is live. -->
				{#each scissors as s (s.edge)}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="stage__scissor"
						class:stage__scissor--side={s.edge === 'e' || s.edge === 'w'}
						style="left: {s.left}px; top: {s.top}px; width: {SCISSOR_SIZE}px; height: {SCISSOR_SIZE}px; cursor: {s.cursor};"
						title="Drag to crop (Alt-drag the edge pin works too)"
						onpointerdown={(e) => onCropPointerDown(e, s.edge)}
						onpointermove={onPointerMove}
						onpointerup={onPointerUp}
						onpointercancel={onPointerUp}
						onlostpointercapture={onPointerUp}
					><Icon name="scissors" size="xs" /></div>
				{/each}
				<!-- Rotation grip — one chip just left of the top-centre pin, on the top edge
				     (always in view, #6). Drag to orbit the layer (Shift snaps 15°). -->
				{#if rotateGrip}
					<!-- svelte-ignore a11y_no_static_element_interactions -->
					<div
						class="stage__rotate"
						style="left: {rotateGrip.gx - ROTATE_SIZE / 2}px; top: {rotateGrip.gy - ROTATE_SIZE / 2}px; width: {ROTATE_SIZE}px; height: {ROTATE_SIZE}px;"
						title="Drag to rotate (hold Shift to snap 15°)"
						onpointerdown={onRotatePointerDown}
						onpointermove={onPointerMove}
						onpointerup={onPointerUp}
						onpointercancel={onPointerUp}
						onlostpointercapture={onPointerUp}
					><Icon name="rotate-cw" size="xs" /></div>
				{/if}
			</div>
		{/if}
	</div>

	<div class="stage__badge stage__badge--zoom">{zoomPct}%</div>
	<div class="stage__badge stage__badge--dims">{page.width}×{page.height}</div>
</div>

<style>
	.stage {
		position: relative;
		flex: 1;
		display: flex;
		align-items: center;
		justify-content: center;
		min-width: 0;
		/* Uniform inset on all sides; the artboard centres in this padded content box
		   (matches INSET_* in computeScale). */
		box-sizing: border-box;
		padding: 24px;
		background: var(--cv-color-neutral-100, #f5f5f5);
		overflow: hidden;
	}

	.stage__artboard {
		position: relative;
		/* Own stacking context (#0298) so the preview backdrop's z-index:-1 is contained:
		   above the artboard background, below the in-flow canvas + positioned overlays. */
		isolation: isolate;
		box-shadow: 0 8px 30px rgba(16, 16, 8, 0.18);
		background: #fff;
	}

	/* Watermark preview (#0298): a transparent-checker artboard face so the mark's
	   transparency reads against the picked backdrop (the checker shows wherever the
	   colour/image backdrop and the mark are both absent). Replaces the solid white. */
	.stage__artboard--backdrop {
		background-color: #fff;
		background-image:
			linear-gradient(45deg, #e8e8e8 25%, transparent 25%),
			linear-gradient(-45deg, #e8e8e8 25%, transparent 25%),
			linear-gradient(45deg, transparent 75%, #e8e8e8 75%),
			linear-gradient(-45deg, transparent 75%, #e8e8e8 75%);
		background-size: 20px 20px;
		background-position:
			0 0,
			0 10px,
			10px -10px,
			-10px 0;
	}

	/* PREVIEW-ONLY backdrop face — fills the artboard BEHIND the in-flow canvas. A NEGATIVE
	   z-index (inside the artboard's isolated stacking context) sits above the artboard
	   background but below the static canvas AND the positioned selection/handle overlays,
	   so the transparent mark previews over it without disturbing the editor chrome. */
	.stage__backdrop {
		position: absolute;
		inset: 0;
		z-index: -1;
		pointer-events: none;
	}

	/* Drag-to-bind: a primary ring around the artboard while hovering a replace
	   target, so the relink/rebind target reads clearly during the drag. */
	.stage__artboard--dropping {
		outline: 2px solid var(--cv-color-primary, #333333);
		outline-offset: 2px;
	}

	/* The layer that will be relinked/rebound on drop — a dashed primary overlay
	   (distinct from the solid selection box). */
	.stage__drop {
		position: absolute;
		pointer-events: none;
		border: 2px dashed var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	/* "Replace" badge on the target box — top-left, solid primary so the repoint is
	   unmistakable (vs. an empty-artboard place, which shows no box at all). */
	.stage__drop-label {
		position: absolute;
		top: 0;
		left: 0;
		padding: 0.125rem 0.375rem;
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.02em;
		line-height: 1.3;
		color: #fff;
		background: var(--cv-color-primary, #333333);
		border-bottom-right-radius: var(--cv-radius-sm, 0.25rem);
		white-space: nowrap;
	}

	/* Placement-tool draw preview — the box being dragged out before it's created. */
	.stage__tooldraw {
		position: absolute;
		pointer-events: none;
		border: 1px dashed var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	.stage__canvas {
		display: block;
		/* Select tool: default arrow; a grab cursor while actively moving. */
		cursor: default;
		touch-action: none;
	}

	.stage__canvas--dragging {
		cursor: grabbing;
	}

	/* Placement tool armed (Map panel): crosshair — the next click drops the preset. */
	.stage__canvas--placing {
		cursor: crosshair;
	}

	/* Selection overlay group — rotated as one about the layer's box centre so the
	   box, handles, scissors and rotation grip track rotated content. Transparent
	   to pointer events; the interactive chips below re-enable themselves. */
	.stage__overlay {
		position: absolute;
		inset: 0;
		pointer-events: none;
	}

	.stage__selection {
		position: absolute;
		pointer-events: none;
		border: 1.5px solid var(--cv-color-primary, #333333);
		box-shadow: 0 0 0 1px var(--cv-color-surface, #fff);
	}

	/* Resize handles — small squares centred on the selection's corners/edges. */
	.stage__handle {
		position: absolute;
		box-sizing: border-box;
		background: var(--cv-color-surface, #fff);
		border: 1.5px solid var(--cv-color-primary, #333333);
		border-radius: 2px;
		pointer-events: auto;
		touch-action: none;
	}

	/* Crop scissors — round chips beside the edge pins; the glyph turns to lie
	   along the cut for the side edges. */
	.stage__scissor {
		position: absolute;
		display: flex;
		align-items: center;
		justify-content: center;
		box-sizing: border-box;
		background: var(--cv-color-surface, #fff);
		border: 1.5px solid var(--cv-color-primary, #333333);
		border-radius: 50%;
		color: var(--cv-color-primary, #333333);
		pointer-events: auto;
		touch-action: none;
	}

	.stage__scissor:hover {
		background: var(--cv-color-primary, #333333);
		color: var(--cv-color-surface, #fff);
	}

	.stage__scissor--side :global(svg) {
		transform: rotate(90deg);
	}

	/* Rotation grip — round chip beside the top-centre pin (relocated #6). A drop
	   shadow + explicit stacking keep it legible over busy image content (the chip
	   sits ON the layer's top edge, often atop a photo) — #6 contrast polish. */
	.stage__rotate {
		position: absolute;
		z-index: 2;
		display: flex;
		align-items: center;
		justify-content: center;
		box-sizing: border-box;
		background: var(--cv-color-surface, #fff);
		border: 1.5px solid var(--cv-color-primary, #333333);
		border-radius: 50%;
		color: var(--cv-color-primary, #333333);
		box-shadow: 0 1px 4px rgba(16, 16, 8, 0.35);
		cursor: grab;
		pointer-events: auto;
		touch-action: none;
	}

	.stage__rotate:hover {
		background: var(--cv-color-primary, #333333);
		color: var(--cv-color-surface, #fff);
	}

	/* Crop-drag cursor forced globally (ScrubUnit pattern) — `cursor` doesn't
	   cascade past elements that set their own. */
	:global(body.cv-crop-ns),
	:global(body.cv-crop-ns *) {
		cursor: ns-resize !important;
		user-select: none !important;
	}

	:global(body.cv-crop-ew),
	:global(body.cv-crop-ew *) {
		cursor: ew-resize !important;
		user-select: none !important;
	}

	:global(body.cv-rotating),
	:global(body.cv-rotating *) {
		cursor: grabbing !important;
		user-select: none !important;
	}

	.stage__badge {
		position: absolute;
		bottom: var(--cv-space-md, 1rem);
		padding: 0.25rem 0.5rem;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--cv-color-neutral-600, #505050);
	}

	.stage__badge--zoom {
		left: var(--cv-space-md, 1rem);
	}

	.stage__badge--dims {
		right: var(--cv-space-md, 1rem);
	}
</style>
