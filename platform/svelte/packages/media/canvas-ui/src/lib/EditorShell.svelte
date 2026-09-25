<script lang="ts">
	// EditorShell — the full-bleed editor (D16). Layout (IA refactor R2):
	//   .editor (column)
	//     <TopToolbar>                       ← full-width: Back/brand/title · Export + save pill
	//     .editor__main (row)
	//       .editor__left (column)
	//         .editor__left-body (row)       ← <LeftRail> + the contextual panel for railContext
	//       .editor__canvas-area             ← <CanvasStage> (it owns its zoom/dims badges + fit)
	//       <Inspector>
	//     <Timeline>                         (bottom, collapsible; Animate on it)
	// Persistence is AUTOMATIC (R2): a debounced effect saves on every document mutation
	// (CanvasEditor.revision) — there is NO Save button — and Cmd/Ctrl-S forces it.
	// Preview + the Animated toggle are NOT shipped here (they need the R3 animation flag
	// — no dead controls). Owns reactive selection + document state (via CanvasEditor) and
	// the shared railContext + timeline-expanded state.
	import { onMount, onDestroy, untrack } from 'svelte';
	// beforeNavigate is client-only ($app/navigation) — it never fires during SSR, so
	// the import is safe at the top level and the hook body runs only in the browser.
	import { beforeNavigate } from '$app/navigation';
	// Modal isn't re-exported from the primitives barrel yet; deep-import it via the
	// core-ui `./*` wildcard export (component-first reuse — no re-implementation).
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import type { Snippet } from 'svelte';
	import { Button, ensureGoogleFont, markFontLoaded } from '@sbx/core-ui/components/primitives';
	import { type Document, type SourceProvider, type SourceRecord, type SourceRef, type FormatterRegistry, type AnimatableProperty, type ExportTarget, type BrandKit, type LayerDefaults, type ResolvedImage, type ImageRenderRole, type MapStaticUrlBuilder, type EditorMode, EMPTY_BRAND_KIT, NEUTRAL_LAYER_DEFAULTS, exportPage, exportMarkUnit, presetToLayer, pickImageVariant, resolveEditorFeatures } from '@sbx/canvas-kit';
	import { PRESETS, placementOf } from './palette.js';
	import type { TemplateGallery } from './template-gallery.js';
	import { CanvasEditor } from './editor-state.svelte.js';
	import { createRasterizer } from './rasterize.js';
	import { downloadBlob, exportFilename } from './download.js';
	import TopToolbar from './TopToolbar.svelte';
	import LeftRail, { type RailContext } from './LeftRail.svelte';
	import DocumentPanel from './DocumentPanel.svelte';
	import SourcesPanel from './SourcesPanel.svelte';
	import LayersPanel from './LayersPanel.svelte';
	import MediaPanel from './MediaPanel.svelte';
	import TextsPanel from './TextsPanel.svelte';
	import MapPanel from './MapPanel.svelte';
	import ComponentsPanel from './ComponentsPanel.svelte';
	import ResizeDialog from './ResizeDialog.svelte';
	import ExportDialog from './ExportDialog.svelte';
	import AddAnimationDialog from './AddAnimationDialog.svelte';
	import AnimationInfoPopover from './AnimationInfoPopover.svelte';
	import CanvasStage from './CanvasStage.svelte';
	import Inspector from './Inspector.svelte';
	import { SourceBrowser } from '@sbx/ui-source-picker';
	import Timeline from './Timeline.svelte';
	import { sourceTitle } from './source-display.js';

	interface Props {
		doc: Document;
		sourceProviders?: SourceProvider[];
		formatters?: FormatterRegistry;
		readonly?: boolean;
		/** Persist the current document (slice 3). The consumer (the BR route) owns
		 *  the API call; the kit/ui stay generic. Returns ok/error so the shell can
		 *  surface Saving…/Saved/Retry on the action cluster. */
		onsave?: (doc: Document) => Promise<{ ok: boolean; error?: string; authExpired?: boolean }>;
		/** Maps a layer image src to the URL actually loaded (CanvasStage + export
		 *  rasterizer). The consumer injects a same-origin proxy rewriter so the
		 *  export canvas is CORS-clean (S-EXPORT). Default = identity. */
		resolveImageSrc?: (src: string) => string;
		/** Consumer brand entries (Decision #0286 Phase A) — the Inspector lists them
		 *  as color swatches + the FONT select. Default = no brand entries. */
		brandKit?: BrandKit;
		/** Brand-derived colour/font defaults NEW layers are born with (presetToLayer).
		 *  Default = the kit's brand-neutral values. */
		layerDefaults?: LayerDefaults;
		/** BUILT-IN sources (#0286 refinement): always-on refs passed by code under
		 *  reserved aliases (e.g. Branding). Injected into resolution + panels, never
		 *  persisted, never attachable/detachable — their kinds are excluded from the
		 *  attach picker (their provider must still be in `sourceProviders` for
		 *  fields/imageCategories/re-resolve). */
		builtinSources?: SourceRef[];
		/** Optional subset of `sourceProviders` offered in the attach picker. When
		 *  omitted, all `sourceProviders` minus built-in kinds are offered. Pass a
		 *  feature-gated subset to keep in-progress providers resolvable (for documents
		 *  that already have them attached) without surfacing them in the picker. */
		attachableProviders?: SourceProvider[];
		/** Real templates for the Templates panel (#4) — summaries + lazy doc loader +
		 *  apply. Omit → the panel shows its empty state. */
		templates?: TemplateGallery;
		/** Load the Google Fonts catalogue for the Inspector FontPicker (#3) — the
		 *  consumer owns the server-side proxy fetch. Called once on mount; [] on failure
		 *  degrades the picker to system/brand fonts. */
		loadFontCatalogue?: () => Promise<{ family: string; category?: string }[]>;
		/** Read the current marketing alias for an enum value — forwarded to Inspector. */
		getEnumAlias?: (set: string, code: string) => string;
		/** Persist a new enum alias — forwarded to Inspector. */
		onaliaschange?: (set: string, code: string, alias: string) => void;
		/** Build a keyless /api/map/static URL for a map layer (T4→T5 contract). Injected by
		 *  the consumer (BR injects buildStaticMapPath from @sbx/ui-map). Without it, placed
		 *  map layers render blank on the artboard and in exports. */
		buildMapUrl?: MapStaticUrlBuilder;
		/** Editor experience preset (Slice A · Decision #0298). 'design' (default) is the
		 *  full Media Canvas — exactly as before (back-compat). 'watermark' presets a
		 *  static overlay-authoring surface: no animation/live-sources/components/map,
		 *  gains Publish + Copy + a Watermark inspector block + a preview backdrop. The
		 *  gating is UI-only — server-side validation owns persistence (Slice B). */
		mode?: EditorMode;
		/** Product-level rail hiding (task 2607-033) — rail contexts the CONSUMER suppresses on
		 *  top of the mode filter, without changing the mode's feature bundle. BR's Media Canvas
		 *  passes ['map','components'] to hide Map + Shapes while staying in full design mode.
		 *  Forwarded to LeftRail; if the active context becomes hidden it falls back to 'document'. */
		hiddenRailItems?: RailContext[];
		/** Publish the current design into the watermark registry (the Publish action,
		 *  watermark mode). The consumer owns the render→upload→registry call. Shown only
		 *  when `mode` enables publish AND this is provided. Receives BOTH the full-frame
		 *  overlay PNG and the cropped MARK UNIT PNG (#0299 · Slice C0) — the mark unit is the
		 *  repeat tile the headless apply worker re-tiles at each photo's pixels; null when the
		 *  mark is empty (a blank overlay). */
		onpublish?: (overlay: Blob, markUnit: Blob | null) => void;
		/** Whether the current document is already published/Active (watermark mode). Flips
		 *  the toolbar's Publish CTA to Unpublish; set by the consumer from its registry. */
		published?: boolean;
		/** Unpublish the current document (watermark mode) — the consumer owns the API call +
		 *  the refresh. Surfaced (as the flipped Publish CTA) in the toolbar only when published. */
		onunpublish?: () => void;
		/** Optional consumer-owned publishability rule, evaluated against the LIVE artboard size
		 *  (so it re-checks on resize). When it returns false the Publish CTA is disabled and a
		 *  note appears next to the artboard size. The shell stays domain-neutral — the consumer
		 *  owns the rule (BR: only a square watermark is publishable). Omit → always publishable. */
		isPublishable?: (width: number, height: number) => boolean;
		/** The reason shown when `isPublishable` returns false — the disabled Publish CTA's hover
		 *  title AND the note next to the artboard size (e.g. "Only square watermarks can be published."). */
		publishHint?: string;
		/** Where the toolbar Back link navigates (the full-bleed editor escape). Defaults
		 *  to the Media Canvas hub; the watermark route passes its own list (#0298). */
		backHref?: string;
		/** The Back link text + the wordmark eyebrow (the surface name). Default to the
		 *  Media Canvas; the watermark route passes "Back to Watermarks" / "WATERMARK". */
		backLabel?: string;
		/** Whether the toolbar Back link shows (default true). The watermark surface hides
		 *  it and routes home via the clickable wordmark instead. */
		showBack?: boolean;
		/** When set, the toolbar wordmark links here (the admin dashboard) — the watermark
		 *  surface's way home in place of a Back link. */
		homeHref?: string;
		eyebrow?: string;
		/** Optional consumer-owned uploader rendered at the top of the Media panel's Images
		 *  view (BR: an "Upload watermark" ImagePicker → the watermark library). The kit only
		 *  hosts the slot; the consumer owns the upload + the source it lands in. */
		mediaUploadSlot?: Snippet;
		/** Optional consumer-owned Documents LIST — sibling documents to switch between (the
		 *  watermark route lists its watermark docs with status + switch + delete). Rendered
		 *  INSIDE the Document panel, BELOW the current-doc properties (one merged panel — the
		 *  former separate 'documents' rail tab is gone). GENERIC by contract: the shell only
		 *  hosts the slot; it knows NOTHING of watermarks/status/registry (the consumer owns
		 *  every semantic). Omit → properties-only Document panel (Media Canvas, back-compat). */
		documentsPanel?: Snippet;
		/** Optional consumer-owned header action for the Document panel — a ＋ button to the
		 *  right of the panel title (the watermark route's "create new" → seed + open). Forwarded
		 *  to Panel.headerAction. Only meaningful alongside `documentsPanel`. */
		documentsHeaderAction?: Snippet;
		/** Optional consumer-owned action rendered at the FOOT of the Document-settings section
		 *  (below the artboard size). The watermark route puts a subtle "Delete watermark" link
		 *  here so the CURRENT document is deletable from its own settings (the sibling list only
		 *  deletes other docs). GENERIC by contract — the shell hosts the slot; the consumer owns
		 *  the action + its confirm + any redirect. Omit → no footer action (Media Canvas). */
		documentSettingsAction?: Snippet;
		/** Fired SYNCHRONOUSLY when the user commits a document rename (blur / Enter in the title
		 *  field) — BEFORE the debounced auto-save. Lets the consumer update a local title override
		 *  immediately so sibling-document lists stay in sync without waiting for the persist cycle. */
		ondocumentrename?: (name: string) => void;
	}

	let {
		doc,
		sourceProviders = [],
		formatters = {},
		readonly = false,
		onsave,
		resolveImageSrc = (s) => s,
		brandKit = EMPTY_BRAND_KIT,
		layerDefaults = NEUTRAL_LAYER_DEFAULTS,
		builtinSources = [],
		attachableProviders,
		templates,
		loadFontCatalogue,
		getEnumAlias,
		onaliaschange,
		buildMapUrl,
		mode = 'design',
		hiddenRailItems,
		onpublish,
		published = false,
		onunpublish,
		isPublishable,
		publishHint,
		backHref,
		backLabel,
		showBack = true,
		homeHref,
		eyebrow,
		mediaUploadSlot,
		documentsPanel,
		documentsHeaderAction,
		documentSettingsAction,
		ondocumentrename
	}: Props = $props();

	// Resolve the mode to its feature bundle (Decision #0298). Every capability gate
	// below reads `features.*` — 'design' resolves to the full set, so the editor
	// behaves EXACTLY as before when `mode` is omitted (back-compat). (`animationEnabled`
	// is derived below, once `editor` exists — it reads editor.animated.)
	const features = $derived(resolveEditorFeatures(mode));

	// Watermark sizing ratios (Slice S1) — the iPhone-Photos-style ratio chips the
	// ResizeDialog offers in photo-sizing mode. Watermark documents are full overlays
	// sized to the image max (2048, the watermark facet cap — hardcoded here so canvas-ui
	// stays free of a ui-image dependency). Only used when features.watermarkPhotoSizing.
	const WATERMARK_RATIOS = [
		{ id: '16x9', label: '16:9', w: 16, h: 9 },
		{ id: '4x5', label: '4:5', w: 4, h: 5 },
		{ id: '3x2', label: '3:2', w: 3, h: 2 },
		{ id: '4x3', label: '4:3', w: 4, h: 3 },
		{ id: '1x1', label: '1:1', w: 1, h: 1 }
	];
	const WATERMARK_MAX_DIMENSION = 2048;

	// Google Fonts catalogue (#3) — loaded once on mount via the consumer's proxy; the
	// Inspector FontPicker filters/paginates it. Empty until loaded / on failure.
	let fontCatalogue = $state<{ family: string; category?: string }[]>([]);

	// The editor owns the document's mutable state from here on; seed it ONCE from
	// the prop (untrack makes the one-time hand-off explicit — re-seeding on every
	// doc change is a later concern when documents load by id, slice 3). Providers +
	// formatters seed the binding hub (the attach picker + Inspector 🏷 read them).
	const editor = new CanvasEditor(untrack(() => doc), {
		providers: untrack(() => sourceProviders),
		formatters: untrack(() => formatters),
		layerDefaults: untrack(() => layerDefaults),
		builtinSources: untrack(() => builtinSources)
	});

	// Animation is gated by the MODE, not only the doc: in watermark mode it's forced
	// off regardless of the document's `animated` flag (a watermark is static — the
	// Timeline, the Animated toggle, and the Inspector animate chips all disappear). In
	// design mode this is exactly `editor.animated` (unchanged behaviour). The doc flag
	// is never mutated — gating is UI-only.
	const animationEnabled = $derived(features.animation && editor.animated);

	// Keep the editor's layerDefaults in sync with the prop: the consumer (BR route)
	// builds brand defaults CLIENT-side after mount (reading live CSS tokens needs the
	// DOM), so the prop legitimately changes after construction. The editor reads it
	// imperatively (placeImageLayer) — a plain re-assignment is all sync needed.
	$effect(() => {
		editor.layerDefaults = layerDefaults;
	});

	// Font loading (#3): brand fonts are already in the page → mark them so ensureGoogleFont
	// no-ops; every DISTINCT layer font is ensured (injects the css2 <link>) so an opened
	// document paints in its real face, not a fallback. CanvasStage re-draws on font load.
	const layerFonts = $derived.by(() => {
		const fams = new Set<string>();
		for (const p of editor.pages) for (const l of p.layers) if (l.font) fams.add(l.font);
		return [...fams];
	});
	$effect(() => {
		for (const f of brandKit.fonts) markFontLoaded(f.family);
		for (const fam of layerFonts) void ensureGoogleFont(fam);
	});

	// Same client-side hand-off for the built-in sources (#0286): the consumer builds
	// them after mount (Branding reads live CSS/config). $state on the editor — panels
	// + canvas resolution re-render from the merged `sources` getter.
	$effect(() => {
		editor.builtinSources = builtinSources;
	});

	/** Attach-picker providers: built-in kinds are NOT attachable (always-on by
	 *  injection) — offering them would create a second, detachable doc-level copy.
	 *  When `attachableProviders` is provided explicitly, use it (allows the consumer
	 *  to feature-gate the picker while keeping all providers available for re-resolve). */
	const pickerProviders = $derived.by(() => {
		if (attachableProviders) return attachableProviders;
		const builtinKinds = new Set(builtinSources.map((s) => s.kind));
		return sourceProviders.filter((p) => !builtinKinds.has(p.kind));
	});

	// Role-based image variants (task 2606-001). A layer's bound image src IS the
	// ResolvedImage `url`; on screen we want the lighter `medium`, on export the full
	// `original`. Index every attached source's resolved images by their `url` (the
	// bind-target identity), then map a layer src → the role variant BEFORE the
	// consumer's same-origin proxy. A src with no indexed variant (maps, branding, a
	// legacy `coverImage.url` token, or an old document) passes through unchanged →
	// behaviour is exactly as before (additive, back-compat).
	const imageVariantsByUrl = $derived.by(() => {
		const byUrl = new Map<string, ResolvedImage>();
		for (const ref of editor.sources) {
			const images = ref.snapshot?.images;
			if (!images || typeof images !== 'object') continue;
			for (const group of Object.values(images as Record<string, unknown>)) {
				if (!Array.isArray(group)) continue;
				for (const img of group as ResolvedImage[]) {
					if (img && typeof img.url === 'string') byUrl.set(img.url, img);
				}
			}
		}
		return byUrl;
	});

	/** Resolve a layer image src for a render role, then route through the consumer's
	 *  same-origin proxy (CORS-clean). Unknown src → role-agnostic passthrough. */
	function resolveImageForRole(src: string, role: ImageRenderRole): string {
		const img = imageVariantsByUrl.get(src);
		return resolveImageSrc(img ? pickImageVariant(img, role) : src);
	}
	// On-screen artboard = medium; export = full-resolution original.
	const screenImageSrc = (src: string): string => resolveImageForRole(src, 'medium');
	const exportImageSrc = (src: string): string => resolveImageForRole(src, 'original');

	let railContext = $state<RailContext>('document'); // default = Document (IA refactor R1)
	// If a consumer hides the currently-active rail context (task 2607-033 hiddenRailItems),
	// fall back to Document so the panel column never renders an orphaned/hidden context.
	$effect(() => {
		if (hiddenRailItems?.includes(railContext)) railContext = 'document';
	});
	let timelineExpanded = $state(false); // collapsed by default (D10)
	// Attach-source modal (now multi-kind via <SourceBrowser>): `attachOpen` drives the Modal,
	// `attachInitialKind` opens straight on a kind (Settings pre-selection) or null = kind chooser.
	let attachOpen = $state(false);
	let attachInitialKind = $state<string | null>(null);
	// Surfaced attach failure (a record's resolve threw) — shown in the modal so the failure is
	// never silent; any records attached before the failure are kept (the message says so).
	let attachError = $state<string | null>(null);
	let resizeOpen = $state(false); // the functional Resize dialog (Settings → Document settings)
	let exportOpen = $state(false); // the Export dialog (R4) — replaces the direct Export action

	// Change-source modal (S-SOURCE-CHANGE): re-pick the record for an EXISTING alias. A
	// single-select picker for THAT source's kind, pre-highlighting the current record;
	// confirming RE-ATTACHES under the SAME alias, so every binding on that source follows
	// automatically (bindings reference the alias + token, never the record id).
	let changeOpen = $state(false);
	let changeAlias = $state<string | null>(null);
	let changeKind = $state<string | null>(null);
	let changeRefId = $state<string | null>(null);
	let changing = $state(false);
	let changeError = $state<string | null>(null);
	// One provider (the source's own kind) → SourceBrowser skips the kind chooser and opens
	// straight on that kind's picker (you replace a property with a property, etc.).
	const changeProviders = $derived(
		changeKind ? sourceProviders.filter((p) => p.kind === changeKind) : []
	);

	// Detach-confirm: detaching a source that has bound layers warns first (the bindings would
	// fall back to their placeholder). A source with zero bound layers detaches immediately.
	let detachOpen = $state(false);
	let detachAlias = $state<string | null>(null);
	let detachTitle = $state('');
	let detachCount = $state(0);

	// Background-apply confirm (user directive 2026-06-06). Policy: replacing a
	// NON-DEFAULT background (image, map, or a non-white color — an explicitly
	// picked white reads as default by design) always asks; a MAP apply asks even
	// over the pristine default; only image-over-default applies silently.
	let bgConfirm = $state<null | { mode: 'image' | 'map'; token: string; alias: string; label?: string }>(null);

	function requestBackground(mode: 'image' | 'map', token: string, alias: string, label?: string): void {
		if (mode === 'map' || !editor.backgroundIsDefault()) {
			bgConfirm = { mode, token, alias, label };
			return;
		}
		applyBackground(mode, token, alias);
	}

	function applyBackground(mode: 'image' | 'map', token: string, alias: string): void {
		editor.setBackgroundImage(mode, token, alias);
		// Land selected on the background so the Inspector shows what just changed.
		editor.selectBackground();
	}

	// Save lifecycle (slice 3). The consumer's onsave does the API call; the shell
	// only reflects its result on the action cluster. 'saved' is transient (reverts
	// to idle) so it reads as a confirmation, not a permanent state.
	let saveState = $state<'idle' | 'saving' | 'saved' | 'error'>('idle');
	let savedTimer: ReturnType<typeof setTimeout> | null = null;

	// Auto-save (R2). `lastSavedRevision` is the CanvasEditor.revision persisted by the
	// last successful save (or the freshly-loaded document, which starts at 0 — clean).
	// `dirty` = the document changed since; a debounced effect saves SAVE_DEBOUNCE_MS
	// after the last edit. `saveStatus` is what the toolbar pill renders.
	let lastSavedRevision = $state(0);
	// The most recent revision an auto-save ATTEMPTED (success or failure) — the
	// no-tight-loop latch: a failed save must NOT re-fire for the same revision, only
	// for a NEW edit (or the explicit Retry pill). Plain latch — never a reactive dep.
	let lastAttemptedRevision = -1;
	// Terminal: the session lapsed (a 401 hard-redirected to /login). Once set, the
	// auto-saver stops for good so it can't loop a dead write while the navigation
	// to /login is still in flight (#5 — window.location.href is async).
	let authExpired = $state(false);
	let saveDebounce: ReturnType<typeof setTimeout> | null = null;
	const SAVE_DEBOUNCE_MS = 800;
	const dirty = $derived(editor.revision !== lastSavedRevision);
	// 'idle' (resting + clean) renders NOTHING in the toolbar — no permanent "Saved" pill
	// (user directive 2026-06-26). 'saved' is the TRANSIENT post-save confirmation only:
	// saveState goes 'saved' → (1.8s) → 'idle', so the pill flashes "Saved" once then hides.
	const saveStatus = $derived<'idle' | 'saved' | 'pending' | 'saving' | 'error'>(
		saveState === 'saving'
			? 'saving'
			: saveState === 'error'
				? 'error'
				: dirty
					? 'pending'
					: saveState === 'saved'
						? 'saved'
						: 'idle'
	);
	// True while resolving a just-picked record's snapshot (attach modal busy state).
	let attaching = $state(false);

	/** Strip the in-memory source snapshots before persisting (D20: the document keeps
	 *  source refs + per-binding overrides only — never a frozen snapshot; it re-resolves
	 *  live on open). Overrides live on `layer.bindings[].override` and are kept. */
	function toPersist(document: Document): Document {
		if (!document.sources?.length) return document;
		return { ...document, sources: document.sources.map(({ snapshot: _snapshot, ...ref }) => ref) };
	}

	async function handleSave(): Promise<boolean> {
		if (readonly || !onsave || saveState === 'saving') return true;
		// A forced/triggered save pre-empts any queued debounce.
		if (saveDebounce) {
			clearTimeout(saveDebounce);
			saveDebounce = null;
		}
		if (savedTimer) {
			clearTimeout(savedTimer);
			savedTimer = null;
		}
		// Snapshot the revision being persisted: edits made DURING the await advance
		// editor.revision past this, so the auto-save effect re-fires for the tail.
		const rev = editor.revision;
		saveState = 'saving';
		try {
			const res = await onsave(toPersist(editor.document));
			lastAttemptedRevision = rev;
			if (res.ok) {
				lastSavedRevision = rev;
				saveState = 'saved';
			} else {
				saveState = 'error';
				// Session lapsed → the consumer redirected to /login; stop the auto-saver
				// for good so it never loops the dead write behind the navigation (#5).
				if (res.authExpired) authExpired = true;
			}
		} catch {
			lastAttemptedRevision = rev;
			saveState = 'error';
		}
		if (saveState === 'saved') {
			savedTimer = setTimeout(() => (saveState = 'idle'), 1800);
		}
		// Report success so a triggered flush (e.g. pre-publish) can abort on failure.
		return saveState !== 'error';
	}

	// Debounced auto-save: re-runs whenever the document mutates (editor.revision) or a
	// save resolves (lastSavedRevision / saveState). Schedules a single trailing save and
	// resets the timer on each edit; a save in flight defers (the revision snapshot in
	// handleSave makes this re-fire for edits made during the await). No effect-cleanup
	// of the timer — that would drop a pending save when the effect re-runs for a
	// non-edit reason; onDestroy clears it instead.
	$effect(() => {
		const rev = editor.revision; // track persisted mutations
		if (readonly || !onsave) return;
		if (authExpired) return; // session died — stop auto-saving (the redirect to /login is in flight, #5)
		if (rev === lastSavedRevision) return; // nothing new since the last save
		if (saveState === 'saving') return; // in flight — re-fires on completion
		// Already FAILED this exact revision → don't tight-loop the same dead write; wait
		// for a NEW edit (rev advances) or the explicit Retry pill (onforcesave). (#5)
		if (saveState === 'error' && rev === lastAttemptedRevision) return;
		if (saveDebounce) clearTimeout(saveDebounce);
		saveDebounce = setTimeout(() => {
			saveDebounce = null;
			void handleSave();
		}, SAVE_DEBOUNCE_MS);
	});

	// Export (S-EXPORT). Build a target from the active page + the chosen format,
	// resolve the frame at the playhead, rasterize via the canvas-ui renderer, and
	// download. Re-entrancy-guarded; a stalled document.fonts.ready cannot hang the
	// export (raced against a timeout).
	let exporting = $state(false);

	function fontsReadyOrTimeout(ms: number): Promise<void> | undefined {
		if (typeof document === 'undefined' || !document.fonts) return undefined;
		return Promise.race([
			document.fonts.ready.then(() => undefined),
			new Promise<void>((resolve) => setTimeout(resolve, ms))
		]);
	}

	// Export (R4) — the ExportDialog picks the format + page scope; the shell rasterizes.
	// 'current' exports the active page; 'all' loops every page (multipage docs) into one
	// file each. Re-entrancy-guarded; a stalled document.fonts.ready can't hang it (timeout).
	async function handleExport(scope: 'current' | 'all' = 'current'): Promise<void> {
		if (readonly || exporting) return;
		const pages = scope === 'all' ? editor.pages : editor.activePage ? [editor.activePage] : [];
		if (!pages.length) return;
		exporting = true;
		const format = editor.exportFormat;
		try {
			for (let i = 0; i < pages.length; i++) {
				const page = pages[i];
				const target: ExportTarget = {
					format,
					width: page.width,
					height: page.height,
					retina: true,
					quality: format === 'jpeg' ? 92 : undefined
				};
				const blob = await exportPage(page, target, {
					t: editor.time,
					animate: animationEnabled,
					sources: editor.sources, // merged: doc-attached + built-ins (#0286) — exports resolve branding bindings
					formatters,
					fontsReady: fontsReadyOrTimeout(3000),
					rasterize: createRasterizer(exportImageSrc),
					buildMapUrl // T4→T5: bake map layer src before the rasterizer draws
				});
				// Multipage → suffix the page number so the downloads don't collide.
				const name = pages.length > 1 ? `${editor.document.name} ${i + 1}` : editor.document.name;
				downloadBlob(blob, exportFilename(name, format));
			}
		} catch (err) {
			// No silent success: surface the failure (a toast surface is a follow-up).
			console.error('canvas export failed', err);
		} finally {
			exporting = false;
		}
	}

	// Publish (watermark mode · #0298 Slice B; mark unit · #0299 Slice C0). Render the ACTIVE
	// page to a TRANSPARENT overlay PNG by REUSING the SAME export call ExportDialog drives
	// (handleExport): exportPage(activePage, {format:'png',…}, {animate:false, sources, rasterize}).
	// Then render a SECOND PNG — the cropped MARK UNIT (exportMarkUnit) — the repeat tile the
	// headless Go apply worker re-tiles at each photo's exact pixels (it can't run canvas-kit, so
	// the tile MUST be rendered here with full client fidelity). The preview backdrop is stage
	// chrome (never a layer) so it is already excluded — both PNGs carry only the mark. We FLUSH
	// any pending edit first (handleSave self-guards readonly / no-onsave / in-flight) so a
	// just-edited design is persisted before it ships; the registry call then references the same
	// document the editor saved. The shell stays REGISTRY-AGNOSTIC: it only hands the Blobs to the
	// consumer's onpublish — it never knows the watermark endpoints or imports ui-image.
	// Re-entrancy-guarded (publishing).
	let publishing = $state(false);

	async function handlePublish(): Promise<void> {
		if (readonly || publishing) return;
		// Defence-in-depth: the toolbar already disables Publish when not publishable, but never
		// ship an overlay the consumer's rule rejects (e.g. a non-square watermark).
		if (!publishable) return;
		const page = editor.activePage;
		if (!page) return;
		publishing = true;
		try {
			// Flush a just-edited design so the published overlay and the saved doc agree.
			// If the flush FAILS (network / session lapse), ABORT: the server re-loads the
			// PERSISTED doc on publish, so shipping the live-rendered overlay now would
			// diverge it from the stored snapshot. The save-error pill surfaces the failure.
			if (!(await handleSave())) return;
			const target: ExportTarget = {
				format: 'png',
				width: page.width,
				height: page.height,
				retina: true
			};
			const overlayBlob = await exportPage(page, target, {
				t: editor.time,
				animate: false, // a watermark is static — never bake animation/the t=0 frame
				sources: editor.sources, // merged: doc-attached + built-ins (#0286)
				formatters,
				fontsReady: fontsReadyOrTimeout(3000),
				rasterize: createRasterizer(exportImageSrc),
				buildMapUrl
			});
			// The MARK UNIT (#0299 · Slice C0) — the cropped repeat tile the headless apply
			// worker re-tiles at each photo's pixels. Same resolve seams as the overlay; null
			// when the mark is empty (a blank overlay → no tile to apply).
			const markBlob = await exportMarkUnit(page, {
				t: editor.time,
				animate: false,
				sources: editor.sources,
				formatters,
				fontsReady: fontsReadyOrTimeout(3000),
				rasterize: createRasterizer(exportImageSrc),
				buildMapUrl,
				retina: true
			});
			onpublish?.(overlayBlob, markBlob);
		} catch (err) {
			// No silent success: the consumer surfaces upload failures; a render failure logs here.
			console.error('watermark publish render failed', err);
		} finally {
			publishing = false;
		}
	}

	/** Selection confirmed in the attach browser — resolve each chosen record's live snapshot
	 *  and attach it under the next free alias (additive multi-source: first → 'primary', then
	 *  'secondary', …). Best-effort; the picker surfaces its own load errors, and a failed
	 *  resolve leaves the modal open to retry. */
	async function handleAttachConfirm(kind: string, records: SourceRecord[]): Promise<void> {
		const provider = sourceProviders.find((p) => p.kind === kind);
		if (!provider || attaching) return;
		attaching = true;
		attachError = null;
		try {
			for (const record of records) {
				const snapshot = await provider.resolve(record.id);
				editor.attachAs(editor.nextAlias(), kind, record.id, snapshot);
			}
			attachOpen = false;
			attachInitialKind = null;
		} catch {
			// Resolve is shell-side (the picker only surfaces list errors), so surface it here —
			// never silent. Records attached before the failure are kept; say so. Modal stays open.
			attachError = 'Couldn’t attach one or more sources. Any already added are kept — close, then try the rest.';
		} finally {
			attaching = false;
		}
	}

	/** Open the change-source modal for an existing alias — seeds the current kind + record so
	 *  the picker opens on that kind and pre-highlights the current selection. */
	function openChange(alias: string): void {
		const ref = editor.sources.find((s) => s.alias === alias);
		if (!ref) return;
		changeAlias = alias;
		changeKind = ref.kind;
		changeRefId = ref.refId;
		changeError = null;
		changeOpen = true;
	}

	function closeChange(): void {
		changeOpen = false;
		changeAlias = null;
		changeKind = null;
		changeRefId = null;
		changeError = null;
	}

	/** A replacement record was confirmed — resolve its live snapshot and RE-ATTACH under the
	 *  SAME alias (every binding on that source follows automatically). Picking the current
	 *  record is a no-op close. A failed resolve keeps the modal open with a message. */
	async function handleChangeConfirm(kind: string, record: SourceRecord): Promise<void> {
		if (!changeAlias || changing) return;
		if (record.id === changeRefId) {
			closeChange(); // unchanged selection → nothing to replace.
			return;
		}
		const provider = sourceProviders.find((p) => p.kind === kind);
		if (!provider) return;
		changing = true;
		changeError = null;
		try {
			const snapshot = await provider.resolve(record.id);
			editor.attachAs(changeAlias, kind, record.id, snapshot);
			closeChange();
		} catch {
			changeError = 'Couldn’t load that source — try another.';
		} finally {
			changing = false;
		}
	}

	/** Detach intent from a SourceCell ✕ — confirm first when layers are bound (they'd fall
	 *  back to a placeholder), else detach immediately. */
	function requestDetach(alias: string): void {
		const n = editor.boundLayerCount(alias);
		if (n === 0) {
			editor.detachSource(alias);
			return;
		}
		const ref = editor.sources.find((s) => s.alias === alias);
		detachAlias = alias;
		detachTitle = ref ? sourceTitle(ref) : alias;
		detachCount = n;
		detachOpen = true;
	}

	function confirmDetach(): void {
		if (detachAlias) editor.detachSource(detachAlias);
		closeDetach();
	}

	function closeDetach(): void {
		detachOpen = false;
		detachAlias = null;
		detachTitle = '';
		detachCount = 0;
	}

	/** Re-resolve every attached source live when the document opens (D20 freshness) —
	 *  fills the in-memory snapshots the canvas + Inspector render from. Best-effort:
	 *  a failed resolve leaves the binding on its `fallback`, never blocks the editor. */
	async function reresolveSources(): Promise<void> {
		for (const ref of editor.sources) {
			const provider = editor.providerFor(ref.kind);
			if (!provider) continue;
			try {
				const snapshot = await provider.resolve(ref.refId);
				editor.setSnapshot(ref.alias, snapshot);
			} catch {
				// keep whatever snapshot exists (or none → fallback)
			}
		}
	}

	// Add-animation dialog state. Opened from the Inspector ≣ for a specific property
	// at 0s (§12.4 "from anywhere else → 0s") on the selected layer, OR from the
	// timeline gray-track click for a specific layer at the clicked time (no
	// preselected property). `animLayerId` lets the timeline target a layer other
	// than the current selection.
	let animOpen = $state(false);
	let animProperty = $state<AnimatableProperty | undefined>(undefined);
	let animStartMs = $state(0);
	let animLayerId = $state<string | null>(null);

	function openAnimDialog(property: AnimatableProperty, startMs: number): void {
		if (readonly || !editor.selectedLayerId) return;
		animLayerId = editor.selectedLayerId;
		animProperty = property;
		animStartMs = startMs;
		animOpen = true;
	}

	// Timeline gray-track add (§12.4): select the clicked layer + open the dialog at
	// the clicked time with NO preselected property (the dropdown offers its
	// not-yet-animated props).
	function openAnimDialogForLayer(layerId: string, startMs: number): void {
		if (readonly) return;
		editor.select(layerId);
		animLayerId = layerId;
		animProperty = undefined;
		animStartMs = startMs;
		animOpen = true;
	}

	// Timeline double-click (§12.5): open the info/edit popover for a layer's property.
	let infoOpen = $state(false);
	let infoLayerId = $state<string | null>(null);
	let infoProperty = $state<AnimatableProperty | null>(null);

	function openAnimInfo(layerId: string, property: AnimatableProperty): void {
		if (readonly) return;
		infoLayerId = layerId;
		infoProperty = property;
		infoOpen = true;
	}

	// Artboard dimensions for the Settings info rows + Resize dialog (active page).
	const artboardWidth = $derived(editor.activePage?.width ?? 0);
	const artboardHeight = $derived(editor.activePage?.height ?? 0);

	// Publishability is the consumer's rule (e.g. BR: square only), evaluated against the LIVE
	// artboard size so a resize re-checks it. No rule → always publishable. Drives the toolbar
	// Publish-disable, the Document-settings size note, and the publish guard — ONE source.
	const publishable = $derived(isPublishable ? isPublishable(artboardWidth, artboardHeight) : true);

	// Whether the document already has user content (drives the Templates apply
	// confirm: empty → apply directly; non-empty → warn). System layers (e.g. the
	// background) don't count as user content.
	const documentHasContent = $derived(editor.layers.some((l) => !l.system));

	/**
	 * Materialise a preset as a new layer CENTRED on (cx, cy) artboard px — shared by
	 * click-insert (page centre) and drag-to-place (drop point, D-2). The factory
	 * returns null for base types that have no renderer yet (widget — the only
	 * non-renderable base left in the palette) — a safe no-op until its later slice.
	 * The new layer is auto-selected by the add reducer (MC-SC-01).
	 */
	function insertPresetAt(presetId: string, cx: number, cy: number, drawW?: number, drawH?: number): void {
		if (readonly) return;
		const preset = PRESETS.find((p) => p.id === presetId);
		if (!preset) return;
		// Materialise at origin to read the seed size; a DRAWN box (drawW/drawH from a
		// click-drag) overrides it, else the seed's default size. Then offset to centre on (cx, cy).
		const seed = presetToLayer(preset, 0, 0, layerDefaults);
		if (!seed) return; // TODO(widget render slice): renderable base types only this cut.
		const width = drawW ?? seed.width ?? 0;
		const height = drawH ?? seed.height ?? 0;
		const x = Math.round(cx - width / 2);
		const y = Math.round(cy - height / 2);
		let layer = presetToLayer(preset, x, y, layerDefaults);
		if (!layer) return;
		// Apply the drawn box size (Figma-style draw-to-create).
		if (drawW != null && drawH != null) layer = { ...layer, width: Math.round(drawW), height: Math.round(drawH) };
		// A placed map opens on the document's PROPERTY location (B7), not the neutral
		// factory fallback — falls back to the factory default when no source geo exists.
		if (layer.type === 'map') {
			const cfg = editor.mapDefaultConfig();
			if (cfg) layer = { ...layer, map_config: cfg };
		}
		// Anchor annotations (marker/outline/callout) to a map surface so they bake/project
		// (no-op for non-annotation presets). (cx, cy) is the drop point for drags and the
		// page centre for clicks — both already flow in.
		layer = editor.anchorIfAnnotation(layer, cx, cy);
		editor.add(layer);
	}

	/** The armed placement TOOL's preset id (or null). The grid card stays lit and the
	 *  canvas cursor is a crosshair while a tool is armed. */
	let activeTool = $state<string | null>(null);

	/** Palette card click — the standardized placement model. 'instant' presets drop in
	 *  the page centre immediately (Canva-style, content blocks). 'tool' presets ARM
	 *  (toggle) the cursor tool, so the next canvas click/drag creates them where you
	 *  point (Figma-style). Re-clicking the same tool card, Esc, or placing it disarms. */
	function handleInsert(presetId: string): void {
		if (placementOf(presetId).mode === 'instant') {
			const page = editor.activePage;
			if (!page) return;
			insertPresetAt(presetId, page.width / 2, page.height / 2);
			activeTool = null;
		} else {
			activeTool = activeTool === presetId ? null : presetId;
		}
	}

	// Delete / Backspace removes the selected non-system layer. Guard: ignore when
	// the user is typing in a form control (the Inspector inputs) so editing text
	// never deletes the layer. window-level listener — browser-only (onMount).
	function isTextEntryTarget(target: EventTarget | null): boolean {
		if (!(target instanceof HTMLElement)) return false;
		const tag = target.tagName;
		return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable;
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (readonly) return;
		// Cmd/Ctrl-S forces an immediate save (pre-empts the debounce). Must run before
		// the text-entry guard so it also works while typing in the Inspector inputs.
		if ((event.metaKey || event.ctrlKey) && (event.key === 's' || event.key === 'S')) {
			event.preventDefault();
			void handleSave();
			return;
		}
		// Esc disarms an active placement tool (before any other handling).
		if (event.key === 'Escape' && activeTool) {
			activeTool = null;
			return;
		}
		if (event.key !== 'Delete' && event.key !== 'Backspace') return;
		if (isTextEntryTarget(event.target)) return;
		// Editor shortcuts are OFF while any modal dialog is up — focus may sit on
		// document.body (core-ui Modal has no focus trap), so a target-based guard
		// can't see the dialog; the ARIA contract can. Covers EVERY Modal consumer
		// (unlink / detach / background confirms) with no cross-package signal.
		if (document.querySelector('[role="dialog"][aria-modal="true"]')) return;
		const id = editor.selectedLayerId;
		if (!id) return;
		const layer = editor.selectedLayer;
		if (!layer || layer.system) return;
		event.preventDefault();
		editor.delete(id);
	}

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		// Re-resolve attached sources live on open (fire-and-forget; never blocks mount).
		void reresolveSources();
		// Load the Google Fonts catalogue (#3) for the FontPicker; [] on failure (degrades
		// to system/brand fonts). Fire-and-forget — the picker renders without it.
		if (loadFontCatalogue) void loadFontCatalogue().then((c) => (fontCatalogue = c)).catch(() => {});
		return () => window.removeEventListener('keydown', handleKeydown);
	});

	// Stop the playback rAF loop when the editor is torn down (navigation/unmount),
	// otherwise a self-rescheduling loop (esp. with loop=true) dangles on the
	// orphaned editor and keeps firing every frame.
	onDestroy(() => {
		editor.pause();
		if (savedTimer) clearTimeout(savedTimer);
		if (saveDebounce) {
			clearTimeout(saveDebounce);
			saveDebounce = null;
		}
		// Flush a pending edit on teardown so navigating away within the debounce window
		// doesn't silently drop it (SPA navigation keeps the in-flight fetch alive).
		// handleSave guards readonly / no-onsave / already-saving itself.
		if (editor.revision !== lastSavedRevision) void handleSave();
	});

	// Flush a pending/dirty auto-save BEFORE any navigation leaves this editor — the
	// Documents-rail switch (a goto to a sibling doc) and the browser Back button both
	// route through here. onDestroy alone only fires AFTER the navigation has committed
	// (SPA teardown order), so the trailing debounced save could be dropped on a fast
	// switch; flushing here persists the current document first. handleSave clears the
	// debounce + persists, and self-guards readonly / no-onsave / in-flight (a readonly
	// — e.g. published — doc has nothing to flush). beforeNavigate is client-only, so
	// this never runs during SSR.
	beforeNavigate(() => {
		if (readonly) return;
		if (editor.revision !== lastSavedRevision) void handleSave();
	});
</script>

<div class="editor" class:editor--readonly={readonly}>
	<TopToolbar
		title={editor.document.name}
		isTemplate={editor.document.isTemplate}
		{backHref}
		{backLabel}
		{showBack}
		{homeHref}
		{eyebrow}
		status={saveStatus}
		animated={animationEnabled}
		showAnimatedToggle={features.animation}
		onToggleAnimated={(v) => editor.setAnimated(v)}
		onexport={() => (exportOpen = true)}
		onpublish={features.publish && onpublish ? handlePublish : undefined}
		published={features.publish ? published : false}
		onunpublish={features.publish ? onunpublish : undefined}
		publishDisabled={features.publish && !publishable}
		publishDisabledReason={publishHint}
		onforcesave={handleSave}
	/>

	<div class="editor__main">
		<div class="editor__left">
			<div class="editor__left-body">
				<LeftRail
					{mode}
					{hiddenRailItems}
					active={railContext}
					onselect={(c) => (railContext = c)}
				/>

				{#if railContext === 'document'}
					<!-- One merged Document panel: current-doc properties on top, the optional
					     consumer documents LIST below (the former separate 'documents' tab). The
					     ＋ create affordance rides in the panel header (headerAction). -->
					<DocumentPanel
						panelTitle={documentsPanel ? 'Documents' : 'Document'}
						title={editor.document.name}
						{artboardWidth}
						{artboardHeight}
						isTemplate={editor.document.isTemplate}
						{documentHasContent}
						gallery={templates}
						{resolveImageSrc}
						{formatters}
						showKind={!documentsPanel}
						kindSelectable={features.kindSelectable}
						kindLabel="Watermark"
						showTemplates={features.documentTemplates && !!templates}
						documentsList={documentsPanel}
						headerAction={documentsHeaderAction}
						settingsAction={documentSettingsAction}
						sizeNote={features.publish && !publishable ? publishHint : undefined}
						onresize={() => (resizeOpen = true)}
						onrename={(name) => { editor.rename(name); ondocumentrename?.(name); }}
						onkindchange={(kind) => editor.setIsTemplate(kind === 'template')}
					/>
				{:else if railContext === 'sources' && features.liveSources}
					<SourcesPanel
						sources={editor.document.sources}
						sourceProviders={pickerProviders}
						onaddsource={(kind) => {
							attachInitialKind = kind;
							attachOpen = true;
						}}
						onchangesource={(alias) => openChange(alias)}
						ondetachsource={(alias) => requestDetach(alias)}
					/>
				{:else if railContext === 'layers'}
					<LayersPanel
						layers={editor.layers}
						pages={editor.pages}
						selectedLayerId={editor.selectedLayerId}
						activePageId={editor.activePage?.id}
						onselect={(id) => editor.select(id)}
						onreorder={(from, to) => editor.reorder(from, to)}
						ontogglevisibility={(id) => editor.toggleVisibility(id)}
						ontogglelock={(id) => editor.toggleLock(id)}
					/>
				{:else if railContext === 'media'}
					<MediaPanel
						uploadSlot={mediaUploadSlot}
						sources={editor.imageSources()}
						canBind={editor.selectedLayer?.type === 'image' && !editor.selectedLayer.system}
						boundAlias={editor.bindingFor(editor.selectedLayer, 'src')?.sourceAlias}
						boundToken={editor.bindingFor(editor.selectedLayer, 'src')?.token}
						onbind={(token, alias) => {
							if (editor.selectedLayerId) editor.bindImageToken(editor.selectedLayerId, token, alias);
						}}
						oninsertimage={(token, alias) => {
							const page = editor.activePage;
							if (page) editor.placeImageLayer(token, page.width / 2, page.height / 2, alias);
						}}
						backgroundSrc={editor.background?.bg_mode === 'image' || editor.background?.bg_mode === 'map'
							? editor.resolvedValue(editor.background, 'src') || undefined
							: undefined}
						bgBoundAlias={editor.bindingFor(editor.background, 'src')?.sourceAlias}
						bgBoundToken={editor.bindingFor(editor.background, 'src')?.token}
						onsetbackground={(token, alias) => requestBackground('image', token, alias)}
						onattach={() => {
							attachInitialKind = editor.activeSource?.kind ?? null;
							attachOpen = true;
						}}
					/>
				{:else if railContext === 'text'}
					<TextsPanel
						oninsert={handleInsert}
						{activeTool}
						fieldGroups={editor.fieldGroups('text')}
						builtinGroups={editor.builtinFieldGroups('text')}
						canBind={editor.selectedLayer?.type === 'text' || editor.selectedLayer?.type === 'callout'}
						boundAlias={editor.bindingFor(editor.selectedLayer, 'content')?.sourceAlias}
						boundToken={editor.bindingFor(editor.selectedLayer, 'content')?.token}
						onbindfield={(field, alias) =>
							editor.selectedLayerId && editor.bindField(editor.selectedLayerId, 'content', field, alias)}
					/>
				{:else if railContext === 'components' && features.components}
					<ComponentsPanel oninsert={handleInsert} {activeTool} />
				{:else if railContext === 'map' && features.map}
					<MapPanel
						sources={editor.imageSources()}
						canBind={editor.selectedLayer?.type === 'image' && !editor.selectedLayer.system}
						boundAlias={editor.bindingFor(editor.selectedLayer, 'src')?.sourceAlias}
						boundToken={editor.bindingFor(editor.selectedLayer, 'src')?.token}
						hasMapSurface={editor.layers.some((l) => l.type === 'map' && !l.system)}
						{activeTool}
						oninsert={handleInsert}
						onbind={(token, alias) => {
							if (editor.selectedLayerId) editor.bindImageToken(editor.selectedLayerId, token, alias);
						}}
						onsetbackground={(token, alias, label) => requestBackground('map', token, alias, label)}
						onattach={() => {
							attachInitialKind = editor.activeSource?.kind ?? null;
							attachOpen = true;
						}}
					/>
				{/if}
			</div>
		</div>

		<div class="editor__canvas-area">
			{#if editor.activePage}
				<CanvasStage
					page={editor.activePage}
					selectedLayerId={editor.selectedLayerId}
					sources={editor.sources}
					{formatters}
					{readonly}
					time={editor.time}
					animated={animationEnabled}
					resolveImageSrc={screenImageSrc}
					{buildMapUrl}
					previewBackdrop={features.previewBackdrop ? editor.previewBackdrop : undefined}
					onselect={(id) => editor.select(id)}
					onupdate={(id, patch) => editor.update(id, patch)}
					onrebindimage={(id, token, alias) => editor.bindImageToken(id, token, alias)}
					onplaceimage={(token, x, y, alias) => editor.placeImageLayer(token, x, y, alias)}
					onrebindfield={(id, field, alias) => editor.bindField(id, 'content', field, alias)}
					onplacefield={(field, x, y, alias) => editor.placeTextLayer(field, x, y, alias)}
					{activeTool}
					onplacepreset={(id, x, y, w, h) => {
						insertPresetAt(id, x, y, w, h);
						activeTool = null; // disarm after a tool-placement (harmless for drag-drops)
					}}
					oncanceltool={() => (activeTool = null)}
				/>
			{/if}
		</div>

		<Inspector
			{editor}
			{brandKit}
			{fontCatalogue}
			showAnimate={features.animation}
			showWatermarkBlock={features.watermarkInspector}
			layer={editor.selectedLayer}
			onupdate={(patch) => editor.selectedLayerId && editor.update(editor.selectedLayerId, patch)}
			ontogglevisibility={() => editor.selectedLayerId && editor.toggleVisibility(editor.selectedLayerId)}
			ondelete={() => editor.selectedLayerId && editor.delete(editor.selectedLayerId)}
			onanimate={(p) => openAnimDialog(p, 0)}
			oneditanimate={(p) => editor.selectedLayerId && openAnimInfo(editor.selectedLayerId, p)}
			onopenlayers={() => (railContext = 'layers')}
			onreveal={(ctx) => (railContext = ctx)}
			{getEnumAlias}
			{onaliaschange}
		/>
	</div>

	<!-- Timeline only exists in animated mode (R3): hidden ENTIRELY (not collapsed) for a
	     static document. Authoring an animation flips editor.animated → it appears. A
	     mode without animation (watermark · #0298) never shows it regardless of the doc. -->
	{#if editor.activePage && animationEnabled}
		<Timeline
			page={editor.activePage}
			{editor}
			expanded={timelineExpanded}
			ontoggle={() => (timelineExpanded = !timelineExpanded)}
			onaddanimation={openAnimDialogForLayer}
			oneditanimation={openAnimInfo}
		/>
	{/if}
</div>

<!-- "+ Add" source opens a MODAL (never a redirect — hard rule). SourceBrowser spans every
     registered kind (chooser → picker); multi-select confirms via its own Add(N) footer, the
     Modal ✕ cancels. onconfirm resolves each record + attaches it under the next free alias. -->
<Modal
	open={attachOpen}
	title="Attach a source"
	onclose={() => {
		attachOpen = false;
		attachInitialKind = null;
		attachError = null;
	}}
>
	{#if pickerProviders.length > 0}
		<SourceBrowser
			providers={pickerProviders}
			initialKind={attachInitialKind ?? undefined}
			select="multi"
			onconfirm={handleAttachConfirm}
		/>
		{#if attaching || attachError}
			<p
				class="attach-modal__status"
				class:attach-modal__status--error={!!attachError}
				role={attachError ? 'alert' : 'status'}
			>
				{attachError ?? 'Attaching…'}
			</p>
		{/if}
	{:else}
		<p class="attach-modal__intro">No source provider is available.</p>
	{/if}
</Modal>

<!-- "Change" a source: a single-select picker for THAT source's kind, pre-highlighting the
     current record. The footer "Replace" re-attaches under the SAME alias so every binding on
     that source follows. ✕ cancels. A resolve failure is surfaced under the picker, never silent. -->
<Modal open={changeOpen} title="Change source" onclose={closeChange}>
	{#if changeProviders.length > 0}
		<!-- Keyed on the current record so the picker re-mounts (and re-seeds its pre-highlight)
		     for each source changed — explicit re-seed, not relying solely on the Modal's unmount. -->
		{#key changeRefId}
			<SourceBrowser
				providers={changeProviders}
				initialKind={changeKind ?? undefined}
				select="single"
				confirm
				confirmLabel="Replace"
				selectedIds={changeRefId ? [changeRefId] : []}
				onpick={handleChangeConfirm}
			/>
		{/key}
		{#if changing || changeError}
			<p
				class="attach-modal__status"
				class:attach-modal__status--error={!!changeError}
				role={changeError ? 'alert' : 'status'}
			>
				{changeError ?? 'Loading…'}
			</p>
		{/if}
	{/if}
</Modal>

<!-- Detach-confirm: a source with bound layers warns before detaching (the bindings would
     fall back to a placeholder). A source with zero bound layers detaches with no prompt. -->
<Modal open={detachOpen} title="Detach source" size="md" onclose={closeDetach}>
	<p class="detach-msg">
		<strong>{detachCount}</strong>
		{detachCount === 1 ? 'layer is' : 'layers are'} bound to “{detachTitle}”. Detaching unlinks
		{detachCount === 1 ? 'it' : 'them'} — the {detachCount === 1 ? 'layer' : 'layers'} will fall back to
		a placeholder value until you bind another source.
	</p>
	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={closeDetach}>Cancel</Button>
		<Button variant="primary" size="sm" onclick={confirmDetach}>Detach</Button>
	{/snippet}
</Modal>

<!-- Background-apply confirm — replacing a user-set background (or any MAP apply)
     asks first; the previous background face is replaced, not stacked. -->
<Modal open={!!bgConfirm} title="Set page background" size="sm" onclose={() => (bgConfirm = null)}>
	<p class="detach-msg">
		{#if editor.backgroundIsDefault()}
			Set {bgConfirm?.label ? `“${bgConfirm.label}”` : 'this image'} as the page background?
		{:else}
			Replace the current background with {bgConfirm?.label ? `“${bgConfirm.label}”` : 'this image'}?
			The previous background is discarded.
		{/if}
	</p>
	{#snippet footer()}
		<Button variant="secondary" size="sm" onclick={() => (bgConfirm = null)}>Cancel</Button>
		<Button
			variant="primary"
			size="sm"
			onclick={() => {
				if (bgConfirm) applyBackground(bgConfirm.mode, bgConfirm.token, bgConfirm.alias);
				bgConfirm = null;
			}}>Set background</Button>
	{/snippet}
</Modal>

<!-- Resize the active artboard — FUNCTIONAL (mutates page.width/height; the stage
     re-fits). The dialog lives here so its Modal portals without clipping. In watermark
     mode (#0298 · Slice S1) it presents the iPhone-Photos-style orientation/ratio control
     capped at the watermark max; design mode passes neither → the classic preset grid. -->
<ResizeDialog
	open={resizeOpen}
	width={artboardWidth}
	height={artboardHeight}
	ratios={features.watermarkPhotoSizing ? WATERMARK_RATIOS : undefined}
	maxDimension={features.watermarkPhotoSizing ? WATERMARK_MAX_DIMENSION : undefined}
	onclose={() => (resizeOpen = false)}
	onapply={(w, h) => editor.resizePage(w, h)}
/>

<!-- Export dialog (R4) — replaces the direct Export action. Picks image format + page scope;
     handleExport rasterizes. Save-as-template is deferred (no template registry backend yet). -->
<ExportDialog
	open={exportOpen}
	format={editor.exportFormat}
	pageCount={editor.pages.length}
	animated={editor.animated}
	width={artboardWidth}
	height={artboardHeight}
	{exporting}
	onclose={() => (exportOpen = false)}
	onexport={(f, s) => {
		editor.setExportFormat(f);
		exportOpen = false;
		void handleExport(s);
	}}
/>

<!-- Author a property animation. Opened from the Inspector ≣ (selected layer +
     property @0s) OR the timeline gray-track click (clicked layer @clicked time, no
     preselected property); builds a 2-keyframe track via the kit. -->
<AddAnimationDialog
	open={animOpen}
	{editor}
	layerId={animLayerId}
	property={animProperty}
	startMs={animStartMs}
	onclose={() => (animOpen = false)}
/>

<!-- Edit / delete an existing property animation (slice-5b, §12.5 double-click).
     Opened from a timeline bar double-click; live-edits the track via the kit. -->
<AnimationInfoPopover
	open={infoOpen}
	{editor}
	layerId={infoLayerId}
	property={infoProperty}
	onclose={() => (infoOpen = false)}
/>

<style>
	.editor {
		display: flex;
		flex-direction: column;
		height: 100%;
		min-height: 0;
		background: var(--cv-color-neutral-100, #f5f5f5);
		color: var(--cv-color-neutral-800, #202020);
		font-family: var(--cv-font-body, system-ui, sans-serif);
	}

	.editor__main {
		flex: 1;
		display: flex;
		min-height: 0;
		overflow: hidden;
	}

	/* Left column: rail + the contextual panel (the brand card moved to the toolbar). */
	.editor__left {
		display: flex;
		flex-direction: column;
		min-height: 0;
	}

	.editor__left-body {
		flex: 1;
		display: flex;
		min-height: 0;
	}

	/* Canvas area — wraps the stage, which owns its own zoom/dims badges + fit. */
	.editor__canvas-area {
		position: relative;
		flex: 1;
		display: flex;
		min-width: 0;
	}

	.attach-modal__intro {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.5;
		color: var(--cv-color-neutral-600, #505050);
	}

	/* Detach-confirm message — readable body copy, full width (never clipped). */
	.detach-msg {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.5;
		color: var(--cv-color-neutral-700, #383838);
	}

	/* Attach busy/error feedback — shown under the browser so a resolve failure is never
	   silent (the footer that used to carry this was removed; Add(N) lives inside the picker). */
	.attach-modal__status {
		margin: var(--cv-space-sm, 0.5rem) 0 0;
		font-size: 0.8125rem;
		line-height: 1.4;
		color: var(--cv-color-neutral-500, #707070);
	}

	.attach-modal__status--error {
		color: var(--cv-color-error, #c0392b);
	}

	/* Visible slim scrollbars on every editor scroll region (the Inspector, the left
	   contextual-panel body, and the timeline track list). macOS overlay scrollbars
	   auto-hide, so a region taller than its viewport — e.g. the Inspector once the
	   timeline is expanded on a short screen — READ as clipped: the affordance to
	   reach the rows below the fold (Width/Height/Opacity/Rotation) was invisible.
	   One definition for all three (never-copy-paste-css), scoped inside .editor so
	   it never leaks to the rest of the app. scrollbar-gutter keeps the track
	   reserved so toggling the timeline never shifts the panel content. */
	.editor :global(.inspector),
	.editor :global(.panel__body),
	.editor :global(.tracks) {
		scrollbar-gutter: stable;
	}

	.editor :global(.inspector)::-webkit-scrollbar,
	.editor :global(.panel__body)::-webkit-scrollbar,
	.editor :global(.tracks)::-webkit-scrollbar {
		width: 10px;
		height: 10px;
	}

	.editor :global(.inspector)::-webkit-scrollbar-thumb,
	.editor :global(.panel__body)::-webkit-scrollbar-thumb,
	.editor :global(.tracks)::-webkit-scrollbar-thumb {
		background: var(--cv-color-neutral-300, #d9d9d9);
		border: 3px solid var(--cv-color-surface, #fff);
		border-radius: 999px;
	}

	.editor :global(.inspector)::-webkit-scrollbar-thumb:hover,
	.editor :global(.panel__body)::-webkit-scrollbar-thumb:hover,
	.editor :global(.tracks)::-webkit-scrollbar-thumb:hover {
		background: var(--cv-color-neutral-400, #a0a0a0);
	}
</style>
