// Reactive editor state — the Svelte 5 wrapper over the headless kit reducers.
// The kit (@sbx/canvas-kit) stays framework-agnostic; this class adds $state
// reactivity so components re-render when the document or selection changes.
// Every mutation routes through a pure kit reducer, so the editor's behaviour is
// the same logic the headless store tests cover.

import {
	type Document,
	type Layer,
	type MapConfig,
	type Page,
	type EditorState,
	type AnimationSpec,
	type AnimatableProperty,
	type SourceProvider,
	type SourceRef,
	type SourceSnapshot,
	type ResolvedImage,
	type ImageCategory,
	type FieldDescriptor,
	type FormatterRegistry,
	type FormatterDescriptor,
	type FormatOpts,
	type Binding,
	type LayerDefaults,
	type WatermarkParams,
	type PreviewBackdrop,
	NEUTRAL_LAYER_DEFAULTS,
	addLayer,
	reorderLayer,
	resizePage,
	setPageDuration,
	setDocumentMeta,
	selectLayer,
	updateLayer,
	deleteLayer,
	addAnimation,
	updateAnimation,
	removeAnimation,
	nextSourceAlias,
	attachSource,
	detachSource,
	setSourceSnapshot,
	bindLayer,
	unbindLayer,
	setBindingOverride,
	clearBindingOverride,
	resolveBindings,
	resolveTemplateContent,
	splitSourceToken,
	descriptorDefaults,
	presetToLayer,
	resolveAnchorTarget,
	anchorAnnotationToMap,
	findPage,
	type BackgroundSpec,
	backgroundFor,
	ensureBackgroundLayers,
	setBackground,
	isDefaultBackground
} from '@sbx/canvas-kit';
import { parseTemplate, type Template, type TemplateToken, type TokenResolver, type TokenCatalog } from '@sbx/text-template';
import type { ExportFormat } from './download.js';
import { sourceTitle } from './source-display.js';

/** A per-source image bundle for the Media panel's [Sources ▾] filter (G6b) — one entry
 *  per attached, image-capable source. */
export interface ImageSourceView {
	alias: string;
	kind: string;
	/** Provider label (e.g. "Property") for the filter option + empty states. */
	label: string;
	/** The record's human title (e.g. "Freehold Hillside"). */
	title: string;
	/** Resolved images keyed by group id ({ cover, gallery, … }). */
	images: Record<string, ResolvedImage[]>;
	/** Image groups the source's provider offers. */
	categories: ImageCategory[];
}

/** Which provider field types may bind to a given layer property (D20 type-matching). */
function fieldMatches(field: FieldDescriptor, kind: 'text' | 'image'): boolean {
	if (field.category !== 'field') return false;
	return kind === 'image' ? field.type === 'image' : field.type !== 'image';
}

/** Sensible base value for an animatable property when the layer hasn't set one
 *  (the "From" prefill, §12.4). Model units: opacity/scale fractions, angle deg. */
function baseValue(layer: Layer, property: AnimatableProperty): number {
	switch (property) {
		case 'opacity':
			return layer.opacity ?? 1;
		case 'scale':
			return layer.scale ?? 1;
		case 'rotation':
			return layer.rotation ?? 0;
		case 'width':
			return layer.width ?? 0;
		case 'height':
			return layer.height ?? 0;
		default:
			return layer[property] ?? 0; // x | y
	}
}

/** Reactive holder for the open document + current selection. */
export class CanvasEditor {
	private state = $state<EditorState>({ document: { id: '', name: '', pages: [] }, selectedLayerId: null });

	/** Monotonic counter of PERSISTED document mutations — the auto-save change signal
	 *  (R2). Bumped ONLY by content changes that survive `toPersist` (layers, sources,
	 *  bindings, animations, artboard, meta); selection, snapshot re-resolve-on-open,
	 *  and playback reassign `this.state` directly and never bump it, so opening a
	 *  document or clicking a layer never triggers a write. */
	private revisionState = $state(0);

	/** Playhead time in ms — what the stage renders + the timeline scrubs (slice-5a). */
	time = $state(0);
	/** Whether on-canvas playback is running (rAF loop active). */
	playing = $state(false);
	/** Loop playback at the page end instead of stopping. */
	loop = $state(false);

	/** Default export format (S-EXPORT) — the Export action reads it; the export
	 *  dialog will set it (R4). Session state; not persisted to the document. */
	exportFormat = $state<ExportFormat>('png');

	#rafId: number | null = null;
	#lastTs = 0;

	/** Registered source providers (BR feeds property/guide/service). Set once by the shell;
	 *  the picker calls their async list/resolve, the binding UI reads their sync fields(). */
	#providers: SourceProvider[] = [];
	/** Formatter registry for binding display (currency ฿ / area rai…). Set once by the shell. */
	#formatters: FormatterRegistry = {};
	/** Defaults for NEW layers the editor materialises itself (placeImageLayer) —
	 *  Decision #0286 Phase A. Seeded at construction; the shell re-assigns it when
	 *  the consumer builds brand defaults client-side after mount (CSS tokens need
	 *  the DOM). Read imperatively at call time — plain field, not $state. */
	layerDefaults: LayerDefaults = NEUTRAL_LAYER_DEFAULTS;

	/** BUILT-IN sources (#0286 refinement): always-on refs the consumer passes by
	 *  code (e.g. Branding — reserved alias). NEVER persisted (toPersist reads
	 *  document.sources only) and never offered in the attach/detach UI; they ride
	 *  every read path (resolution, Media panel, snapshots) via the merged
	 *  `sources` getter. $state — panels render from it and the consumer builds the
	 *  refs client-side after mount. */
	builtinSources = $state<SourceRef[]>([]);

	constructor(
		doc: Document,
		opts: {
			providers?: SourceProvider[];
			formatters?: FormatterRegistry;
			layerDefaults?: LayerDefaults;
			builtinSources?: SourceRef[];
		} = {}
	) {
		// Normalize on entry: every page carries its system background at index 0
		// (docs from the server / templates predate backgrounds). Silent — no
		// revision bump, so opening a document never dirties it; the background
		// persists with the first real edit.
		this.state = ensureBackgroundLayers({ document: doc, selectedLayerId: null });
		this.#providers = opts.providers ?? [];
		this.#formatters = opts.formatters ?? {};
		this.layerDefaults = opts.layerDefaults ?? NEUTRAL_LAYER_DEFAULTS;
		this.builtinSources = opts.builtinSources ?? [];
	}

	get document(): Document {
		return this.state.document;
	}

	get selectedLayerId(): string | null {
		return this.state.selectedLayerId;
	}

	/** The persisted-mutation revision (R2). Read reactively by the shell's debounced
	 *  auto-save; advances on every undoable document change, never on selection. */
	get revision(): number {
		return this.revisionState;
	}

	/** Commit a PERSISTED, document-dirtying mutation: assign the new editor state and
	 *  bump the revision so the shell's debounced auto-save fires. Every mutator that
	 *  changes the `toPersist` shape routes through here; selection / snapshot-refresh /
	 *  playback bypass it (they assign `this.state` directly, no revision bump). */
	private commit(next: EditorState): void {
		// Reducer no-op (returned BY REFERENCE — e.g. the builtin-alias attach refusal,
		// or an unknown-id update): never dirty the document or trigger an auto-save.
		if (next === this.state) return;
		this.state = next;
		this.revisionState++;
	}

	/** All pages of the open document (frame navigator + Document panel read this). */
	get pages(): Page[] {
		return this.state.document.pages;
	}

	/** The first page — the only page this slice edits (multi-page is a later slice). */
	get activePage(): Page | undefined {
		return this.state.document.pages[0];
	}

	/** Flat layer list of the active page (array order = z-order, back→front). */
	get layers(): Layer[] {
		return this.activePage?.layers ?? [];
	}

	/** True if any layer in the document carries animation tracks (drives the
	 *  animated-mode default). */
	get hasAnimations(): boolean {
		return this.state.document.pages.some((p) => p.layers.some((l) => !!l.animations?.length));
	}

	/** R3 animated mode — the explicit doc flag, or (when unset) derived: animated iff
	 *  the document already carries animation tracks. Drives the timeline + render path. */
	get animated(): boolean {
		return this.state.document.animated ?? this.hasAnimations;
	}

	/** The currently selected layer object, or undefined when nothing is selected. */
	get selectedLayer(): Layer | undefined {
		const id = this.state.selectedLayerId;
		if (!id) return undefined;
		return this.layers.find((l) => l.id === id);
	}

	select(layerId: string | null): void {
		this.state = selectLayer(this.state, layerId);
	}

	/** Set the default export format (S-EXPORT). */
	setExportFormat(format: ExportFormat): void {
		this.exportFormat = format;
	}

	add(layer: Layer): void {
		const page = this.activePage;
		if (!page) return;
		this.commit(addLayer(this.state, page.id, layer));
	}

	reorder(fromIndex: number, toIndex: number): void {
		const page = this.activePage;
		if (!page) return;
		// "Move to the bottom" lands just above the system background — the kit
		// refuses index 0 outright, which would silently swallow the gesture.
		const to = toIndex === 0 && page.layers[0]?.system && fromIndex !== 0 ? 1 : toIndex;
		this.commit(reorderLayer(this.state, page.id, fromIndex, to));
	}

	/** Rename the document (Settings → Document settings title field). Blank /
	 *  whitespace-only names are ignored by the reducer (title stays non-blank). */
	rename(name: string): void {
		this.commit(setDocumentMeta(this.state, { name }));
	}

	/** Set the document/template kind (Settings → Document settings kind select).
	 *  Persistence maps `isTemplate` → the server `kind` column. */
	setIsTemplate(isTemplate: boolean): void {
		this.commit(setDocumentMeta(this.state, { isTemplate }));
	}

	/** Toggle R3 animated mode. OFF pauses playback (the timeline disappears + the canvas
	 *  renders static); the animation tracks are KEPT (non-destructive — turning it back
	 *  ON restores motion). */
	setAnimated(animated: boolean): void {
		if (!animated) this.pause();
		this.commit(setDocumentMeta(this.state, { animated }));
	}

	/** The document-level watermark params (Slice A · #0298) — undefined until the
	 *  Watermark inspector block is first edited (the block reads this; missing = the
	 *  defaults shown in the controls until the first patch seeds it). */
	get watermark(): WatermarkParams | undefined {
		return this.state.document.watermark;
	}

	/** Patch the document's watermark params (the Watermark inspector block). A PARTIAL
	 *  merge: the reducer seeds the full defaults on first edit, then merges the patch,
	 *  so a single knob never drops the others. Routes through `commit` → one undoable
	 *  revision → the shell's debounced auto-save, exactly like the other doc mutators. */
	setWatermark(patch: Partial<WatermarkParams>): void {
		this.commit(setDocumentMeta(this.state, { watermark: patch }));
	}

	/** The document-level PREVIEW backdrop (Slice A2.3 · #0298) — the colour/image the
	 *  author previews the transparent mark over. Undefined until the Watermark
	 *  inspector's backdrop control is first changed (missing → the stage shows the
	 *  transparent checker). PREVIEW-ONLY: rendered as stage chrome, never a layer, so
	 *  it is structurally excluded from the export path. */
	get previewBackdrop(): PreviewBackdrop | undefined {
		return this.state.document.previewBackdrop;
	}

	/** Patch the document's preview backdrop (the Watermark inspector backdrop control).
	 *  A PARTIAL merge: the reducer seeds `none` on first edit, then merges the patch, so
	 *  toggling kind never drops a previously chosen colour/image. Routes through `commit`
	 *  → one undoable revision → the shell's debounced auto-save, exactly like the other
	 *  doc mutators (the value persists in the doc JSONB as a preview preference but never
	 *  reaches the exported pixels). */
	setPreviewBackdrop(patch: Partial<PreviewBackdrop>): void {
		this.commit(setDocumentMeta(this.state, { previewBackdrop: patch }));
	}

	/** Resize the active page's artboard (the Resize dialog calls this). The new
	 *  state is re-assigned so $state reactivity fires and CanvasStage re-fits. */
	resizePage(width: number, height: number): void {
		const page = this.activePage;
		if (!page) return;
		this.commit(resizePage(this.state, page.id, width, height));
	}

	// ── Background (user directive 2026-06-06; land-canvas parity) ──────────────

	/** The active page's system background layer (always present — normalized on entry). */
	get background(): Layer | undefined {
		return this.activePage ? backgroundFor(this.activePage) : undefined;
	}

	/** Is the active page's background still the untouched white default? Drives the
	 *  replace-confirm policy (replacing a USER-set background — image, map, or an
	 *  explicitly chosen color — asks first; covering pristine white never does). */
	backgroundIsDefault(): boolean {
		const bg = this.background;
		return !!bg && isDefaultBackground(bg);
	}

	/** Swap the background's face (color | image | map). Pure face swap — image/map
	 *  callers that LINK a source token use `setBackgroundImage` instead. */
	setBackground(spec: BackgroundSpec): void {
		const page = this.activePage;
		if (!page) return;
		this.commit(setBackground(this.state, page.id, spec));
	}

	/** Set the background to a source-linked image/map token in ONE commit (face swap
	 *  + src binding together — a single revision, exactly like a thumbnail bind). */
	setBackgroundImage(mode: 'image' | 'map', token: string, alias?: string): void {
		const page = this.activePage;
		const bg = this.background;
		const a = alias ?? this.activeSource?.alias;
		if (!page || !bg || !a) return;
		const swapped = setBackground(this.state, page.id, { mode });
		const binding: Binding = { property: 'src', sourceAlias: a, token, placeholder: true };
		this.commit(bindLayer(swapped, bg.id, binding));
	}

	/** Select the background layer (the Media/Map affordances + Layers panel route here). */
	selectBackground(): void {
		const bg = this.background;
		if (bg) this.select(bg.id);
	}

	update(layerId: string, patch: Partial<Layer>): void {
		this.commit(updateLayer(this.state, layerId, patch));
	}

	/** Remove a layer (Delete key + Inspector trash). No-op for system layers;
	 *  the reducer clears the selection when the removed layer was selected. */
	delete(layerId: string): void {
		this.commit(deleteLayer(this.state, layerId));
	}

	toggleVisibility(layerId: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		if (!layer) return;
		this.update(layerId, { visible: !(layer.visible !== false) });
	}

	toggleLock(layerId: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		if (!layer) return;
		this.update(layerId, { locked: !layer.locked });
	}

	// --- Animation authoring (slice-5a) — thin wrappers over kit reducers ----

	/** The active page's frame duration in ms (the timeline length). */
	get pageDurationMs(): number {
		return this.activePage?.duration ?? 5000;
	}

	/** Set the active page's frame duration (the `Page duration` control). */
	setPageDuration(durationMs: number): void {
		const page = this.activePage;
		if (!page) return;
		this.commit(setPageDuration(this.state, page.id, durationMs));
		if (this.time > this.pageDurationMs) this.seek(this.pageDurationMs);
	}

	/** Add (or replace) a property's 2-keyframe animation on a layer (§12.4). Authoring
	 *  an animation marks the doc animated (reveals the timeline even if the flag was
	 *  explicitly OFF) — one atomic commit. */
	addAnimation(layerId: string, spec: AnimationSpec): void {
		this.commit(setDocumentMeta(addAnimation(this.state, layerId, spec), { animated: true }));
	}

	/** Patch an existing property animation (popup edits + 5b grabber drag). */
	updateAnimation(layerId: string, property: AnimatableProperty, patch: Partial<Omit<AnimationSpec, 'property'>>): void {
		this.commit(updateAnimation(this.state, layerId, property, patch));
	}

	/** Remove a property's animation (§12.5 delete). */
	removeAnimation(layerId: string, property: AnimatableProperty): void {
		this.commit(removeAnimation(this.state, layerId, property));
	}

	/** The base value to prefill "From" with when authoring a new animation. */
	baseValueFor(layer: Layer, property: AnimatableProperty): number {
		return baseValue(layer, property);
	}

	/** Is this property already animated on the layer? (drives Inspector ≣ active state) */
	isAnimated(layer: Layer | undefined, property: AnimatableProperty): boolean {
		return !!layer?.animations?.some((t) => t.property === property);
	}

	// --- Playback transport (on-canvas, rAF — §12.5) ------------------------

	togglePlay(): void {
		this.playing ? this.pause() : this.play();
	}

	/** Start the rAF playback loop from the current time (restarts at 0 when at the end). */
	play(): void {
		if (this.playing) return;
		if (typeof requestAnimationFrame === 'undefined') return; // SSR guard
		if (this.time >= this.pageDurationMs) this.time = 0;
		this.playing = true;
		this.#lastTs = performance.now();
		const tick = (ts: number): void => {
			if (!this.playing) return;
			const dt = ts - this.#lastTs;
			this.#lastTs = ts;
			const dur = this.pageDurationMs;
			let next = this.time + dt;
			if (next >= dur) {
				if (this.loop) {
					next = dur > 0 ? next % dur : 0;
				} else {
					next = dur;
					this.playing = false;
				}
			}
			this.time = next;
			if (this.playing) this.#rafId = requestAnimationFrame(tick);
		};
		this.#rafId = requestAnimationFrame(tick);
	}

	/** Stop playback, leaving the playhead where it is. */
	pause(): void {
		this.playing = false;
		if (this.#rafId !== null) {
			cancelAnimationFrame(this.#rafId);
			this.#rafId = null;
		}
	}

	/** Move the playhead to `ms` (clamped to [0, page duration]); pauses playback. */
	seek(ms: number): void {
		this.pause();
		this.time = Math.min(Math.max(0, ms), this.pageDurationMs);
	}

	// --- Sources & data-binding (slice 4a) — thin reactive wrappers over kit reducers.
	//     Single 'primary' source this slice; multi-source/alias mgmt is a later slice.

	/** Registered providers (the attach picker + binding UI read these). */
	get providers(): SourceProvider[] {
		return this.#providers;
	}

	/** True when any source provider is registered (drives the 🏷 enabled state). */
	get hasProviders(): boolean {
		return this.#providers.length > 0;
	}

	/** EVERY source visible to resolution + panels: the document's attached refs plus
	 *  the injected built-ins (#0286 — built-ins last, so 'primary' keeps anchoring
	 *  the Media panel default and the attach-order display). Persistence reads
	 *  `document.sources` directly and never sees the built-ins. */
	get sources(): SourceRef[] {
		const attached = this.state.document.sources ?? [];
		return this.builtinSources.length ? [...attached, ...this.builtinSources] : attached;
	}

	/** The Active record — alias 'primary' — that drives the preview + the field chips. */
	get activeSource(): SourceRef | undefined {
		return this.sources.find((s) => s.alias === 'primary');
	}

	/** The provider for a given source kind. */
	providerFor(kind: string): SourceProvider | undefined {
		return this.#providers.find((p) => p.kind === kind);
	}

	/** The provider backing the Active source (whose fields() drive the chips + bind picker). */
	get activeProvider(): SourceProvider | undefined {
		return this.activeSource ? this.providerFor(this.activeSource.kind) : undefined;
	}

	/** The Active source provider's declared fields (chips + content rows). */
	sourceFields(): FieldDescriptor[] {
		return this.activeProvider?.fields() ?? [];
	}

	/** Active-source fields that may bind to a layer property of the given kind (type-matched). */
	fieldsFor(kind: 'text' | 'image'): FieldDescriptor[] {
		return this.sourceFields().filter((f) => fieldMatches(f, kind));
	}

	/** Fields of a SPECIFIC source's provider that may bind to a property of the given
	 *  kind — the Inspector's [item ▾] dropdown lists the BINDING's own source, never
	 *  the Active one (`fieldsFor` would re-anchor a secondary-bound row to 'primary').
	 *  Built-ins included; unknown alias → empty. */
	fieldsForAlias(alias: string, kind: 'text' | 'image'): FieldDescriptor[] {
		const ref = this.sources.find((s) => s.alias === alias);
		const provider = ref ? this.providerFor(ref.kind) : undefined;
		return (provider?.fields() ?? []).filter((f) => fieldMatches(f, kind));
	}

	/** Bindable field groups of the BUILT-IN sources (#0286) — one group per injected
	 *  ref whose provider declares matching fields (e.g. Branding → "Company name").
	 *  Rendered by the panels as extra chip sections; chips bind with the group alias. */
	builtinFieldGroups(kind: 'text' | 'image'): { alias: string; label: string; fields: FieldDescriptor[] }[] {
		const out: { alias: string; label: string; fields: FieldDescriptor[] }[] = [];
		for (const ref of this.builtinSources) {
			const provider = this.providerFor(ref.kind);
			const fields = (provider?.fields() ?? []).filter((f) => fieldMatches(f, kind));
			if (fields.length === 0) continue;
			out.push({ alias: ref.alias, label: provider?.label ?? ref.kind, fields });
		}
		return out;
	}

	/** The resolved token → value snapshot for an alias (filled live on open). */
	snapshotFor(alias: string): SourceSnapshot {
		return this.sources.find((s) => s.alias === alias)?.snapshot ?? {};
	}

	/** The resolved image groups for an alias's snapshot (E2) — { cover, gallery, … }.
	 *  The Media panel lists these; a thumbnail click binds `images.<group>.<index>.url`. */
	imagesFor(alias: string): Record<string, ResolvedImage[]> {
		const imgs = this.snapshotFor(alias).images;
		return imgs && typeof imgs === 'object' ? (imgs as Record<string, ResolvedImage[]>) : {};
	}

	/** Bindable field groups for ALL attached sources (PS-1 per-source blocks in Texts +
	 *  future panels). Returns one entry per source that declares ≥1 matching field (attach
	 *  order; built-ins excluded — they use `builtinFieldGroups`). Each entry carries the
	 *  source's alias so chip drag/click bind to THAT source without re-pointing. */
	fieldGroups(kind: 'text' | 'image'): { alias: string; label: string; title: string; fields: FieldDescriptor[] }[] {
		const out: { alias: string; label: string; title: string; fields: FieldDescriptor[] }[] = [];
		// Only iterate attached document sources (not built-ins — they have their own section).
		const attached = this.state.document.sources ?? [];
		for (const ref of attached) {
			const provider = this.providerFor(ref.kind);
			const fields = (provider?.fields() ?? []).filter((f) => fieldMatches(f, kind));
			if (fields.length === 0) continue;
			out.push({
				alias: ref.alias,
				label: provider?.label ?? ref.kind,
				title: sourceTitle(ref),
				fields
			});
		}
		return out;
	}

	/** The attached, IMAGE-CAPABLE sources for the Media panel's [Sources ▾] filter (G6b):
	 *  every attached source whose provider declares image groups, each bundled with its
	 *  resolved images + a display label/title. Order = attach order (primary first). */
	imageSources(): ImageSourceView[] {
		const out: ImageSourceView[] = [];
		for (const ref of this.sources) {
			const provider = this.providerFor(ref.kind);
			const categories = provider?.imageCategories?.() ?? [];
			if (categories.length === 0) continue; // not an image source — skip
			out.push({
				alias: ref.alias,
				kind: ref.kind,
				label: provider?.label ?? ref.kind,
				title: sourceTitle(ref),
				images: this.imagesFor(ref.alias),
				categories
			});
		}
		return out;
	}

	/** The next free alias for an additive attach — the kit's collision-safe ladder
	 *  (primary → secondary → … → source-N). First attach is 'primary' (keeps the Media
	 *  panel + canvas preview anchored). */
	nextAlias(): string {
		return nextSourceAlias(this.sources.map((s) => s.alias));
	}

	/** Attach (or replace) a record under an explicit alias — the additive multi-source path.
	 *  `attachSource` replaces a same-alias ref, so re-attaching 'primary' swaps the active record. */
	attachAs(alias: string, kind: string, refId: string, snapshot: SourceSnapshot): void {
		const ref: SourceRef = { id: `${alias}:${kind}:${refId}`, kind, refId, alias, snapshot };
		this.commit(attachSource(this.state, ref));
	}

	/** Attach (or replace) the Active 'primary' source. Thin alias over `attachAs('primary', …)`. */
	attachPrimary(kind: string, refId: string, snapshot: SourceSnapshot): void {
		this.attachAs('primary', kind, refId, snapshot);
	}

	/** Attach a source ref directly (re-resolve-on-open fills the snapshot via setSnapshot). */
	attachSource(ref: SourceRef): void {
		this.commit(attachSource(this.state, ref));
	}

	/** Remove an attached source (its bindings fall back to `fallback`). */
	detachSource(alias: string): void {
		this.commit(detachSource(this.state, alias));
	}

	/** How many layers (across ALL pages) bind any property to this source alias — drives the
	 *  detach confirm ("N bound layers will lose their source"). */
	boundLayerCount(alias: string): number {
		let n = 0;
		for (const page of this.state.document.pages) {
			for (const layer of page.layers) {
				if (layer.bindings?.some((b) => b.sourceAlias === alias)) n++;
			}
		}
		return n;
	}

	/** Fill the live snapshot for an alias (re-resolve-on-open). */
	setSnapshot(alias: string, snapshot: SourceSnapshot): void {
		// Built-in refs live OUTSIDE the document state — refresh the injected copy.
		// REPLACE the array (never index-mutate: the shell hands us the route's own
		// $state array by reference; mutating a parent-owned proxy couples owners).
		const builtinIdx = this.builtinSources.findIndex((s) => s.alias === alias);
		if (builtinIdx !== -1) {
			this.builtinSources = this.builtinSources.map((s, i) => (i === builtinIdx ? { ...s, snapshot } : s));
			return;
		}
		this.state = setSourceSnapshot(this.state, alias, snapshot);
	}

	/** The binding on a layer for a property, if any. */
	bindingFor(layer: Layer | undefined, property: string): Binding | undefined {
		return layer?.bindings?.find((b) => b.property === property);
	}

	/** Switch a binding to another attached source, keeping token/format/fallback (and
	 *  any override) — the Inspector source-dropdown swap (E-3): "same slot, other
	 *  record". Tolerant by design: a source missing the token resolves to the
	 *  binding fallback until another item is picked. */
	rebindAlias(layerId: string, property: string, alias: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		const binding = this.bindingFor(layer, property);
		if (!binding || binding.sourceAlias === alias) return;
		this.commit(bindLayer(this.state, layerId, { ...binding, sourceAlias: alias }));
	}

	/** Display summaries of every attached source (alias + provider label + record
	 *  title) — the Inspector's source dropdown (E-3). Order = attach order. */
	sourceSummaries(): { alias: string; kind: string; label: string; title: string }[] {
		return this.sources.map((ref) => ({
			alias: ref.alias,
			kind: ref.kind,
			label: this.providerFor(ref.kind)?.label ?? ref.kind,
			title: sourceTitle(ref)
		}));
	}

	/** Is the property linked to a source field? */
	isBound(layer: Layer | undefined, property: string): boolean {
		return !!this.bindingFor(layer, property);
	}

	/** Is the property linked AND manually overridden (fine-tuned)? */
	isOverridden(layer: Layer | undefined, property: string): boolean {
		return this.bindingFor(layer, property)?.override != null;
	}

	/** Current display value of a (bound, templated, or static) property — override-aware (D20). */
	resolvedValue(layer: Layer | undefined, property: string): string {
		if (!layer) return '';
		// Template content (INC-C) resolves CROSS-source — each token routes to its own source via
		// an optional '<alias>.' prefix (bare tokens → 'primary'). The SAME resolver the artboard
		// (resolveFrame) and export use, so the flatten-to-text path never diverges from the canvas.
		if (property === 'content' && layer.content_template?.length) {
			const byAlias = new Map(this.sources.map((s) => [s.alias, this.snapshotFor(s.alias)]));
			return resolveTemplateContent(layer.content_template, byAlias, this.#formatters, 'primary');
		}
		const binding = this.bindingFor(layer, property);
		if (!binding) return String((layer as unknown as Record<string, unknown>)[property] ?? '');
		const subset: Layer = { ...layer, bindings: [binding] };
		const resolved = resolveBindings(subset, this.snapshotFor(binding.sourceAlias), this.#formatters);
		return String((resolved as unknown as Record<string, unknown>)[property] ?? '');
	}

	/** The LIVE source value of a bound property, ignoring any override — what
	 *  Restore would bring back (the ↺ tooltip's “Restore to …” value). */
	sourceValue(layer: Layer | undefined, property: string): string {
		if (!layer) return '';
		const binding = this.bindingFor(layer, property);
		if (!binding) return '';
		const stripped: Binding = { ...binding };
		delete (stripped as { override?: string }).override;
		const subset: Layer = { ...layer, bindings: [stripped] };
		const resolved = resolveBindings(subset, this.snapshotFor(binding.sourceAlias), this.#formatters);
		return String((resolved as unknown as Record<string, unknown>)[property] ?? '');
	}

	// ── Format options (S-FORMAT) — per-binding display knobs over the kit formatters ──

	/** The formatter descriptor backing a binding's `format` hint ('currency:THB' → the
	 *  'currency' descriptor), or undefined when the binding has no formatter (e.g. image
	 *  src) or the hint names an unregistered formatter. Drives the Inspector Format block. */
	formatDescriptorFor(binding: Binding | undefined): FormatterDescriptor | undefined {
		if (!binding?.format) return undefined;
		return this.#formatters[binding.format.split(':')[0]];
	}

	/** The effective per-knob options for a binding — the descriptor defaults merged with
	 *  the binding's saved `formatOptions` (override wins). Mirrors the resolver's merge, so
	 *  the Inspector's selects + previews show exactly what the canvas renders. */
	formatOptsFor(binding: Binding | undefined): FormatOpts {
		const descriptor = this.formatDescriptorFor(binding);
		return { ...(descriptor ? descriptorDefaults(descriptor) : {}), ...binding?.formatOptions };
	}

	/** The formatter argument — the part after ':' in the hint ('currency:THB' → 'THB',
	 *  'area:rai' → 'rai'). Passed to the descriptor's preview + result rendering. */
	formatArgFor(binding: Binding | undefined): string | undefined {
		return binding?.format?.split(':')[1];
	}

	/** Set one display knob on a bound property (the Inspector Format selects). Writes into
	 *  `binding.formatOptions[key]` and persists like any binding edit (one revision → auto-save);
	 *  no-op when the property is not bound. */
	setFormatOption(layerId: string, property: string, key: string, value: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		const binding = this.bindingFor(layer, property);
		if (!binding) return;
		this.commit(
			bindLayer(this.state, layerId, {
				...binding,
				formatOptions: { ...binding.formatOptions, [key]: value }
			})
		);
	}

	/** Link a layer property to an Active-source field (link / re-link — copies format + fallback). */
	bindField(layerId: string, property: string, field: FieldDescriptor, sourceAlias?: string): void {
		// Default = the Active 'primary' source; built-in field chips (e.g. Branding's
		// "Company name") pass their own alias explicitly (#0286).
		const alias = sourceAlias ?? this.activeSource?.alias;
		if (!alias) return;
		const binding: Binding = {
			property,
			sourceAlias: alias,
			token: field.token,
			...(field.format ? { format: field.format } : {}),
			...(field.fallback ? { fallback: field.fallback } : {})
		};
		this.commit(bindLayer(this.state, layerId, binding));
	}

	/** Bind an image layer's `src` to a positional snapshot image token
	 *  (`images.<group>.<index>.url`) — the Media-panel thumbnail bind. Unlike `bindField`,
	 *  the token is NOT a declared field: gallery images are positional, so the link targets
	 *  a slot, not a named value. `placeholder: true` marks the binding template-grade — it
	 *  re-resolves when the document is spawned from a template / the source is swapped (kit
	 *  Binding semantics). NOTE: the tolerant empty-slot behaviour is NOT driven by this flag
	 *  — `resolveBindings` ignores `placeholder`; an out-of-range token simply resolves to the
	 *  binding fallback (''), so the layer renders empty until the source gains that image
	 *  (the visible placeholder is a render-path concern — the next slice). */
	bindImageToken(layerId: string, token: string, alias?: string): void {
		// Bind to the given source alias (the Media panel's [Sources ▾] selection, G6b);
		// defaults to the Active 'primary' source when omitted.
		const a = alias ?? this.activeSource?.alias;
		if (!a) return;
		const binding: Binding = { property: 'src', sourceAlias: a, token, placeholder: true };
		this.commit(bindLayer(this.state, layerId, binding));
	}

	/** Create a NEW image layer CENTRED on (x, y) artboard px and bind it to the token on the
	 *  given alias — the drag-to-canvas place path (G8; since D-1 any drop NOT on an image
	 *  layer lands here). Composes the kit image factory + addLayer + bindImageToken; selects
	 *  the new layer so it's immediately the active target. */
	placeImageLayer(token: string, x: number, y: number, alias?: string): void {
		const a = alias ?? this.activeSource?.alias;
		if (!a || !this.activePage) return; // need a source alias AND a page to place onto
		// Materialise at origin to read the factory size, then centre on the drop point.
		const seed = presetToLayer({ id: 'image', baseType: 'image' }, 0, 0, this.layerDefaults);
		if (!seed) return;
		const px = Math.round(x - (seed.width ?? 0) / 2);
		const py = Math.round(y - (seed.height ?? 0) / 2);
		const layer = presetToLayer({ id: 'image', baseType: 'image' }, px, py, this.layerDefaults);
		if (!layer) return;
		this.add(layer);
		this.bindImageToken(layer.id, token, a);
		this.select(layer.id);
	}

	/** Create a NEW text layer CENTRED on (x, y) artboard px and bind its content to the
	 *  field — the field-chip drag-to-canvas path (D-1). Mirrors placeImageLayer: kit text
	 *  factory (body seed) + addLayer + bindField; the layer is NAMED after the field so
	 *  the Layers panel reads "Price", not the seed copy; selects it so it's immediately
	 *  the active target. */
	placeTextLayer(field: FieldDescriptor, x: number, y: number, alias?: string): void {
		const a = alias ?? this.activeSource?.alias;
		if (!a || !this.activePage) return; // need a source alias AND a page to place onto
		// Materialise at origin to read the factory size, then centre on the drop point.
		const seed = presetToLayer({ id: 'text.body', baseType: 'text' }, 0, 0, this.layerDefaults);
		if (!seed) return;
		const px = Math.round(x - (seed.width ?? 0) / 2);
		const py = Math.round(y - (seed.height ?? 0) / 2);
		const layer = presetToLayer({ id: 'text.body', baseType: 'text' }, px, py, this.layerDefaults);
		if (!layer) return;
		this.add({ ...layer, name: field.label });
		this.bindField(layer.id, 'content', field, a);
		this.select(layer.id);
	}

	/** If `layer` is an annotation (marker/outline/callout), anchor it to a map surface so it
	 *  bakes/projects. Target = the map under (cx,cy); if none, the single/nearest non-system map
	 *  anchored at its CENTRE (always in-frame — R5 floor). No map at all → returned unchanged
	 *  (the R4-gate prevents inserting annotations without a map). */
	anchorIfAnnotation(layer: Layer, cx: number, cy: number): Layer {
		if (layer.type !== 'marker' && layer.type !== 'outline' && layer.type !== 'callout') return layer;
		// Pure kit helper picks the target map + anchor point (the map under the point,
		// else the nearest map's centre); null = no anchorable map on the page (the
		// R4-gate prevents inserting annotations without one).
		const target = resolveAnchorTarget(this.layers, cx, cy);
		if (!target) return layer;
		return anchorAnnotationToMap(layer, target.x, target.y, target.map);
	}

	/** Default map_config for a newly-placed type:'map' layer — the PRIMARY source's geo
	 *  (pin + saved zoom) so the map opens on the document's property, not the neutral
	 *  factory fallback (B7). Null when the primary source declares no geo → caller keeps
	 *  the factory default. */
	mapDefaultConfig(): MapConfig | null {
		const geo = this.snapshotFor('primary').geo as
			| { latitude?: number; longitude?: number; zoom?: number }
			| undefined;
		if (!geo || typeof geo.latitude !== 'number' || typeof geo.longitude !== 'number') return null;
		return {
			center: { lat: geo.latitude, lng: geo.longitude },
			zoom: typeof geo.zoom === 'number' ? Math.round(geo.zoom) : 15,
			mapType: 'roadmap'
		};
	}

	/** Unlink — drop the binding, keeping the CURRENT resolved value as the new static value. */
	unbind(layerId: string, property: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		const value = this.resolvedValue(layer, property);
		this.commit(unbindLayer(this.state, layerId, property, value));
	}

	/** Set a manual override (fine-tune) on a bound property. */
	setOverride(layerId: string, property: string, value: string): void {
		this.commit(setBindingOverride(this.state, layerId, property, value));
	}

	/** Restore — clear the override, reverting to the live source value. */
	restore(layerId: string, property: string): void {
		this.commit(clearBindingOverride(this.state, layerId, property));
	}

	// ── Text templates (INC-C) — the content field's third kind. The composer authors a
	//    Template (text + token nodes); these adapters bridge the editor to the
	//    @sbx/text-template protocols (consumer-owned resolution/catalog) + persist the result.

	/** Resolve ONE template token against the ACTIVE source — the composer's live badge value.
	 *  `format` defaults to the field's declared format (so [property.price] shows ฿…). */
	resolveToken(token: string, format?: string, formatOptions?: Record<string, string>): string {
		// A source-qualified token ('<alias>.<field>') resolves against THAT source; a bare/legacy
		// token against 'primary' — the SAME default the export + resolvedValue paths use, so a
		// badge keeps the exact source it was inserted from, with no live/export divergence.
		const known = new Set(this.sources.map((s) => s.alias));
		const { alias, field } = splitSourceToken(token, known, 'primary');
		if (!alias) return '';
		const fmt = format ?? this.fieldsForAlias(alias, 'text').find((f) => f.token === field)?.format;
		const binding: Binding = {
			property: 'content',
			sourceAlias: alias,
			token: field,
			...(fmt ? { format: fmt } : {}),
			...(formatOptions ? { formatOptions } : {})
		};
		const synthetic = { bindings: [binding] } as unknown as Layer;
		const resolved = resolveBindings(synthetic, this.snapshotFor(alias), this.#formatters);
		return String((resolved as unknown as Record<string, unknown>).content ?? '');
	}

	/** The insertable tokens for the composer's picker — the active source's bindable text fields
	 *  (TokenCatalog protocol). Each carries the field's label + declared format. */
	tokenCatalog(): TokenCatalog {
		// EVERY attached source's text fields — each token source-QUALIFIED ('<alias>.<field>') and
		// tagged with a `group` (the source label) so the composer's @-menu / reattach picker can
		// offer source → field. A source with no text fields drops out.
		const tokens: TemplateToken[] = [];
		for (const s of this.sourceSummaries()) {
			const fields = this.fieldsForAlias(s.alias, 'text');
			if (fields.length === 0) continue;
			const group = `${s.alias} · ${s.title}`;
			for (const f of fields) {
				tokens.push({ token: `${s.alias}.${f.token}`, label: f.label, group, ...(f.format ? { format: f.format } : {}) });
			}
		}
		return { tokens: () => tokens };
	}

	/** TokenResolver protocol over the active source — what the composer renders each badge with. */
	tokenResolver(): TokenResolver {
		return { resolve: (token, format, formatOptions) => this.resolveToken(token, format, formatOptions) };
	}

	/** Persist the composer's edited template onto a layer's content. Template + a content binding
	 *  are mutually exclusive, so a content binding is dropped. An empty template clears it (the
	 *  field falls back to plain text). */
	setContentTemplate(layerId: string, template: Template): void {
		const layer = this.layers.find((l) => l.id === layerId);
		if (!layer) return;
		const rest = (layer.bindings ?? []).filter((b) => b.property !== 'content');
		this.update(layerId, {
			content_template: template.length ? template : undefined,
			bindings: rest.length ? rest : undefined
		});
	}

	/** Convert a content field to a TEMPLATE: seed from the current binding (one token) or parse
	 *  the static text (picking up any [tokens] already typed). The field becomes a 'template' kind. */
	useTemplate(layerId: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		if (!layer) return;
		const binding = this.bindingFor(layer, 'content');
		const template: Template = binding
			? [
					{
						type: 'token',
						// Qualify with the binding's source so the template badge resolves against the
						// SAME record the binding did (not whatever happens to be active later).
						token: `${binding.sourceAlias}.${binding.token}`,
						...(binding.format ? { format: binding.format } : {}),
						...(binding.formatOptions ? { formatOptions: binding.formatOptions } : {})
					}
				]
			: parseTemplate(layer.content ?? '');
		this.setContentTemplate(layerId, template);
	}

	/** Convert a TEMPLATE field back to plain text, keeping the currently resolved string as the
	 *  static value (so nothing is lost). */
	templateToText(layerId: string): void {
		const layer = this.layers.find((l) => l.id === layerId);
		if (!layer) return;
		const content = this.resolvedValue(layer, 'content');
		this.update(layerId, { content_template: undefined, content });
	}
}

/** Re-exported for components that need to look up a page by id. */
export { findPage };
