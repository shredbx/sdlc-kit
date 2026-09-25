<script lang="ts">
	// Property inspector — Xcode-style categorized, compact layout (2026-06-08).
	// Rows are HORIZONTAL: a right-aligned label gutter + the control, with the
	// per-row mode chips (🏷 link / ≣ animate) trailing. Controls are grouped into
	// collapsible CATEGORIES (reusing CollapsibleSection in its `dense` variant) so
	// the long property list packs into a few labelled, foldable sections.
	//   🏷 link (slice 4a) — LIVE on bindable rows (text/callout CONTENT, image
	//      SOURCE). When bound the row expands into Xcode-style LINK rows below it
	//      (Source ▾ / Field ▾ + reveal/unlink) — the labelled-row treatment.
	//   ≣ animate — LIVE on animatable rows; toggling it makes the value change over
	//      time (an inline From→To · Start · Duration · easing editor renders directly
	//      under the field — design pass C; there is no separate ANIMATION section).
	// <Field> owns the error (the inner control never renders its own).
	import { Button, Field, Input, Select, Textarea, Icon, FontPicker } from '@sbx/core-ui/components/primitives';
	import Modal from '@sbx/core-ui/components/primitives/Modal.svelte';
	import {
		EMPTY_BRAND_KIT,
		easingPreset,
		easingToPresetName,
		formatKnobChoices,
		fieldKindFor,
		DEFAULT_PREVIEW_BACKDROP,
		type BrandKit,
		type Layer,
		type TileEffect,
		type AnimatableProperty,
		type AnimationTrack,
		type AnimationSpec,
		type EasingPresetName,
		type Binding,
		type MapType,
		type MapConfig,
		type PreviewBackdropKind
	} from '@sbx/canvas-kit';
	import type { CanvasEditor, ImageSourceView } from './editor-state.svelte.js';
	import CollapsibleSection from './CollapsibleSection.svelte';
	import ImageSourcePicker from './ImageSourcePicker.svelte';
	import TemplateComposer from './TemplateComposer.svelte';
	import ColorField from './ColorField.svelte';
	import SegmentedControl from './SegmentedControl.svelte';
	import ScrubUnit from './ScrubUnit.svelte';
	import EasingCurvePreview from './EasingCurvePreview.svelte';

	interface Props {
		layer?: Layer;
		/** Consumer brand entries (Decision #0286 Phase A) — color-row swatches +
		 *  the FONT select options. Default = no brand entries (free entry only). */
		brandKit?: BrandKit;
		/** The reactive editor — drives the binding affordances (sources/fields/bind ops). */
		editor?: CanvasEditor;
		onupdate?: (patch: Partial<Layer>) => void;
		ontogglevisibility?: () => void;
		ondelete?: () => void;
		/** Animate a NOT-yet-animated property (≣) — opens the AddAnimationDialog. */
		onanimate?: (property: AnimatableProperty) => void;
		/** Edit an ALREADY-animated property (≣) — opens the info/edit popover.
		 *  Routing here (not onanimate) avoids re-adding, which would REPLACE the track. */
		oneditanimate?: (property: AnimatableProperty) => void;
		/** Open the Layers panel on the left rail (D-4 — the ⧉ reveal next to the
		 *  layer switcher); the panel highlights + reveals the current selection. */
		onopenlayers?: () => void;
		/** Open a left-rail panel where a binding's item lives (E-3 — the binding row's
		 *  reveal); the panel highlights + scrolls to the bound item (E-2). */
		onreveal?: (context: 'media' | 'map' | 'text') => void;
		/** Read the current marketing alias for an enum value — e.g. getEnumAlias('transactionType','sale')
		 *  → 'FOR SALE'. Returns '' when no alias is set (formatter falls back to humanized code). */
		getEnumAlias?: (set: string, code: string) => string;
		/** Persist a new alias for an enum value. Called from the alias editor on every keystroke. */
		onaliaschange?: (set: string, code: string, alias: string) => void;
		/** Google Fonts catalogue (#3) for the FontPicker — filtered/paginated in the picker.
		 *  Empty → the picker shows system/brand fonts only. */
		fontCatalogue?: { family: string; category?: string }[];
		/** Whether the per-property animate chips (≣) render (Decision #0298). Default
		 *  true = the existing Media Canvas. false hides every animate chip (a static
		 *  mode — watermark — has no motion to author). */
		showAnimate?: boolean;
		/** Whether the Watermark inspector block renders at the TOP of the Inspector
		 *  (Decision #0298). Default false. In watermark mode this is true; the params UI
		 *  itself lands in a later slice — the section is a gated placeholder for now. */
		showWatermarkBlock?: boolean;
	}

	let { layer, brandKit = EMPTY_BRAND_KIT, editor, onupdate, ontogglevisibility, ondelete, onanimate, oneditanimate, onopenlayers, onreveal, getEnumAlias, onaliaschange, fontCatalogue = [], showAnimate = true, showWatermarkBlock = false }: Props = $props();

	// Brand/system fonts for the FontPicker — deduped by family, label = the brand name. */
	const brandFontList = $derived.by(() => {
		const seen = new Set<string>();
		const out: { family: string; label?: string }[] = [];
		for (const f of brandKit.fonts) {
			if (seen.has(f.family)) continue;
			seen.add(f.family);
			out.push({ family: f.family, label: f.name });
		}
		return out;
	});

	/** Layer-switcher options (D-4) — every layer of the active page, front-most
	 *  first (the LayersPanel order), labelled by name. Quick selection switching
	 *  without leaving the Inspector; works from the no-selection state too. */
	const layerOptions = $derived(
		[...(editor?.layers ?? [])].reverse().map((l) => ({ value: l.id, label: l.name }))
	);

	// Which property's field picker is open (null = none). Reset when the selection changes —
	// along with any pending unlink confirm, which would otherwise re-resolve against the NEW
	// layer's binding on the same property (wrong-layer unlink). The animation row expansion +
	// add-menu reset too, so they never carry over to a different layer's tracks.
	let bindPickerFor = $state<string | null>(null);
	let lastLayerId: string | undefined;
	$effect(() => {
		const id = layer?.id;
		if (id !== lastLayerId) {
			lastLayerId = id;
			bindPickerFor = null;
			unlinkFor = null;
			expandedAnims = new Set();
		}
	});

	/** Human label for a bound token (falls back to the raw token). Positional image tokens
	 *  (images.<group>.<index>.url, bound via the grid popup — G6c) read the way THEIR
	 *  panel names them: map-role groups by the image alt ("Location map" — the Map tab's
	 *  card labels), photo groups as "Cover 1" / "Gallery 3" from the category label (the
	 *  Media grid's labels) — never the raw dot-path. Field tokens resolve against the
	 *  BINDING's own source (via its alias — built-ins included) first: two kinds may
	 *  declare the same token under different labels (property 'title' = "Name", branding
	 *  'title' = "Company name"), so the active provider is only the fallback. */
	function fieldLabel(token: string, alias?: string): string {
		const m = /^images\.([^.]+)\.(\d+)\.url$/.exec(token);
		if (m) {
			const src = editor?.imageSources().find((s) => s.alias === alias);
			const cat =
				src?.categories.find((c) => c.id === m[1]) ??
				editor
					?.imageSources()
					.flatMap((s) => s.categories)
					.find((c) => c.id === m[1]);
			if (cat?.role === 'map') {
				const alt = src?.images[m[1]]?.[Number(m[2])]?.alt;
				if (alt) return alt;
			}
			const label = cat?.label ?? m[1].charAt(0).toUpperCase() + m[1].slice(1);
			return `${label} ${Number(m[2]) + 1}`;
		}
		if (alias && editor) {
			const ref = editor.sources.find((s) => s.alias === alias);
			const owned = ref && editor.providerFor(ref.kind)?.fields().find((f) => f.token === token)?.label;
			if (owned) return owned;
		}
		return editor?.sourceFields().find((f) => f.token === token)?.label ?? token;
	}

	/** Bind a layer property to the active source field with the given token (link / re-link). */
	function bindByToken(property: string, kind: 'text' | 'image', token: string): void {
		if (!editor || !layer || !token) return;
		const field = editor.fieldsFor(kind).find((f) => f.token === token);
		if (field) editor.bindField(layer.id, property, field);
	}

	/** Re-link a bound row to another field of the BINDING's own source — the [item ▾]
	 *  select on text rows (one click, Canva-style). The alias is pinned: picking an
	 *  item never moves the binding off the source the row shows (`bindByToken` would
	 *  re-anchor it to the Active source). Same-token picks are no-ops. */
	function rebindToken(property: string, token: string): void {
		if (!editor || !layer || !token) return;
		const binding = editor.bindingFor(layer, property);
		if (!binding || binding.token === token) return;
		const field = editor.fieldsForAlias(binding.sourceAlias, 'text').find((f) => f.token === token);
		if (field) editor.bindField(layer.id, property, field, binding.sourceAlias);
	}

	// --- IB-style binding row (E-3) -----------------------------------------

	/** The functional role of a positional image token's group on its source —
	 *  distinguishes a derived map render from photographic imagery (kind glyph,
	 *  popup tab seed, reveal target). Named tokens have no group → undefined. */
	function groupRole(token: string, alias?: string): 'map' | undefined {
		const m = /^images\.([^.]+)\.\d+\.url$/.exec(token);
		if (!m) return undefined;
		return editor
			?.imageSources()
			.find((s) => s.alias === alias)
			?.categories.find((c) => c.id === m[1])?.role;
	}

	/** Left-rail context where the binding's item lives — the row's reveal target. */
	function revealContextFor(binding: Binding, kind: 'text' | 'image'): 'media' | 'map' | 'text' {
		if (kind === 'text') return 'text';
		return groupRole(binding.token, binding.sourceAlias) === 'map' ? 'map' : 'media';
	}

	/** Glyph for the bound item's kind — image · map component · text field. */
	function itemGlyph(binding: Binding, kind: 'text' | 'image'): string {
		if (kind === 'text') return 'book-text';
		return groupRole(binding.token, binding.sourceAlias) === 'map' ? 'map' : 'image';
	}

	/** Open/close the item picker for a property (the row's item button AND the 🏷
	 *  mode toggle both route here). */
	function togglePicker(property: string): void {
		bindPickerFor = bindPickerFor === property ? null : property;
	}

	// Property pending unlink — the row's compact ✕ asks before unbinding (the layer
	// keeps the resolved value as its static one, so the loss is the LINK, not the look).
	let unlinkFor = $state<string | null>(null);

	/** Source-dropdown options for a binding kind — only sources that can serve it
	 *  (image groups for src, declared fields for content). */
	function sourceOptionsFor(kind: 'text' | 'image'): { value: string; label: string }[] {
		if (!editor) return [];
		if (kind === 'image') {
			return editor.imageSources().map((s) => ({ value: s.alias, label: `${s.alias} · ${s.title}` }));
		}
		return editor
			.sourceSummaries()
			.filter((s) => (editor?.providerFor(s.kind)?.fields().length ?? 0) > 0)
			.map((s) => ({ value: s.alias, label: `${s.alias} · ${s.title}` }));
	}

	/** Image-source views narrowed to photographic OR map groups (the popup tabs);
	 *  sources left without groups for the view drop out. */
	function narrowSources(view: 'images' | 'maps'): ImageSourceView[] {
		return (editor?.imageSources() ?? [])
			.map((s) => ({
				...s,
				categories: s.categories.filter((c) => (c.role === 'map') === (view === 'maps'))
			}))
			.filter((s) => s.categories.length > 0);
	}

	// FORMAT [B][I] + ALIGN segments (design pass A, 2026-06-07) — text-editor-style
	// icon controls replacing the old dropdowns. font_weight is a free model string
	// ('normal'|'bold'|'100'-'900'): B reads PRESSED for 'bold' AND numeric ≥600
	// (the canvas renders those bold-ish — the toggle state matches what you see);
	// flipping normalises to 'bold'/'normal'. B and I combine freely (independent
	// toggles), ALIGN is single-select.
	const weightValue = $derived.by(() => {
		const w = layer?.font_weight ?? 'normal';
		if (w === 'bold') return 'bold';
		const n = Number(w);
		return Number.isFinite(n) && n >= 600 ? 'bold' : 'normal';
	});
	const styleValue = $derived(layer?.font_style === 'italic' ? 'italic' : 'normal');

	const ALIGN_SEGMENTS = [
		{ value: 'left', icon: 'align-left', title: 'Align left' },
		{ value: 'center', icon: 'align-center', title: 'Align center' },
		{ value: 'right', icon: 'align-right', title: 'Align right' }
	];

	// Text-fit segments (S-FIT) — how content behaves vs its box, as icon tabs:
	//   none = draw at the set size (may overflow) · fit = SHRINK to fit (never grows) ·
	//   fill = scale UP+down to fill the box · crop = clip to box · truncate = single-line …
	const TEXT_FIT_SEGMENTS = [
		{ value: 'none', icon: 'ban', title: 'None (may overflow)' },
		{ value: 'fit', icon: 'minimize-2', title: 'Shrink to fit' },
		{ value: 'fill', icon: 'maximize-2', title: 'Scale to fill' },
		{ value: 'crop', icon: 'crop', title: 'Crop to box' },
		{ value: 'truncate', icon: 'ellipsis', title: 'Truncate (…)' }
	];

	// Field-mode segments (#4) — text / linked / template as icon tabs, replacing the
	// caption links + the 🏷 bind chip. Single-select; selecting converts the field kind.
	const FIELD_MODE_SEGMENTS = [
		{ value: 'text', icon: 'type', title: 'Plain text — edit directly' },
		{ value: 'linked', icon: 'link', title: 'Linked to a source field' },
		{ value: 'template', icon: 'braces', title: 'Template — text + inserted fields' }
	];

	/** Switch the content field's kind (#4 tabs). text ⇆ linked ⇆ template, keeping the value:
	 *   → text     unlink (linked) or flatten (template) to the resolved string
	 *   → template seed a template from the binding/text
	 *   → linked   flatten any template first, then open the field picker (the link() row) */
	function setContentKind(kind: 'text' | 'linked' | 'template'): void {
		if (!editor || !layer) return;
		const cur = fieldKindFor(layer, 'content');
		if (kind === cur) return;
		if (kind === 'text') {
			if (cur === 'template') editor.templateToText(layer.id);
			else if (cur === 'linked') editor.unbind(layer.id, 'content');
		} else if (kind === 'template') {
			editor.useTemplate(layer.id);
		} else {
			// → linked: a template can't also be bound, so flatten it first, then reveal the picker.
			if (cur === 'template') editor.templateToText(layer.id);
			bindPickerFor = 'content';
		}
	}

	function toggleFormat(which: string): void {
		if (which === 'bold') {
			onupdate?.({ font_weight: weightValue === 'bold' ? 'normal' : 'bold' });
		} else if (which === 'italic') {
			onupdate?.({ font_style: styleValue === 'italic' ? 'normal' : 'italic' });
		}
	}


	function patchNumber(key: keyof Layer, raw: string): void {
		const n = Number(raw);
		if (Number.isFinite(n)) onupdate?.({ [key]: n } as Partial<Layer>);
	}

	// --- Preview backdrop (DOCUMENT-level, PREVIEW-ONLY — #0298) ----------------
	// A colour OR a picked canvas image the author previews the transparent mark over.
	// Rendered by CanvasStage as stage chrome (never a layer / never exported). The kind
	// Select switches the face; the colour reuses ColorField; the image reuses the SAME
	// ImageSourcePicker the Media panel / image-bind popup use (no new uploader).
	const PB_KIND_OPTS: { value: PreviewBackdropKind; label: string }[] = [
		{ value: 'none', label: 'None' },
		{ value: 'color', label: 'Colour' },
		{ value: 'image', label: 'Image' }
	];

	/** Resolve an `images.<group>.<index>.url` token (the ImageSourcePicker's emit) to its
	 *  actual URL via the live snapshots, so the backdrop stores a concrete src the stage
	 *  routes through its proxy. Returns '' when the slot has no image yet. */
	function resolveImageToken(token: string, alias?: string): string {
		const m = /^images\.([^.]+)\.(\d+)\.url$/.exec(token);
		if (!m) return '';
		const src = editor?.imageSources().find((s) => s.alias === alias);
		const img = src?.images[m[1]]?.[Number(m[2])];
		return img?.url ?? '';
	}

	/** All attached photographic (non-map) image sources for the backdrop picker — the
	 *  SAME narrowing the image-bind popup uses. Empty → the picker shows its empty state. */
	const backdropImageSources = $derived(narrowSources('images'));

	// Crop edges clamp at 0 — a negative inset has no meaning (the clip window
	// never grows past the box).
	function patchCrop(key: 'crop_top' | 'crop_right' | 'crop_bottom' | 'crop_left', raw: string): void {
		const n = Number(raw);
		if (Number.isFinite(n)) onupdate?.({ [key]: Math.max(0, n) } as Partial<Layer>);
	}

	const cropSet = $derived(
		!!layer &&
			((layer.crop_top ?? 0) > 0 ||
				(layer.crop_right ?? 0) > 0 ||
				(layer.crop_bottom ?? 0) > 0 ||
				(layer.crop_left ?? 0) > 0)
	);

	// Map config is nested — merge the patch over the existing config so a single
	// knob (zoom / mapType) never drops the others (no-op without a map layer/config).
	function patchMapConfig(patch: Partial<MapConfig>): void {
		if (layer?.type !== 'map' || !layer.map_config) return;
		onupdate?.({ map_config: { ...layer.map_config, ...patch } } as Partial<Layer>);
	}

	// Stroke thickness lives inside the StrokeStyle object — merge it in place.
	function patchStrokeThickness(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		const base = layer.stroke ?? { color: '#000000', thickness: 1, dash_type: 'solid', glow_intensity: 0 };
		onupdate?.({ stroke: { ...base, thickness: Math.max(0, n) } });
	}

	// --- Effects (user-approved 2026-06-07): shadow / border glow / text glow +
	// shadow. First touch of any knob creates the FULL default object; the per-knob
	// inputs then merge into it (clear = the row's ✕). Defaults per the discussion:
	// shadow blur 8 / offset (0,4) / opacity 0.4; glow intensity 5.
	const DEFAULT_SHADOW = { color: '#000000', blur: 8, offset_x: 0, offset_y: 4, opacity: 0.4 };
	const DEFAULT_TEXT_GLOW = { intensity: 5, color: '#ffffff' };

	/** Border glow intensity (0–10) — merges into the EXISTING stroke (the GLOW
	 *  row only renders when a border is set). */
	function patchStrokeGlow(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer?.stroke) return;
		onupdate?.({ stroke: { ...layer.stroke, glow_intensity: Math.min(10, Math.max(0, n)) } });
	}

	/** Merge one numeric knob into the box shadow (creates it on first touch). */
	function patchShadow(key: 'blur' | 'offset_x' | 'offset_y', raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ shadow: { ...(layer.shadow ?? DEFAULT_SHADOW), [key]: n } });
	}

	function patchShadowOpacity(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ shadow: { ...(layer.shadow ?? DEFAULT_SHADOW), opacity: Math.min(1, Math.max(0, n / 100)) } });
	}

	/** Merge one numeric knob into the glyph shadow (creates it on first touch). */
	function patchTextShadow(key: 'blur' | 'offset_x' | 'offset_y', raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ text_shadow: { ...(layer.text_shadow ?? DEFAULT_SHADOW), [key]: n } });
	}

	function patchTextShadowOpacity(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ text_shadow: { ...(layer.text_shadow ?? DEFAULT_SHADOW), opacity: Math.min(1, Math.max(0, n / 100)) } });
	}

	function patchTextGlowIntensity(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ text_glow: { ...(layer.text_glow ?? DEFAULT_TEXT_GLOW), intensity: Math.min(10, Math.max(0, n)) } });
	}

	// --- Image legibility effects (#0300) — a TINT (recolor the alpha silhouette) + a centred
	// GLOW halo, so ONE logo asset reads on any background. Opacity already lives in ARRANGE.
	// The glow is the sanctioned image effect (images take a glow, never an offset box shadow —
	// user call 2026-06-07). Default halo: soft dark (the subtitle / lower-third contrast cue).
	const DEFAULT_IMAGE_GLOW = { intensity: 6, color: '#000000' };
	const DEFAULT_IMAGE_TINT = { color: '#000000', strength: 1 };

	function patchImageGlowIntensity(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ glow: { ...(layer.glow ?? DEFAULT_IMAGE_GLOW), intensity: Math.min(10, Math.max(0, n)) } });
	}

	function patchTintStrength(raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n) || !layer) return;
		onupdate?.({ tint: { ...(layer.tint ?? DEFAULT_IMAGE_TINT), strength: Math.min(1, Math.max(0, n / 100)) } });
	}

	/** One-click "Contrast halo" — the proven any-background config: a soft dark glow halo
	 *  (subtitle / broadcast lower-third dual-contrast). Stays fully editable afterwards.
	 *  #0300: GUARANTEED per-photo legibility is the apply-time auto theme, not this static cue. */
	function applyContrastHalo(): void {
		onupdate?.({ glow: { ...DEFAULT_IMAGE_GLOW } });
	}

	// 0–1 opacity edited as 0–100 percent.
	const opacityPct = $derived(layer ? Math.round((layer.opacity ?? 1) * 100) : 100);
	function patchOpacity(raw: string): void {
		const n = Number(raw);
		if (Number.isFinite(n)) onupdate?.({ opacity: Math.min(1, Math.max(0, n / 100)) });
	}

	// --- [WM] Tiling effect (per-element/group, generic) ---------------------
	// Seeded gap = the source's own size, so the FIRST toggle lays a clean one-gap
	// lattice (copies spaced a full element apart) the author can then tune — never a
	// collapsed clump or a silent angle. Rotation stays the layer's own (default 0).
	const tile = $derived(layer?.tile);
	const tileOn = $derived(!!tile && (tile.repeatH || tile.repeatV));
	/** Default gaps from the live element size (square-ish fallback when unknown). */
	function defaultTile(): TileEffect {
		const w = layer?.width ?? 80;
		const h = layer?.height ?? 80;
		return { repeatH: false, repeatV: false, gapX: Math.round(w), gapY: Math.round(h) };
	}
	/** Merge a patch over the current tile (seeding from defaults on first edit). */
	function patchTile(patch: Partial<TileEffect>): void {
		if (!layer) return;
		onupdate?.({ tile: { ...(layer.tile ?? defaultTile()), ...patch } });
	}
	/** Flip one repeat axis. Toggling the LAST axis off clears the effect entirely so a
	 *  non-tiled element carries no dormant tile in its serialized doc. */
	function toggleRepeat(axis: 'repeatH' | 'repeatV'): void {
		if (!layer) return;
		const base = layer.tile ?? defaultTile();
		const next = { ...base, [axis]: !base[axis] };
		onupdate?.({ tile: next.repeatH || next.repeatV ? next : undefined });
	}
	/** Patch a gap from a raw input string (floored at 0 — a negative pitch is meaningless). */
	function patchTileGap(key: 'gapX' | 'gapY', raw: string): void {
		const n = Number(raw);
		if (Number.isFinite(n)) patchTile({ [key]: Math.max(0, Math.round(n)) });
	}

	// Is this animatable property already animated on the selected layer? Drives the
	// ≣ button's active styling + aria-pressed (reads the layer's own tracks).
	function isAnimated(property: AnimatableProperty): boolean {
		return !!layer?.animations?.some((t) => t.property === property);
	}

	// --- Over-time value editor (design pass C, 2026-06-08) ---------------------
	// A value that goes from its start value to its end value over [start..start+dur]
	// IS the "animation". animFor(prop) reads a track's summary (from/to/timing/easing)
	// back for the inline editor that renders under each animated field. All edits
	// write through editor.updateAnimation — the same reducer the timeline uses.
	interface AnimSummary {
		property: AnimatableProperty;
		label: string;
		from: number;
		to: number;
		startMs: number;
		durMs: number;
		preset: EasingPresetName;
	}

	const ANIM_LABELS: Record<AnimatableProperty, string> = {
		x: 'X',
		y: 'Y',
		width: 'Width',
		height: 'Height',
		opacity: 'Opacity',
		rotation: 'Rotation',
		scale: 'Scale',
		crop_top: 'Crop Top',
		crop_right: 'Crop Right',
		crop_bottom: 'Crop Bottom',
		crop_left: 'Crop Left'
	};

	/** Read a track's 2-keyframe authoring summary back for the row + inline editor
	 *  (start value, end value, segment timing, easing preset). */
	function animSummary(track: AnimationTrack): AnimSummary {
		const kfs = [...track.keyframes].sort((a, b) => a.time - b.time);
		const first = kfs[0];
		const last = kfs[kfs.length - 1];
		const prop = track.property as AnimatableProperty;
		return {
			property: prop,
			label: ANIM_LABELS[prop] ?? track.property,
			from: Number(first?.value ?? 0),
			to: Number(last?.value ?? 0),
			startMs: first?.time ?? 0,
			durMs: (last?.time ?? 0) - (first?.time ?? 0),
			preset: easingToPresetName(first?.easing ?? 'linear')
		};
	}

	const animations = $derived((layer?.animations ?? []).map(animSummary));

	/** The over-time summary for a property, or undefined when it has none. */
	function animFor(property: AnimatableProperty): AnimSummary | undefined {
		return animations.find((a) => a.property === property);
	}

	/** Scrub-badge unit for a value's From/To (raw property units — px for geometry,
	 *  ° for rotation, unitless 0–1 for opacity/scale). */
	function animUnit(property: AnimatableProperty): string {
		if (property === 'rotation') return '°';
		if (property === 'opacity' || property === 'scale') return '';
		return 'px';
	}

	const EASING_OPTS: { value: EasingPresetName; label: string }[] = [
		{ value: 'linear', label: 'Linear' },
		{ value: 'in', label: 'Ease in' },
		{ value: 'out', label: 'Ease out' },
		{ value: 'in-out', label: 'Ease in-out' }
	];

	// Which fields' inline over-time editors are expanded (a value can change over
	// time = "animation"; multiple fields may be live at once, hence a Set).
	let expandedAnims = $state(new Set<string>());

	function toggleAnimExpand(property: string): void {
		const next = new Set(expandedAnims);
		next.has(property) ? next.delete(property) : next.add(property);
		expandedAnims = next;
	}

	/** Enable a field's over-time value — a value that starts at its current value and
	 *  changes to its end value over [start .. start+duration]. From=To at first (no
	 *  visible change until the user edits To); the inline editor opens to edit it. */
	function enableAnim(property: AnimatableProperty): void {
		if (!layer || !editor) return;
		const v = editor.baseValueFor(layer, property);
		editor.addAnimation(layer.id, { property, from: v, to: v, startMs: 0, durMs: 1000, easing: 'linear' });
		const next = new Set(expandedAnims);
		next.add(property);
		expandedAnims = next;
	}

	/** Patch one field of a value's over-time spec through the editor reducer. */
	function patchAnim(property: AnimatableProperty, patch: Partial<Omit<AnimationSpec, 'property'>>): void {
		if (layer) editor?.updateAnimation(layer.id, property, patch);
	}

	function patchAnimNumber(property: AnimatableProperty, key: 'from' | 'to' | 'startMs' | 'durMs', raw: string): void {
		const n = Number(raw);
		if (!Number.isFinite(n)) return;
		patchAnim(property, { [key]: key === 'startMs' || key === 'durMs' ? Math.max(0, n) : n });
	}

	function removeAnim(property: AnimatableProperty): void {
		if (!layer) return;
		editor?.removeAnimation(layer.id, property);
		if (expandedAnims.has(property)) {
			const next = new Set(expandedAnims);
			next.delete(property);
			expandedAnims = next;
		}
	}
</script>

<aside class="inspector">
	<!-- Shared numeric row — [label gutter] [Input + scrubbable unit suffix] [trailing slot].
	     Top-level so BOTH the document-level Watermark block (no selection) and the per-layer
	     `num` snippet reuse the exact same markup (no duplicated .frow CSS). The trailing slot
	     is optional: layer rows pass the link/animate `modes` chips; document rows pass none. -->
	{#snippet numRow(label: string, o: {
		id: string;
		name: string;
		value: number;
		oninput: (v: string) => void;
		onscrub: (n: number) => void;
		unit: string;
		aria: string;
		min?: number;
		max?: number;
		step?: number;
	}, trail?: import('svelte').Snippet)}
		<div class="frow">
			<span class="frow__label">{label}</span>
			<div class="frow__control">
				<Field for={o.id}>
					{#snippet children()}
						<Input id={o.id} name={o.name} type="number" size="sm" class="num" value={String(o.value)} oninput={o.oninput} aria-label={o.aria}>
							{#snippet right()}<ScrubUnit unit={o.unit} value={o.value} min={o.min} max={o.max} step={o.step ?? 1} onscrub={o.onscrub} />{/snippet}
						</Input>
					{/snippet}
				</Field>
			</div>
			<span class="frow__trail">{#if trail}{@render trail()}{/if}</span>
		</div>
	{/snippet}

	<!-- Preview backdrop (#0298) — DOCUMENT-level + PREVIEW-ONLY. The colour/image the author
	     previews the transparent mark over to judge legibility; rendered by CanvasStage as stage
	     chrome and EXCLUDED from the published overlay. It is NOT a per-element watermark property,
	     so it renders at the BOTTOM of the Inspector, collapsed, BELOW the element's own controls —
	     never above them. Kind switches the face: Colour → ColorField · Image → ImageSourcePicker. -->
	{#snippet previewBackdropBlock()}
		{#if showWatermarkBlock && editor}
			{@const pb = editor.previewBackdrop ?? DEFAULT_PREVIEW_BACKDROP()}
			<div class="inspector__watermark">
				<CollapsibleSection dense title="PREVIEW BACKDROP" open={false}>
					<p class="inspector__hint">Preview only — never part of the published mark.</p>
					<div class="frow">
						<span class="frow__label">Type</span>
						<div class="frow__control">
							<Select
								id="pb-kind"
								name="pb_kind"
								size="sm"
								value={pb.kind}
								options={PB_KIND_OPTS}
								onchange={(v) => v && editor.setPreviewBackdrop({ kind: v as PreviewBackdropKind })}
								aria-label="Preview backdrop type"
							/>
						</div>
						<span class="frow__trail"></span>
					</div>
					{#if pb.kind === 'color'}
						<div class="frow">
							<span class="frow__label">Colour</span>
							<div class="frow__control">
								<ColorField
									id="pb-color"
									name="pb_color"
									aria-label="Preview backdrop colour"
									value={pb.color ?? '#ffffff'}
									colors={brandKit.colors}
									onpick={(v) => editor.setPreviewBackdrop({ color: v })}
								/>
							</div>
							<span class="frow__trail"></span>
						</div>
					{:else if pb.kind === 'image'}
						{#if backdropImageSources.length > 0}
							<!-- Pick from the canvas's already-available images (the SAME grid as the
							     Media panel / image-bind popup). Selecting resolves the token → its URL
							     and stores it as the backdrop's imageSrc (the stage proxies it). -->
							<div class="bind__images">
								<ImageSourcePicker
									sources={backdropImageSources}
									canBind={true}
									boundToken={undefined}
									onbind={(token, alias) => {
										const url = resolveImageToken(token, alias);
										if (url) editor.setPreviewBackdrop({ imageSrc: url });
									}}
								/>
							</div>
						{:else}
							<p class="bind__empty">Attach an image source to preview the mark over a photo.</p>
						{/if}
					{/if}
				</CollapsibleSection>
			</div>
		{/if}
	{/snippet}

	<!-- The DOCUMENT-level, PREVIEW-ONLY backdrop block renders at the BOTTOM of the Inspector
	     (below the per-element controls), collapsed — it is a preview aid, never the published
	     mark, so it must never sit above the element's own watermark properties. See the
	     `previewBackdropBlock` snippet rendered just before </aside>. -->

	<!-- Layer switcher (D-4) — sits ABOVE the title, rendered in the empty state too:
	     the dropbox jumps between layers without touching the canvas; ⧉ reveals the
	     current layer in the Layers panel to continue working there. -->
	{#if layerOptions.length > 0}
		<div class="inspector__switcher">
			<div class="inspector__switcher-select">
				<Select
					id="insp-layer"
					name="layer_switcher"
					size="sm"
					value={layer?.id ?? ''}
					options={layerOptions}
					placeholder="Select a layer…"
					onchange={(id) => id && editor?.select(id)}
					aria-label="Active layer"
				/>
			</div>
			<button
				class="inspector__icon-btn"
				type="button"
				aria-label="Show in Layers panel"
				title="Show in Layers panel"
				onclick={() => onopenlayers?.()}
			>
				<Icon name="layers" size="sm" />
			</button>
		</div>
	{/if}
	{#if !layer}
		<div class="inspector__empty">
			<Icon name="layers" size="lg" />
			<!-- Mention the dropbox only when it's actually rendered (a layer-less document
			     shows no switcher row). -->
			<p class="inspector__empty-text">
				{layerOptions.length > 0
					? 'Select an element on the canvas — or pick a layer above — to edit its properties.'
					: 'Select an element on the canvas to edit its properties.'}
			</p>
		</div>
	{:else}
		<!-- Trailing per-row chips. The animate target is passed EXPLICITLY as a typed
		     AnimatableProperty (never inferred from the display label — that collided
		     across reused labels like "Opacity"×3 / dropped the crop edges). -->
		{#snippet modes(opts?: { bind?: { property: string; kind: 'text' | 'image' }; animate?: AnimatableProperty })}
			{@const bind = opts?.bind}
			{@const prop = opts?.animate ?? null}
			{@const animated = !!prop && isAnimated(prop)}
			{@const bound = !!bind && !!editor?.isBound(layer, bind.property)}
			{@const canBind = !!bind && !!editor && (!!editor.activeSource || bound)}
			<span class="modes">
				{#if bind}
					<!-- 🏷 link — the entry/status chip on bindable rows. On a bound TEXT row it
					     becomes a pure state badge (re-link = the Field ▾ row, unlink = its ✕),
					     never a dead click. -->
					{@const tagInert = bind.kind === 'text' && bound}
					<button
						class="mode"
						class:mode--active={bound}
						type="button"
						disabled={!canBind || tagInert}
						aria-pressed={bound}
						aria-label={bound ? 'Bound to a source field' : 'Bind to a source field'}
						title={canBind
							? tagInert
								? 'Bound — pick a field in the Field row below'
								: bound
									? 'Re-link or unlink this field'
									: 'Bind to a source field'
							: 'Attach a source in Settings to bind'}
						onclick={() => togglePicker(bind.property)}
					><Icon name="tag" size="xs" /></button>
				{/if}
				{#if prop && showAnimate}
					<!-- Make the value change over time (an "animation" = a value that goes
					     from its start value to its end value over a duration). Toggling on
					     adds a default over-time value + opens its inline editor; toggling
					     off removes it. Hidden in a static mode (watermark · #0298). -->
					<button
						class="mode"
						class:mode--active={animated}
						type="button"
						aria-pressed={animated}
						aria-label={animated ? 'Stop this value changing over time' : 'Make this value change over time'}
						title={animated ? 'Stop this value changing over time' : 'Make this value change over time'}
						onclick={() => (animated ? removeAnim : enableAnim)(prop)}
					><Icon name="sparkles" size="xs" /></button>
				{/if}
			</span>
		{/snippet}

		<!-- A compact PER-LAYER number row — the shared `numRow` plus the trailing link/animate
		     `modes` chips for this layer's property. Covers every numeric property (geometry,
		     crop, effect knobs); callbacks/limits/bind/animate arrive in `o`. The row markup
		     itself lives once in `numRow` (top of the file) — no duplicated .frow. -->
		{#snippet num(label: string, o: {
			id: string;
			name: string;
			value: number;
			oninput: (v: string) => void;
			onscrub: (n: number) => void;
			unit: string;
			aria: string;
			min?: number;
			max?: number;
			step?: number;
			bind?: { property: string; kind: 'text' | 'image' };
			animate?: AnimatableProperty;
		})}
			{#snippet trail()}{@render modes({ bind: o.bind, animate: o.animate })}{/snippet}
			{@render numRow(label, o, trail)}
		{/snippet}

		<!-- Inline over-time editor (design pass C) — rendered directly under an
		     animated field. A value that changes from its start value to its end value
		     over [start .. start+duration] IS the animation. Collapsed = a compact
		     summary (curve icon + From→To · duration); expand to edit. The easing is a
		     CURVE ICON, not text (EasingCurvePreview), de-emphasising the curve. -->
		{#snippet animEditor(property: AnimatableProperty)}
			{@const a = animFor(property)}
			<!-- showAnimate gate (#0298): no inline over-time editor in a no-animation mode
			     (watermark), even if the doc carries a stray track. -->
			{#if showAnimate && a}
				{@const open = expandedAnims.has(property)}
				{@const u = animUnit(property)}
				<div class="anim" class:anim--open={open}>
					<div class="anim__head">
						<button type="button" class="anim__summary" aria-expanded={open} title={open ? 'Collapse' : 'Edit value over time'} onclick={() => toggleAnimExpand(property)}>
							<Icon name={open ? 'chevron-down' : 'chevron-right'} size="xs" />
							<span class="anim__curve"><EasingCurvePreview easing={a.preset} size={{ w: 22, h: 14 }} /></span>
							<span class="anim__range">{a.from}{u} → {a.to}{u}</span>
							<span class="anim__dur">{a.durMs}ms</span>
						</button>
						<button type="button" class="row__clear anim__remove" title="Remove" aria-label="Stop this value changing over time" onclick={() => removeAnim(property)}><Icon name="x" size="xs" /></button>
					</div>
					{#if open}
						<div class="anim__detail">
							<div class="pair">
								{@render num('From', { id: `anim-from-${property}`, name: `anim_from_${property}`, value: a.from, oninput: (v) => patchAnimNumber(property, 'from', v), onscrub: (n) => patchAnimNumber(property, 'from', String(n)), unit: u, aria: `${a.label} start value` })}
								{@render num('To', { id: `anim-to-${property}`, name: `anim_to_${property}`, value: a.to, oninput: (v) => patchAnimNumber(property, 'to', v), onscrub: (n) => patchAnimNumber(property, 'to', String(n)), unit: u, aria: `${a.label} end value` })}
							</div>
							<div class="pair">
								{@render num('Start', { id: `anim-start-${property}`, name: `anim_start_${property}`, value: a.startMs, oninput: (v) => patchAnimNumber(property, 'startMs', v), onscrub: (n) => patchAnimNumber(property, 'startMs', String(n)), unit: 'ms', min: 0, aria: `${a.label} start time` })}
								{@render num('Duration', { id: `anim-dur-${property}`, name: `anim_dur_${property}`, value: a.durMs, oninput: (v) => patchAnimNumber(property, 'durMs', v), onscrub: (n) => patchAnimNumber(property, 'durMs', String(n)), unit: 'ms', min: 0, aria: `${a.label} duration` })}
							</div>
							<div class="frow anim__easerow">
								<span class="frow__label">Easing</span>
								<div class="frow__control anim__eases">
									{#each EASING_OPTS as opt (opt.value)}
										<button type="button" class="anim__ease" class:anim__ease--on={a.preset === opt.value} aria-pressed={a.preset === opt.value} title={opt.label} onclick={() => patchAnim(property, { easing: easingPreset(opt.value) })}>
											<EasingCurvePreview easing={opt.value} size={{ w: 30, h: 18 }} />
										</button>
									{/each}
								</div>
								<span class="frow__trail"></span>
							</div>
						</div>
					{/if}
				</div>
			{/if}
		{/snippet}

		{#snippet link(property: string, kind: 'text' | 'image')}
			{@const binding = editor?.bindingFor(layer, property)}
			{@const bound = !!binding}
			{@const opts = (editor?.fieldsFor(kind) ?? []).map((f) => ({ value: f.token, label: f.label }))}
			{#if editor && layer && (bound || bindPickerFor === property)}
				<!-- Xcode-style LINK rows (E-3) — when bound, the property expands into
				     labelled Source / Field rows (selectors + reveal/unlink actions). -->
				<div class="linkgroup">
					{#if bound}
						<!-- A binding may point at a DETACHED source (detach keeps bindings by design) —
						     prepend its alias as a degenerate option so the select never sits blank at
						     selectedIndex -1 (the srcOptions guard just below). -->
						{@const srcOpts = sourceOptionsFor(kind)}
						{@const srcOptions = srcOpts.some((o) => o.value === binding!.sourceAlias)
							? srcOpts
							: [{ value: binding!.sourceAlias, label: `${binding!.sourceAlias} · (detached)` }, ...srcOpts]}
						<div class="frow frow--link frow--stack">
							<span class="frow__label">Source</span>
							<div class="frow__control">
								<Select
									id={`bindsrc-${property}`}
									name={`bindsrc-${property}`}
									size="sm"
									value={binding!.sourceAlias}
									options={srcOptions}
									onchange={(alias) => alias && editor.rebindAlias(layer.id, property, alias)}
									aria-label="Bound source"
								/>
							</div>
							<span class="frow__trail"></span>
						</div>
						<div class="frow frow--link frow--stack">
							<span class="frow__label">Field</span>
							<div class="frow__control">
								{#if kind === 'text'}
									<!-- Text item = a SELECT (one click, Canva-style). The list belongs to the
									     BINDING's source; the current token guards in as a degenerate option so
									     the select never sits blank (srcOptions pattern). -->
									{@const itemOpts = editor
										.fieldsForAlias(binding!.sourceAlias, 'text')
										.map((f) => ({ value: f.token, label: f.label }))}
									{@const itemOptions = itemOpts.some((o) => o.value === binding!.token)
										? itemOpts
										: [{ value: binding!.token, label: fieldLabel(binding!.token, binding!.sourceAlias) }, ...itemOpts]}
									<Select
										id={`binditem-${property}`}
										name={`binditem-${property}`}
										size="sm"
										value={binding!.token}
										options={itemOptions}
										onchange={(token) => token && rebindToken(property, token)}
										aria-label="Bound field"
									/>
								{:else}
									<button
										type="button"
										class="bindrow__item"
										aria-expanded={bindPickerFor === property}
										title={`${fieldLabel(binding!.token, binding!.sourceAlias)} — pick another`}
										onclick={() => togglePicker(property)}
									>
										<Icon name={itemGlyph(binding!, kind)} size="xs" />
										<span class="bindrow__label">{fieldLabel(binding!.token, binding!.sourceAlias)}</span>
										<Icon name="chevron-down" size="xs" />
									</button>
								{/if}
							</div>
							<span class="frow__trail">
								<button
									type="button"
									class="inspector__icon-btn"
									aria-label="Reveal in panel"
									title="Reveal in panel"
									onclick={() => onreveal?.(revealContextFor(binding!, kind))}
								>
									<Icon name="external-link" size="sm" />
								</button>
								<button
									type="button"
									class="inspector__icon-btn"
									aria-label="Unlink"
									title="Unlink"
									onclick={() => (unlinkFor = property)}
								>
									<Icon name="x" size="sm" />
								</button>
							</span>
						</div>
					{/if}
					<!-- Format knobs (S-FORMAT) — INSIDE the LINK group, shown when the bound
					     property's formatter declares display options. One <Select> per knob; each
					     option's label is an ISOLATED PREVIEW (INC-0b — the choice on a fixed sample
					     with the OTHER knobs held at default, so the dropdown shows just that knob's
					     effect and never jumps as other knobs change), falling back to the choice NAME
					     when previews don't distinguish the choices. The ⟶ line shows the single live
					     rendered result for the record's actual value. -->
					{#if bound && editor}
						{@const fmtDesc = editor.formatDescriptorFor(binding)}
						{#if fmtDesc && fmtDesc.options.length > 0}
							{@const curOpts = editor.formatOptsFor(binding)}
							{@const fmtArg = editor.formatArgFor(binding)}
							{#each fmtDesc.options as opt (opt.key)}
								{@const choiceOptions = formatKnobChoices(fmtDesc, opt, fmtArg)}
								<div class="frow frow--link frow--stack">
									<span class="frow__label">{opt.label}</span>
									<div class="frow__control">
										<Select
											id={`bindfmt-${property}-${opt.key}`}
											name={`bindfmt-${property}-${opt.key}`}
											size="sm"
											value={curOpts[opt.key]}
											options={choiceOptions}
											onchange={(v) => v && editor.setFormatOption(layer.id, property, opt.key, v)}
											aria-label={`${opt.label} format`}
										/>
									</div>
									<span class="frow__trail"></span>
								</div>
							{/each}
							<!-- Enum alias editor — visible when the formatter is 'enum' and the consumer
							     provides read/write alias callbacks. One input row per known enum code;
							     edits mutate the alias map in-place (the formatter reads at call time)
							     and the consumer persists to localStorage. -->
							{#if fmtDesc.name === 'enum' && fmtArg && getEnumAlias && onaliaschange}
								{@const boundField = editor.fieldsForAlias(binding!.sourceAlias, 'text').find((f) => f.token === binding!.token)}
								{@const enumVals = boundField?.enumValues ?? []}
								{#if enumVals.length > 0}
									<div class="frow frow--link enum-aliases">
										<span class="frow__label">Aliases</span>
										<div class="frow__control enum-aliases__col">
											{#each enumVals as code (code)}
												{@const humanized = code.replace(/[-_]+/g, ' ')}
												<div class="enum-alias">
													<span class="enum-alias__code">{code}</span>
													<Input
														type="text"
														id={`bindalias-${property}-${code}`}
														name={`bindalias-${property}-${code}`}
														size="sm"
														value={getEnumAlias(fmtArg, code)}
														placeholder={humanized}
														oninput={(v) => onaliaschange!(fmtArg!, code, v)}
														aria-label={`Alias for '${code}'`}
													/>
												</div>
											{/each}
										</div>
										<span class="frow__trail"></span>
									</div>
								{/if}
							{/if}
							<div class="linkgroup__result" aria-live="polite">
								<span class="linkgroup__arrow" aria-hidden="true">⟶</span>
								<span class="linkgroup__value">{editor.resolvedValue(layer, property)}</span>
							</div>
						{/if}
					{/if}
					<!-- The revealed picker: the image grid popup, or the FIRST-link field select
					     for unbound text rows. Bound text rows never reveal it — their Field ▾ row
					     re-links in place. -->
					{#if bindPickerFor === property && (kind === 'image' || !bound)}
						{#if kind === 'image'}
							{@const photo = narrowSources('images')}
							{#if photo.length > 0}
								<!-- Image bind = the SAME grid as the Media panel (shared ImageSourcePicker,
								     G6c) — photographic images ONLY: map renders re-link via the Map panel. -->
								<div class="bind__images">
									<ImageSourcePicker
										sources={photo}
										canBind={true}
										boundAlias={binding?.sourceAlias}
										boundToken={binding?.token}
										onbind={(token, alias) => {
											editor.bindImageToken(layer.id, token, alias);
											bindPickerFor = null;
										}}
									/>
								</div>
							{:else}
								<p class="bind__empty">Attach an image source in Sources to bind.</p>
							{/if}
						{:else if opts.length > 0}
							<div class="frow frow--link frow--stack">
								<span class="frow__label">Link</span>
								<div class="frow__control">
									<Select
										id={`bind-${property}`}
										name={`bind-${property}`}
										size="sm"
										value=""
										options={[{ value: '', label: 'Choose a field…' }, ...opts]}
										onchange={(token) => {
											bindByToken(property, kind, token);
											bindPickerFor = null;
										}}
										aria-label="Choose a field to link"
									/>
								</div>
								<span class="frow__trail"></span>
							</div>
						{:else}
							<p class="bind__empty">No matching fields on the attached source.</p>
						{/if}
					{/if}
				</div>
			{/if}
		{/snippet}

		<header class="inspector__head">
			<span class="inspector__title">{layer.name}</span>
			<div class="inspector__head-actions">
				<button
					class="inspector__icon-btn"
					type="button"
					aria-label={layer.visible === false ? 'Show element' : 'Hide element'}
					onclick={() => ontogglevisibility?.()}
				>
					<Icon name={layer.visible === false ? 'eye-off' : 'eye'} size="sm" />
				</button>
				{#if !layer.system}
					<button
						class="inspector__icon-btn inspector__icon-btn--danger"
						type="button"
						aria-label="Delete element"
						title="Delete element"
						onclick={() => ondelete?.()}
					>
						<Icon name="trash-2" size="sm" />
					</button>
				{/if}
			</div>
		</header>
		<p class="inspector__kind">{layer.system ? 'BACKGROUND' : layer.type.toUpperCase()}</p>

		{#snippet imageSourceRow()}
			<!-- The bound-image SOURCE row — shared by image layers AND the image/map
			     background (same binding machinery: bind chip, picker popup, override). -->
			<div class="frow frow--top">
				<span class="frow__label">Source</span>
				<div class="frow__control">
					<Field for="insp-src">
						{#snippet children()}
							{#if editor?.isBound(layer!, 'src')}
								<div class="valrow">
									<Input id="insp-src" name="src" size="sm" value={editor.resolvedValue(layer, 'src')} oninput={(v) => editor.setOverride(layer!.id, 'src', v)} aria-label="Image URL (bound — edit to override)" />
									{#if editor.isOverridden(layer, 'src')}
										<button
											type="button"
											class="inspector__icon-btn valrow__restore"
											title={`Restore to “${editor.sourceValue(layer, 'src')}”`}
											aria-label="Restore the source value"
											onclick={() => editor.restore(layer!.id, 'src')}
										><Icon name="rotate-ccw" size="xs" /></button>
									{/if}
								</div>
							{:else}
								<Input id="insp-src" name="src" size="sm" value={layer!.src ?? ''} oninput={(v) => onupdate?.({ src: v })} aria-label="Image URL" />
							{/if}
						{/snippet}
					</Field>
				</div>
				<span class="frow__trail">{@render modes({ bind: { property: 'src', kind: 'image' } })}</span>
			</div>
			{@render link('src', 'image')}
		{/snippet}

		{#snippet borderGlowRow()}
			<!-- Border glow (land-canvas port) — only meaningful WITH a border, so the
			     row appears once one is set. 0 = off. -->
			{#if layer!.stroke}
				{@render num('Glow', { id: 'insp-border-glow', name: 'border_glow', value: layer!.stroke?.glow_intensity ?? 0, oninput: (v) => patchStrokeGlow(v), onscrub: (n) => patchStrokeGlow(String(n)), unit: '/10', min: 0, max: 10, step: 0.1, aria: 'Border glow intensity (0–10)' })}
			{/if}
		{/snippet}

		{#snippet shadowRows()}
			<!-- Box drop shadow — the colour row sets/clears it; its blur/offset/opacity
			     settings sit indented beneath and appear once it's on (grouped, so each
			     knob clearly belongs to the shadow). -->
			<div class="frow">
				<span class="frow__label">Shadow</span>
				<div class="frow__control"><ColorField id="insp-shadow-color" name="shadow_color" aria-label="Shadow color" value={layer!.shadow?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ shadow: { ...(layer!.shadow ?? DEFAULT_SHADOW), color: v } })} /></div>
				<span class="frow__trail">
					{#if layer!.shadow}
						<button type="button" class="row__clear" title="Remove shadow" aria-label="Remove shadow" onclick={() => onupdate?.({ shadow: undefined })}><Icon name="x" size="xs" /></button>
					{/if}
				</span>
			</div>
			{#if layer!.shadow}
				<div class="effgroup">
					{@render num('Blur', { id: 'insp-shadow-blur', name: 'shadow_blur', value: layer!.shadow?.blur ?? DEFAULT_SHADOW.blur, oninput: (v) => patchShadow('blur', v), onscrub: (n) => patchShadow('blur', String(n)), unit: 'px', min: 0, aria: 'Shadow blur' })}
					<div class="pair">
						{@render num('Offset X', { id: 'insp-shadow-ox', name: 'shadow_offset_x', value: layer!.shadow?.offset_x ?? 0, oninput: (v) => patchShadow('offset_x', v), onscrub: (n) => patchShadow('offset_x', String(n)), unit: 'px', aria: 'Shadow offset X' })}
						{@render num('Offset Y', { id: 'insp-shadow-oy', name: 'shadow_offset_y', value: layer!.shadow?.offset_y ?? 0, oninput: (v) => patchShadow('offset_y', v), onscrub: (n) => patchShadow('offset_y', String(n)), unit: 'px', aria: 'Shadow offset Y' })}
					</div>
					{@render num('Opacity', { id: 'insp-shadow-op', name: 'shadow_opacity', value: Math.round((layer!.shadow?.opacity ?? 0.4) * 100), oninput: (v) => patchShadowOpacity(v), onscrub: (n) => patchShadowOpacity(String(n)), unit: '%', min: 0, max: 100, aria: 'Shadow opacity percent' })}
				</div>
			{/if}
		{/snippet}

		<!-- [WM] TILE — a GENERIC repeat effect on ANY element (or group): mirror the element
		     across / down to fill the canvas, with an independent gap per axis. It renders LIVE on
		     the artboard (same kit renderer as export), and the whole pattern follows the element's
		     own Rotation — there is no hidden angle. Clicking any repeated copy selects this source
		     element. The header ✕ removes the effect. Reset = both off → the effect is dropped.
		     Extracted to a snippet so watermark mode can PROMOTE it to the top of the per-element
		     controls (tiling IS the watermark preference); design mode keeps it after ARRANGE. -->
		{#snippet tileSection()}
			<CollapsibleSection dense title="TILE" open>
				{#snippet actions()}
					{#if tileOn}
						<button type="button" class="row__clear" title="Remove tiling" aria-label="Remove tiling" onclick={() => onupdate?.({ tile: undefined })}><Icon name="x" size="xs" /></button>
					{/if}
				{/snippet}
				<div class="frow">
					<span class="frow__label">Repeat</span>
					<div class="frow__control">
						<SegmentedControl
							aria-label="Repeat the element to fill the canvas"
							options={[
								{ value: 'repeatH', label: 'Across', title: 'Repeat left and right', pressed: !!tile?.repeatH },
								{ value: 'repeatV', label: 'Down', title: 'Repeat up and down', pressed: !!tile?.repeatV }
							]}
							onchange={(v) => toggleRepeat(v as 'repeatH' | 'repeatV')}
						/>
					</div>
					<span class="frow__trail"></span>
				</div>
				<!-- Gaps only matter once an axis repeats — and each gap pairs with its axis. -->
				{#if tileOn}
					<div class="pair">
						{#if tile?.repeatH}
							{@render num('Gap across', { id: 'insp-tile-gapx', name: 'tile_gap_x', value: tile?.gapX ?? 0, oninput: (v) => patchTileGap('gapX', v), onscrub: (n) => patchTileGap('gapX', String(n)), unit: 'px', min: 0, aria: 'Horizontal gap between copies' })}
						{/if}
						{#if tile?.repeatV}
							{@render num('Gap down', { id: 'insp-tile-gapy', name: 'tile_gap_y', value: tile?.gapY ?? 0, oninput: (v) => patchTileGap('gapY', v), onscrub: (n) => patchTileGap('gapY', String(n)), unit: 'px', min: 0, aria: 'Vertical gap between copies' })}
						{/if}
					</div>
				{/if}
			</CollapsibleSection>
		{/snippet}

		<div class="inspector__cats">
			<!-- Watermark mode PROMOTES the per-element TILE controls to the TOP — tiling is THE
			     watermark preference, so it sits directly under the element header, applied LIVE. -->
			{#if showWatermarkBlock && !layer.system}
				{@render tileSection()}
			{/if}
			{#if layer.system}
				<!-- Background panel (user directive 2026-06-06): TYPE picks the face —
				     color (fill swatch), image (the standard bound-image row), map (picked
				     in the Map panel — its cards apply with confirmation). Geometry is
				     pinned to the artboard, so no Arrange/Crop here. -->
				<CollapsibleSection dense title="BACKGROUND">
					<div class="frow">
						<span class="frow__label">Type</span>
						<div class="frow__control">
							<Select
								id="insp-bg-mode"
								name="bg_mode"
								size="sm"
								value={layer.bg_mode ?? 'color'}
								options={[
									{ value: 'color', label: 'Color' },
									{ value: 'image', label: 'Image' },
									{ value: 'map', label: 'Map' }
								]}
								onchange={(m) => {
									if (m === 'color') {
										// The kit preserves `fill` across image/map — landing back on
										// Color restores the last chosen color, never resets to white.
										editor?.setBackground({ mode: 'color', fill: layer!.fill ?? { color: '#ffffff', transparency: 0 } });
									} else if (m === 'image' || m === 'map') {
										editor?.setBackground({ mode: m });
									}
								}}
								aria-label="Background type"
							/>
						</div>
						<span class="frow__trail"></span>
					</div>
					{#if (layer.bg_mode ?? 'color') === 'color'}
						<div class="frow">
							<span class="frow__label">Color</span>
							<div class="frow__control">
								<ColorField
									id="insp-bg-color"
									name="bg_color"
									aria-label="Background color"
									value={layer.fill?.color ?? '#ffffff'}
									colors={brandKit.colors}
									onpick={(v) => editor?.setBackground({ mode: 'color', fill: { ...(layer!.fill ?? { color: '#ffffff', transparency: 0 }), color: v } })}
								/>
							</div>
							<span class="frow__trail"></span>
						</div>
					{:else if layer.bg_mode === 'map' && !editor?.isBound(layer, 'src')}
						<!-- An unlinked map background is picked in the Map panel (its cards
						     carry the Set-as-background action). -->
						<p class="bind__empty">Pick a map in the Map panel — each card can set the background.</p>
						<span class="bind__actions">
							<button type="button" class="bind__btn" onclick={() => onreveal?.('map')}>Open Map panel</button>
						</span>
					{:else}
						{@render imageSourceRow()}
					{/if}
				</CollapsibleSection>
			{:else if layer.type === 'text' || layer.type === 'callout'}
				{@const contentKind = editor ? fieldKindFor(layer, 'content') : 'text'}
				<CollapsibleSection dense title="TEXT">
					<!-- CONTENT (#3 #4) — "Content" label + the field-mode tabs on the header line;
					     the control spans the section width below (no left gutter). The tabs
					     (text · linked · template) replace the old caption links + 🏷 chip —
					     selecting one converts the field kind, keeping the value. -->
					<div class="content">
						<div class="content__head">
							<span class="frow__label">Content</span>
							{#if editor}
								<SegmentedControl aria-label="Content field type" value={contentKind} options={FIELD_MODE_SEGMENTS} onchange={(k) => setContentKind(k as 'text' | 'linked' | 'template')} />
							{/if}
						</div>
						<div class="content__control">
							<!-- #2: rebuild the editing control when the SELECTED LAYER changes, so
							     switching layers always shows the new layer's content (the composer's
							     focus-guard otherwise keeps the prior layer's rendered badges; keying
							     by id is the clean Svelte reset for both composer and textarea). -->
							{#key layer.id}
							{#if contentKind === 'template' && editor}
								<!-- TEMPLATE (INC-C) — the contenteditable composer: each badge shows a
								     field's RESOLVED value under its title; type free text and insert
								     fields between them. Protocol-driven (@sbx/text-template), reused
								     beyond canvas. -->
								<TemplateComposer
									template={layer.content_template ?? []}
									catalog={editor.tokenCatalog()}
									resolver={editor.tokenResolver()}
									onchange={(t) => editor.setContentTemplate(layer.id, t)}
									placeholder="Type text and insert fields…"
								>
									{#snippet badgeFormat(ctx)}
										<!-- Reuse the SAME Format knobs as a linked field, targeting the badge's
										     token (#1c). A synthetic binding feeds the descriptor lookup; each
										     knob writes back via ctx.setFormat. -->
										{@const synthetic = { property: 'content', sourceAlias: '', token: ctx.token, format: ctx.format, formatOptions: ctx.formatOptions }}
										{@const fmtDesc = editor.formatDescriptorFor(synthetic)}
										{#if fmtDesc && fmtDesc.options.length > 0}
											{@const curOpts = editor.formatOptsFor(synthetic)}
											{@const fmtArg = editor.formatArgFor(synthetic)}
											{#each fmtDesc.options as opt (opt.key)}
												{@const choiceOptions = formatKnobChoices(fmtDesc, opt, fmtArg)}
												<div class="ttc-knob">
													<span class="ttc-knob__label">{opt.label}</span>
													<Select id={`tplfmt-${opt.key}`} name={`tplfmt-${opt.key}`} size="sm" value={curOpts[opt.key]} options={choiceOptions} onchange={(v) => v && ctx.setFormat(ctx.format, { ...ctx.formatOptions, [opt.key]: v })} aria-label={`${opt.label} format`} />
												</div>
											{/each}
										{/if}
									{/snippet}
								</TemplateComposer>
							{:else}
							<Field for="insp-content">
								{#snippet children()}
									<!-- Multi-line text in a multi-line control; grows via field-sizing
									     (.insp-grow). A LINKED field is LOCKED (read-only): a manual edit must
									     NOT silently override the binding and go stale (INC-B) — change its
									     display via the Format knobs below, or "Edit as text" to unlink. A
									     legacy manual override keeps its compact ↺ restore. -->
									{#if editor?.isBound(layer, 'content')}
										{@const overridden = editor.isOverridden(layer, 'content')}
										<div class="valrow">
											<Textarea id="insp-content" name="content" rows={2} class="insp-grow" value={editor.resolvedValue(layer, 'content')} readonly aria-label={overridden ? 'Content (linked, manually overridden — locked)' : 'Content (linked — locked)'} />
											{#if overridden}
												<button
													type="button"
													class="inspector__icon-btn valrow__restore"
													title={`Restore to “${editor.sourceValue(layer, 'content')}”`}
													aria-label="Restore the source value"
													onclick={() => editor.restore(layer.id, 'content')}
												><Icon name="rotate-ccw" size="xs" /></button>
											{/if}
										</div>
									{:else}
										<Textarea id="insp-content" name="content" rows={2} class="insp-grow" value={layer.content ?? ''} oninput={(v) => onupdate?.({ content: v })} aria-label="Content" />
									{/if}
								{/snippet}
							</Field>
							{/if}
							{/key}
						</div>
					</div>
					{#if contentKind !== 'template'}
						{@render link('content', 'text')}
					{/if}

					<div class="frow">
						<span class="frow__label">Font</span>
						<div class="frow__control">
							<!-- FontPicker (#3) — searchable: brand/system → recent → the full Google
							     catalogue (windowed, each row previews in its own face). It awaits the
							     font load before onselect, so the applied font paints immediately. -->
							<FontPicker
								value={layer.font ?? ''}
								brandFonts={brandFontList}
								catalogue={fontCatalogue}
								label="Font"
								onselect={(family) => onupdate?.({ font: family })}
							/>
						</div>
						<span class="frow__trail">{@render modes()}</span>
					</div>

					<div class="pair">
						{@render num('Size', { id: 'insp-size', name: 'font_size', value: layer.font_size ?? 16, oninput: (v) => patchNumber('font_size', v), onscrub: (n) => patchNumber('font_size', String(n)), unit: 'px', min: 1, aria: 'Font size' })}
						<div class="frow">
							<span class="frow__label">Color</span>
							<div class="frow__control"><ColorField id="insp-color" name="text_color" aria-label="Text color" value={layer.text_color ?? '#111111'} colors={brandKit.colors} onpick={(v) => onupdate?.({ text_color: v })} /></div>
							<span class="frow__trail"></span>
						</div>
					</div>

					<div class="pair">
						<div class="frow">
							<span class="frow__label">Format</span>
							<div class="frow__control">
								<SegmentedControl
									aria-label="Text format"
									options={[
										{ value: 'bold', icon: 'bold', title: 'Bold', pressed: weightValue === 'bold' },
										{ value: 'italic', icon: 'italic', title: 'Italic', pressed: styleValue === 'italic' }
									]}
									onchange={toggleFormat}
								/>
							</div>
							<span class="frow__trail"></span>
						</div>
						<div class="frow">
							<span class="frow__label">Align</span>
							<div class="frow__control">
								<SegmentedControl
									aria-label="Text alignment"
									value={layer.text_align ?? 'left'}
									options={ALIGN_SEGMENTS}
									onchange={(v) => onupdate?.({ text_align: v as Layer['text_align'] })}
								/>
							</div>
							<span class="frow__trail"></span>
						</div>
					</div>

					<!-- Text fit (S-FIT) — how content behaves when it would overflow its box
					     (so bound content of varying length never "jumps": scale, crop or
					     truncate instead of overflowing). -->
					<div class="frow">
						<span class="frow__label">Fit</span>
						<div class="frow__control">
							<SegmentedControl aria-label="Text fit" value={layer.text_fit ?? 'none'} options={TEXT_FIT_SEGMENTS} onchange={(v) => onupdate?.({ text_fit: v as Layer['text_fit'] })} />
						</div>
						<span class="frow__trail"></span>
					</div>
				</CollapsibleSection>

				<!-- Text box (user directive 2026-06-07): background + border + corner radius
				     + the box's drop shadow — collapsed by default (secondary decoration). -->
				<CollapsibleSection dense title="TEXT BOX" open={false}>
					<div class="pair">
						<div class="frow">
							<span class="frow__label">Background</span>
							<div class="frow__control"><ColorField id="insp-text-bg" name="text_bg" aria-label="Text background color" value={layer.fill?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ fill: { ...(layer.fill ?? { color: '#ffffff', transparency: 0 }), color: v } })} /></div>
							<span class="frow__trail">
								{#if layer.fill}
									<button type="button" class="row__clear" title="Remove background" aria-label="Remove background" onclick={() => onupdate?.({ fill: undefined })}><Icon name="x" size="xs" /></button>
								{/if}
							</span>
						</div>
						{@render num('Radius', { id: 'insp-radius', name: 'corner_radius', value: layer.corner_radius ?? 0, oninput: (v) => patchNumber('corner_radius', v), onscrub: (n) => patchNumber('corner_radius', String(n)), unit: 'px', min: 0, aria: 'Corner radius' })}
					</div>

					<div class="pair">
						<div class="frow">
							<span class="frow__label">Border</span>
							<div class="frow__control"><ColorField id="insp-text-border" name="text_border" aria-label="Text border color" value={layer.stroke?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ stroke: { ...(layer.stroke ?? { color: '#000000', thickness: 1, dash_type: 'solid', glow_intensity: 0 }), color: v } })} /></div>
							<span class="frow__trail">
								{#if layer.stroke}
									<button type="button" class="row__clear" title="Remove border" aria-label="Remove border" onclick={() => onupdate?.({ stroke: undefined })}><Icon name="x" size="xs" /></button>
								{/if}
							</span>
						</div>
						{@render num('Thickness', { id: 'insp-text-thickness', name: 'text_border_thickness', value: layer.stroke?.thickness ?? 0, oninput: (v) => patchStrokeThickness(v), onscrub: (n) => patchStrokeThickness(String(n)), unit: 'px', min: 0, aria: 'Border thickness' })}
					</div>

					{@render borderGlowRow()}
					{@render shadowRows()}
				</CollapsibleSection>

				<!-- Glyph effects — glow wins over shadow when both are set (kit semantics). -->
				<CollapsibleSection dense title="EFFECTS" open={false}>
					<!-- Each effect = a colour row that sets/clears it, with ITS settings
					     indented beneath and shown only when the effect is on — so it's
					     clear which knob (Intensity, Blur…) belongs to which effect. -->
					<div class="frow">
						<span class="frow__label">Text Glow</span>
						<div class="frow__control"><ColorField id="insp-tglow-color" name="text_glow_color" aria-label="Text glow color" value={layer.text_glow?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ text_glow: { ...(layer.text_glow ?? DEFAULT_TEXT_GLOW), color: v } })} /></div>
						<span class="frow__trail">
							{#if layer.text_glow}
								<button type="button" class="row__clear" title="Remove text glow" aria-label="Remove text glow" onclick={() => onupdate?.({ text_glow: undefined })}><Icon name="x" size="xs" /></button>
							{/if}
						</span>
					</div>
					{#if layer.text_glow}
						<div class="effgroup">
							{@render num('Intensity', { id: 'insp-tglow-int', name: 'text_glow_intensity', value: layer.text_glow?.intensity ?? DEFAULT_TEXT_GLOW.intensity, oninput: (v) => patchTextGlowIntensity(v), onscrub: (n) => patchTextGlowIntensity(String(n)), unit: '/10', min: 0, max: 10, step: 0.1, aria: 'Text glow intensity (0–10)' })}
						</div>
					{/if}

					<div class="frow">
						<span class="frow__label">Text Shadow</span>
						<div class="frow__control"><ColorField id="insp-tshadow-color" name="text_shadow_color" aria-label="Text shadow color" value={layer.text_shadow?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ text_shadow: { ...(layer.text_shadow ?? DEFAULT_SHADOW), color: v } })} /></div>
						<span class="frow__trail">
							{#if layer.text_shadow}
								<button type="button" class="row__clear" title="Remove text shadow" aria-label="Remove text shadow" onclick={() => onupdate?.({ text_shadow: undefined })}><Icon name="x" size="xs" /></button>
							{/if}
						</span>
					</div>
					{#if layer.text_shadow}
						<div class="effgroup">
							{@render num('Blur', { id: 'insp-tshadow-blur', name: 'text_shadow_blur', value: layer.text_shadow?.blur ?? DEFAULT_SHADOW.blur, oninput: (v) => patchTextShadow('blur', v), onscrub: (n) => patchTextShadow('blur', String(n)), unit: 'px', min: 0, aria: 'Text shadow blur' })}
							<div class="pair">
								{@render num('Offset X', { id: 'insp-tshadow-ox', name: 'text_shadow_offset_x', value: layer.text_shadow?.offset_x ?? 0, oninput: (v) => patchTextShadow('offset_x', v), onscrub: (n) => patchTextShadow('offset_x', String(n)), unit: 'px', aria: 'Text shadow offset X' })}
								{@render num('Offset Y', { id: 'insp-tshadow-oy', name: 'text_shadow_offset_y', value: layer.text_shadow?.offset_y ?? 0, oninput: (v) => patchTextShadow('offset_y', v), onscrub: (n) => patchTextShadow('offset_y', String(n)), unit: 'px', aria: 'Text shadow offset Y' })}
							</div>
							{@render num('Opacity', { id: 'insp-tshadow-op', name: 'text_shadow_opacity', value: Math.round((layer.text_shadow?.opacity ?? 0.4) * 100), oninput: (v) => patchTextShadowOpacity(v), onscrub: (n) => patchTextShadowOpacity(String(n)), unit: '%', min: 0, max: 100, aria: 'Text shadow opacity percent' })}
						</div>
					{/if}
				</CollapsibleSection>
			{:else if layer.type === 'shape'}
				<CollapsibleSection dense title="APPEARANCE">
					<div class="frow">
						<span class="frow__label">Fill</span>
						<div class="frow__control"><ColorField id="insp-fill" name="fill" aria-label="Fill color" value={layer.fill?.color ?? '#000000'} colors={brandKit.colors} onpick={(v) => onupdate?.({ fill: { ...(layer.fill ?? { color: '#000000', transparency: 0, pattern: 'solid' }), color: v } })} /></div>
						<span class="frow__trail">{@render modes()}</span>
					</div>

					<div class="pair">
						<div class="frow">
							<span class="frow__label">Stroke</span>
							<div class="frow__control"><ColorField id="insp-stroke" name="stroke" aria-label="Stroke color" value={layer.stroke?.color ?? '#000000'} colors={brandKit.colors} onpick={(v) => onupdate?.({ stroke: { ...(layer.stroke ?? { color: '#000000', thickness: 1, dash_type: 'solid', glow_intensity: 0 }), color: v } })} /></div>
							<span class="frow__trail">{@render modes()}</span>
						</div>
						{@render num('Thickness', { id: 'insp-stroke-thickness', name: 'stroke_thickness', value: layer.stroke?.thickness ?? 1, oninput: (v) => patchStrokeThickness(v), onscrub: (n) => patchStrokeThickness(String(n)), unit: 'px', min: 0, aria: 'Stroke thickness' })}
					</div>

					{@render num('Radius', { id: 'insp-shape-radius', name: 'corner_radius', value: layer.corner_radius ?? 0, oninput: (v) => patchNumber('corner_radius', v), onscrub: (n) => patchNumber('corner_radius', String(n)), unit: 'px', min: 0, aria: 'Corner radius' })}

					{@render borderGlowRow()}
				</CollapsibleSection>

				<CollapsibleSection dense title="EFFECTS" open={false}>
					{@render shadowRows()}
				</CollapsibleSection>
			{:else if layer.type === 'image'}
				<CollapsibleSection dense title="SOURCE">
					{@render imageSourceRow()}
				</CollapsibleSection>

				<!-- Images take a BORDER (+ glow) but never a box shadow (user call 2026-06-07). -->
				<CollapsibleSection dense title="BORDER" open={false}>
					<div class="pair">
						<div class="frow">
							<span class="frow__label">Border</span>
							<div class="frow__control"><ColorField id="insp-img-border" name="image_border" aria-label="Image border color" value={layer.stroke?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ stroke: { ...(layer.stroke ?? { color: '#000000', thickness: 1, dash_type: 'solid', glow_intensity: 0 }), color: v } })} /></div>
							<span class="frow__trail">
								{#if layer.stroke}
									<button type="button" class="row__clear" title="Remove border" aria-label="Remove border" onclick={() => onupdate?.({ stroke: undefined })}><Icon name="x" size="xs" /></button>
								{/if}
							</span>
						</div>
						{@render num('Thickness', { id: 'insp-img-thickness', name: 'image_border_thickness', value: layer.stroke?.thickness ?? 0, oninput: (v) => patchStrokeThickness(v), onscrub: (n) => patchStrokeThickness(String(n)), unit: 'px', min: 0, aria: 'Image border thickness' })}
					</div>
					{@render borderGlowRow()}
				</CollapsibleSection>

				<!-- Legibility effects (#0300) — a TINT (recolor the silhouette) + a GLOW halo so
				     ONE logo reads on any background. Opacity is in ARRANGE. "Contrast halo" sets
				     the proven soft-dark-halo preset (subtitle / lower-third); all stays editable. -->
				<CollapsibleSection dense title="EFFECTS" open={false}>
					<button type="button" class="wm-preset" title="Add a soft dark halo so the mark reads on any background" onclick={applyContrastHalo}>
						<Icon name="sparkles" size="sm" />
						<span>Contrast halo</span>
					</button>
					<!-- Tint — recolor the alpha silhouette (one asset → light or dark). -->
					<div class="frow">
						<span class="frow__label">Tint</span>
						<div class="frow__control"><ColorField id="insp-img-tint" name="image_tint" aria-label="Image tint color" value={layer.tint?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ tint: { ...(layer!.tint ?? DEFAULT_IMAGE_TINT), color: v } })} /></div>
						<span class="frow__trail">
							{#if layer.tint}
								<button type="button" class="row__clear" title="Remove tint" aria-label="Remove tint" onclick={() => onupdate?.({ tint: undefined })}><Icon name="x" size="xs" /></button>
							{/if}
						</span>
					</div>
					{#if layer.tint}
						{@render num('Strength', { id: 'insp-img-tint-strength', name: 'image_tint_strength', value: Math.round((layer.tint?.strength ?? 1) * 100), oninput: (v) => patchTintStrength(v), onscrub: (n) => patchTintStrength(String(n)), unit: '%', min: 0, max: 100, aria: 'Tint strength percent' })}
					{/if}
					<!-- Glow halo — the centred contrasting halo (offset-free; the any-bg cue). -->
					<div class="frow">
						<span class="frow__label">Glow</span>
						<div class="frow__control"><ColorField id="insp-img-glow" name="image_glow" aria-label="Image glow color" value={layer.glow?.color ?? ''} colors={brandKit.colors} onpick={(v) => onupdate?.({ glow: { ...(layer!.glow ?? DEFAULT_IMAGE_GLOW), color: v } })} /></div>
						<span class="frow__trail">
							{#if layer.glow}
								<button type="button" class="row__clear" title="Remove glow" aria-label="Remove glow" onclick={() => onupdate?.({ glow: undefined })}><Icon name="x" size="xs" /></button>
							{/if}
						</span>
					</div>
					{#if layer.glow}
						{@render num('Intensity', { id: 'insp-img-glow-int', name: 'image_glow_intensity', value: layer.glow?.intensity ?? DEFAULT_IMAGE_GLOW.intensity, oninput: (v) => patchImageGlowIntensity(v), onscrub: (n) => patchImageGlowIntensity(String(n)), unit: '/10', min: 0, max: 10, step: 0.1, aria: 'Image glow intensity (0–10)' })}
					{/if}
				</CollapsibleSection>
			{:else if layer.type === 'map'}
				<!-- Map viewport — zoom + display type. Center is set by anchoring/drag,
				     not typed here (out of scope this slice). -->
				<CollapsibleSection dense title="MAP">
					{@render num('Zoom', { id: 'insp-map-zoom', name: 'map_zoom', value: layer.map_config?.zoom ?? 14, oninput: (v) => { const n = Number(v); if (Number.isFinite(n)) patchMapConfig({ zoom: n }); }, onscrub: (n) => patchMapConfig({ zoom: n }), unit: '', min: 1, max: 21, step: 1, aria: 'Map zoom level' })}
					<div class="frow">
						<span class="frow__label">Map type</span>
						<div class="frow__control">
							<Select
								id="insp-map-type"
								name="map_type"
								size="sm"
								value={layer.map_config?.mapType ?? 'roadmap'}
								options={[
									{ value: 'roadmap', label: 'Roadmap' },
									{ value: 'satellite', label: 'Satellite' },
									{ value: 'terrain', label: 'Terrain' },
									{ value: 'hybrid', label: 'Hybrid' }
								]}
								onchange={(m) => m && patchMapConfig({ mapType: m as MapType })}
								aria-label="Map type"
							/>
						</div>
						<span class="frow__trail"></span>
					</div>
				</CollapsibleSection>
			{/if}

			{#if !layer.system}
				<!-- Common geometry (all layer types except the pinned background). -->
				<CollapsibleSection dense title="ARRANGE">
					<div class="pair">
						{@render num('X', { id: 'insp-x', name: 'x', value: layer.x, oninput: (v) => patchNumber('x', v), onscrub: (n) => patchNumber('x', String(n)), unit: 'px', aria: 'X position', animate: 'x' })}
						{@render num('Y', { id: 'insp-y', name: 'y', value: layer.y, oninput: (v) => patchNumber('y', v), onscrub: (n) => patchNumber('y', String(n)), unit: 'px', aria: 'Y position', animate: 'y' })}
					</div>
					{#if isAnimated('x')}{@render animEditor('x')}{/if}
					{#if isAnimated('y')}{@render animEditor('y')}{/if}
					<div class="pair">
						{@render num('Width', { id: 'insp-w', name: 'width', value: layer.width ?? 0, oninput: (v) => patchNumber('width', v), onscrub: (n) => patchNumber('width', String(n)), unit: 'px', min: 1, aria: 'Width', animate: 'width' })}
						{@render num('Height', { id: 'insp-h', name: 'height', value: layer.height ?? 0, oninput: (v) => patchNumber('height', v), onscrub: (n) => patchNumber('height', String(n)), unit: 'px', min: 1, aria: 'Height', animate: 'height' })}
					</div>
					{#if isAnimated('width')}{@render animEditor('width')}{/if}
					{#if isAnimated('height')}{@render animEditor('height')}{/if}
					<div class="pair">
						{@render num('Opacity', { id: 'insp-opacity', name: 'opacity', value: opacityPct, oninput: (v) => patchOpacity(v), onscrub: (n) => patchOpacity(String(n)), unit: '%', min: 0, max: 100, aria: 'Opacity percent', animate: 'opacity' })}
						{@render num('Rotation', { id: 'insp-rotation', name: 'rotation', value: layer.rotation ?? 0, oninput: (v) => patchNumber('rotation', v), onscrub: (n) => patchNumber('rotation', String(n)), unit: '°', aria: 'Rotation degrees', animate: 'rotation' })}
					</div>
					{#if isAnimated('opacity')}{@render animEditor('opacity')}{/if}
					{#if isAnimated('rotation')}{@render animEditor('rotation')}{/if}
				</CollapsibleSection>

				<!-- Design mode keeps TILE here (after ARRANGE). Watermark mode renders it at the
				     TOP of the per-element controls instead (see the cats opening) — never twice. -->
				{#if !showWatermarkBlock}
					{@render tileSection()}
				{/if}

				<!-- Per-edge inset crop (user directive 2026-06-07) — clips the layer's box;
				     each edge can change over time (▦). The header ✕ resets all four. -->
				<CollapsibleSection dense title="CROP" open={false}>
					{#snippet actions()}
						{#if cropSet}
							<button type="button" class="row__clear" title="Reset crop" aria-label="Reset crop" onclick={() => onupdate?.({ crop_top: 0, crop_right: 0, crop_bottom: 0, crop_left: 0 })}><Icon name="x" size="xs" /></button>
						{/if}
					{/snippet}
					<div class="pair">
						{@render num('Top', { id: 'insp-crop-top', name: 'crop_top', value: layer.crop_top ?? 0, oninput: (v) => patchCrop('crop_top', v), onscrub: (n) => patchCrop('crop_top', String(n)), unit: 'px', min: 0, aria: 'Crop top', animate: 'crop_top' })}
						{@render num('Right', { id: 'insp-crop-right', name: 'crop_right', value: layer.crop_right ?? 0, oninput: (v) => patchCrop('crop_right', v), onscrub: (n) => patchCrop('crop_right', String(n)), unit: 'px', min: 0, aria: 'Crop right', animate: 'crop_right' })}
					</div>
					{#if isAnimated('crop_top')}{@render animEditor('crop_top')}{/if}
					{#if isAnimated('crop_right')}{@render animEditor('crop_right')}{/if}
					<div class="pair">
						{@render num('Bottom', { id: 'insp-crop-bottom', name: 'crop_bottom', value: layer.crop_bottom ?? 0, oninput: (v) => patchCrop('crop_bottom', v), onscrub: (n) => patchCrop('crop_bottom', String(n)), unit: 'px', min: 0, aria: 'Crop bottom', animate: 'crop_bottom' })}
						{@render num('Left', { id: 'insp-crop-left', name: 'crop_left', value: layer.crop_left ?? 0, oninput: (v) => patchCrop('crop_left', v), onscrub: (n) => patchCrop('crop_left', String(n)), unit: 'px', min: 0, aria: 'Crop left', animate: 'crop_left' })}
					</div>
					{#if isAnimated('crop_bottom')}{@render animEditor('crop_bottom')}{/if}
					{#if isAnimated('crop_left')}{@render animEditor('crop_left')}{/if}
				</CollapsibleSection>
			{/if}
		</div>
	{/if}

	<!-- DOCUMENT-level preview backdrop — pinned to the BOTTOM, collapsed, BELOW the
	     per-element controls (and shown in the no-selection state too). Never above the
	     element's own watermark properties. -->
	{@render previewBackdropBlock()}
</aside>

<!-- Unlink confirm — the row ✕ asks before unbinding. The layer keeps the resolved
     value as its static one (unbind semantics), so the loss is the LINK, not the look. -->
{#if editor && layer}
	{@const pending = unlinkFor ? editor.bindingFor(layer, unlinkFor) : undefined}
	<Modal open={!!pending} title="Unlink field" size="sm" onclose={() => (unlinkFor = null)}>
		<p>
			Unlink “{pending ? fieldLabel(pending.token, pending.sourceAlias) : ''}”? The layer keeps the
			current value as its own — it just stops following the source.
		</p>
		{#snippet footer()}
			<Button variant="secondary" size="sm" onclick={() => (unlinkFor = null)}>Cancel</Button>
			<Button
				variant="primary"
				size="sm"
				onclick={() => {
					if (unlinkFor) {
						editor.unbind(layer.id, unlinkFor);
						bindPickerFor = null;
					}
					unlinkFor = null;
				}}>Unlink</Button>
		{/snippet}
	</Modal>
{/if}

<style>
	.inspector {
		display: flex;
		flex-direction: column;
		width: 288px;
		min-width: 288px;
		background: var(--cv-color-surface, #fff);
		border-left: 1px solid var(--cv-border-color, #e0e0e0);
		overflow-y: auto;
		/* LEFT-aligned label gutter for single (non-pair) rows — labels on the left,
		   inputs filling to the right, every single row's label + input in one column
		   (no more right-aligned zigzag). Pair cells hug their own short labels. */
		--insp-label-w: 5rem;
	}

	/* Layer switcher (D-4) — the quick-jump row above the title. Sticky so it stays
	   reachable when the property list scrolls ("switch quick" from anywhere). */
	.inspector__switcher {
		position: sticky;
		top: 0;
		z-index: 1;
		display: flex;
		align-items: center;
		gap: var(--cv-space-sm, 0.5rem);
		padding: var(--cv-space-md, 1rem) var(--cv-space-md, 1rem) var(--cv-space-sm, 0.5rem);
		background: var(--cv-color-surface, #fff);
	}

	.inspector__switcher-select {
		flex: 1;
		min-width: 0;
	}

	.inspector__empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--cv-space-md, 1rem);
		flex: 1;
		padding: var(--cv-space-xl, 2rem);
		color: var(--cv-color-neutral-400, #a0a0a0);
		text-align: center;
	}

	.inspector__empty-text {
		margin: 0;
		font-size: 0.8125rem;
		line-height: 1.5;
	}

	/* Watermark block (#0298) — pinned at the top with a hairline separator below so it
	   reads as a distinct document-level section above the per-layer Inspector. */
	.inspector__watermark {
		padding: 0 var(--cv-space-md, 1rem);
		border-top: 1px solid var(--cv-color-neutral-200, #e8e8e8);
	}

	/* "Shutterstock Classic" preset — a full-width quiet CTA at the top of the block.
	   Accent-tinted so it reads as the one-click shortcut, not a destructive action. */
	.wm-preset {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.375rem;
		width: 100%;
		margin: 0 0 var(--cv-space-sm, 0.5rem);
		padding: 0.4rem 0.6rem;
		border: 1px solid var(--cv-color-accent-soft, #e6dcc0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-accent-soft, #f3ecda);
		color: var(--cv-color-accent-dark, #7a6326);
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
	}

	.wm-preset:hover {
		filter: brightness(0.97);
	}

	.wm-preset:focus-visible {
		outline: 2px solid var(--cv-color-accent-dark, #7a6326);
		outline-offset: 1px;
	}

	/* Inline caption for a control group (e.g. the preview-backdrop "preview only" note) —
	   full width, never clipped, reads as a quiet annotation under the section header. */
	.inspector__hint {
		margin: 0 0 var(--cv-space-xs, 0.25rem);
		font-size: 0.6875rem;
		line-height: 1.4;
		color: var(--cv-color-neutral-500, #707070);
	}

	.inspector__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--cv-space-sm, 0.5rem);
		padding: var(--cv-space-md, 1rem) var(--cv-space-md, 1rem) 0;
	}

	.inspector__title {
		font-size: 0.9375rem;
		font-weight: 600;
		color: var(--cv-color-neutral-800, #202020);
	}

	/* Right-aligned action cluster (visibility + delete). */
	.inspector__head-actions {
		display: inline-flex;
		align-items: center;
		gap: 2px;
	}

	.inspector__icon-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.75rem;
		height: 1.75rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-500, #707070);
		cursor: pointer;
	}

	.inspector__icon-btn:hover {
		background: var(--cv-color-neutral-200, #e8e8e8);
	}

	.inspector__icon-btn--danger:hover {
		background: var(--cv-color-danger-soft, rgba(217, 45, 32, 0.1));
		color: var(--cv-color-danger, #d92d20);
	}

	.inspector__kind {
		margin: 0;
		padding: 0 var(--cv-space-md, 1rem) var(--cv-space-xs, 0.25rem);
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.08em;
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	/* Category stack — each CollapsibleSection (dense) owns its own divider; the
	   inspector just adds horizontal padding so headers + rows align. */
	.inspector__cats {
		display: flex;
		flex-direction: column;
		padding: 0 var(--cv-space-md, 1rem);
		border-top: 1px solid var(--cv-border-color, #e0e0e0);
	}

	/* Horizontal property row — [label gutter] [control] [trailing modes/clear].
	   The control column may shrink (min-width 0) so long values never push the
	   trailing chips out of the panel. */
	.frow {
		display: grid;
		grid-template-columns: var(--insp-label-w) minmax(0, 1fr) auto;
		align-items: center;
		column-gap: var(--cv-space-sm, 0.5rem);
		min-height: 1.75rem;
	}

	/* Top-align the label + trailing for tall controls (CONTENT textarea, bound
	   image URL with restore). */
	.frow--top {
		align-items: start;
	}

	.frow--top .frow__label,
	.frow--top .frow__trail {
		margin-top: 0.3125rem;
	}

	/* CONTENT field (#3) — label + field-mode tabs on the header line, the control
	   full-width below (no label gutter), so the text area / composer spans the
	   section's horizontal padding edges like the panel's other full-width blocks. */
	.content {
		display: flex;
		flex-direction: column;
		gap: 0.375rem;
	}

	.content__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--cv-space-sm, 0.5rem);
		min-height: 1.75rem;
	}

	.content__control {
		min-width: 0;
	}

	/* Format knobs reused inside the TemplateComposer's badge popover (#1c) — one label/select
	   row per knob, aligned to the popover's own label column. */
	.ttc-knob {
		display: grid;
		grid-template-columns: 5rem 1fr;
		align-items: center;
		gap: var(--cv-space-md, 1rem);
		margin-top: 0.375rem;
	}

	.ttc-knob__label {
		font-size: 0.6875rem;
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
		color: var(--cv-color-neutral-500, #707070);
	}

	.frow__label {
		font-size: 0.6875rem;
		font-weight: 600;
		letter-spacing: 0.01em;
		text-align: left;
		color: var(--cv-color-neutral-500, #707070);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.frow__control {
		min-width: 0;
	}

	/* Trailing slot — holds the mode chips and/or a clear ✕; collapses to 0 width
	   when empty so the control gets the full row. */
	.frow__trail {
		display: inline-flex;
		align-items: center;
		gap: 2px;
	}

	/* Pair — two compact cells on one line (X/Y, Size/Color, …). Cells hug their
	   own short labels instead of using the panel-wide gutter. They sit 2-up while
	   they fit; a cell whose label + value + chip can't make half the panel wraps
	   to its own full-width line — the value never clips (the Xcode "long value
	   spills to its own line" behaviour). */
	.pair {
		display: flex;
		flex-wrap: wrap;
		gap: var(--cv-space-sm, 0.5rem);
		align-items: start;
	}

	.pair .frow {
		/* basis 0 (not auto) so the wrap decision is driven by each cell's
		   MIN-content (label + input floor + chip), not the 1fr max-content —
		   short pairs (X/Y) keep 2-up, long ones break to their own line. */
		flex: 1 1 0;
		/* Tighter inner gap than single rows so short pairs (X/Y) keep 2-up. */
		column-gap: var(--cv-space-xs, 0.25rem);
		/* Input floor fits a 4-digit value + unit so numbers never clip; when two
		   floors can't share one row, flex-wrap drops the pair to single lines. */
		grid-template-columns: auto minmax(4.25rem, 1fr) auto;
	}

	.pair .frow__label {
		text-align: left;
		overflow: visible;
	}

	/* LINK rows (E-3) — labelled Source / Field selectors under a bound property,
	   indented with a left accent so they read as belonging to the row above. */
	.linkgroup {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-xs, 0.25rem);
		/* Left indent ties it to the bound field above; vertical spacing comes from
		   the section body gap (no compounded margins → uniform rhythm). */
		margin-left: var(--cv-space-sm, 0.5rem);
		padding-left: var(--cv-space-sm, 0.5rem);
		border-left: 2px solid var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
	}

	/* Live formatter result (S-FORMAT) — the ⟶ line under the Format knobs, showing the
	   actual rendered value with the current options applied. Never clipped (full value). */
	.linkgroup__result {
		display: flex;
		align-items: baseline;
		gap: var(--cv-space-xs, 0.25rem);
		font-size: 0.75rem;
		color: var(--cv-color-neutral-600, #525252);
	}

	.linkgroup__arrow {
		color: var(--cv-color-neutral-400, #a0a0a0);
	}

	.linkgroup__value {
		font-variant-numeric: tabular-nums;
		font-weight: 500;
		color: var(--cv-color-neutral-800, #2a2a2a);
	}

	.frow--link {
		--insp-label-w: 3.25rem;
	}

	/* Stacked row — label (+ any trailing actions) on the first line, the control
	   full-width on the second. Used for long-value selects (Source/Field/Font) so
	   the value never clips (Xcode "Full Path"-style spill to its own line). */
	.frow--stack {
		grid-template-columns: 1fr auto;
		grid-template-areas:
			'label trail'
			'control control';
		row-gap: 2px;
	}

	.frow--stack .frow__label {
		grid-area: label;
		text-align: left;
	}

	.frow--stack .frow__trail {
		grid-area: trail;
	}

	.frow--stack .frow__control {
		grid-area: control;
	}

	/* Effect settings group — a left-accented indent under an effect's colour row so
	   each knob (Intensity, Blur, Offset…) clearly belongs to its effect. */
	.effgroup {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-xs, 0.25rem);
		margin-left: var(--cv-space-sm, 0.5rem);
		padding-left: var(--cv-space-sm, 0.5rem);
		border-left: 2px solid var(--cv-color-neutral-200, #e8e8e8);
	}

	/* Inline image-bind grid (G6c) — caps height so a long gallery scrolls inside the
	   Inspector instead of pushing the panel. */
	.bind__images {
		max-height: 18rem;
		overflow-y: auto;
	}

	/* Image item button (the Field row for image binds — opens the grid popup). */
	.bindrow__item {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		width: 100%;
		padding: 0.25rem 0.5rem;
		font: inherit;
		font-size: 0.75rem;
		font-weight: 600;
		text-align: left;
		color: var(--cv-color-primary, #333333);
		background: var(--cv-color-primary-soft, rgba(0, 0, 0, 0.08));
		border: 1px solid transparent;
		border-radius: var(--cv-radius-sm, 0.375rem);
		cursor: pointer;
	}

	.bindrow__item:hover {
		border-color: var(--cv-color-primary, #333333);
	}

	.bindrow__label {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* Actions right-aligned (hard rule). */
	.bind__actions {
		display: inline-flex;
		gap: 0.25rem;
		margin-left: auto;
	}

	.bind__btn {
		border: none;
		background: transparent;
		padding: 0;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--cv-color-primary, #333333);
		cursor: pointer;
	}

	.bind__btn:hover {
		text-decoration: underline;
	}

	.bind__empty {
		margin: 0;
		font-size: 0.75rem;
		color: var(--cv-color-neutral-500, #707070);
	}

	/* Bound value + compact ↺ restore (replaces the EDITED badge + Restore meta line;
	   the tooltip carries “Restore to <source value>”). */
	.valrow {
		display: flex;
		align-items: flex-start;
		gap: var(--cv-space-xs, 0.25rem);
	}

	.valrow > :global(:first-child) {
		flex: 1;
		min-width: 0;
	}

	.valrow__restore {
		flex-shrink: 0;
		width: 1.25rem;
		height: 1.25rem;
		margin-top: 0.25rem;
	}

	/* Mode toggles as small squares — bordered chips; the ACTIVE mode is a filled
	   primary square (link/animate engaged), inactive stays quiet. */
	.modes {
		display: inline-flex;
		gap: 2px;
	}

	.mode {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.25rem;
		height: 1.25rem;
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-surface, #fff);
		color: var(--cv-color-neutral-400, #a0a0a0);
		cursor: pointer;
	}

	.mode:hover:not(:disabled) {
		background: var(--cv-color-neutral-100, #f5f5f5);
		color: var(--cv-color-neutral-700, #383838);
	}

	.mode--active {
		background: var(--cv-color-primary, #333333);
		border-color: var(--cv-color-primary, #333333);
		color: var(--cv-color-surface, #fff);
	}

	.mode:disabled {
		cursor: not-allowed;
		opacity: 0.45;
	}

	/* A bound text row's 🏷 is a STATE badge — no action left (re-link = the Field ▾
	   row, unlink = the row ✕), but it must stay lit, never washed out. */
	.mode--active:disabled {
		cursor: default;
		opacity: 1;
	}

	/* Clear-✕ beside a set color / effect — resets that face to none. */
	.row__clear {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 1.25rem;
		height: 1.25rem;
		border: none;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		color: var(--cv-color-neutral-400, #a0a0a0);
		cursor: pointer;
	}

	.row__clear:hover {
		background: var(--cv-color-neutral-200, #e8e8e8);
		color: var(--cv-color-error, #c0392b);
	}

	/* Inline over-time editor — a bordered box directly under an animated field.
	   Collapsed = curve icon + From→To · duration summary; expand to edit. The box
	   is indented + accented so it reads as belonging to the field above. */
	.anim {
		/* Left indent ties it to the field above; vertical spacing comes from the
		   section body gap so the rhythm stays uniform (no compounded margins). */
		margin-left: var(--cv-space-sm, 0.5rem);
		border: 1px solid var(--cv-border-color, #e0e0e0);
		border-left: 2px solid var(--cv-color-primary-soft, rgba(0, 0, 0, 0.12));
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: var(--cv-color-neutral-50, #fafafa);
	}

	.anim--open {
		border-left-color: var(--cv-color-primary, #333333);
	}

	.anim__head {
		display: flex;
		align-items: center;
	}

	.anim__summary {
		flex: 1;
		display: flex;
		align-items: center;
		gap: var(--cv-space-xs, 0.25rem);
		min-width: 0;
		padding: 0.3125rem 0.375rem;
		border: none;
		background: transparent;
		color: var(--cv-color-neutral-700, #383838);
		cursor: pointer;
		text-align: left;
	}

	.anim__curve {
		display: inline-flex;
		flex-shrink: 0;
	}

	.anim__range {
		font-size: 0.75rem;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	.anim__dur {
		margin-left: auto;
		padding-left: var(--cv-space-xs, 0.25rem);
		font-size: 0.6875rem;
		font-variant-numeric: tabular-nums;
		color: var(--cv-color-neutral-500, #707070);
		white-space: nowrap;
	}

	.anim__remove {
		flex-shrink: 0;
		margin-right: 0.1875rem;
	}

	.anim__detail {
		display: flex;
		flex-direction: column;
		gap: var(--cv-space-xs, 0.25rem);
		padding: 0 0.375rem 0.375rem;
	}

	/* Easing shown as small CURVE ICONS (EasingCurvePreview), not text. */
	.anim__eases {
		display: flex;
		gap: 0.25rem;
		flex-wrap: wrap;
	}

	.anim__ease {
		display: inline-flex;
		padding: 0;
		border: 1px solid transparent;
		border-radius: var(--cv-radius-sm, 0.375rem);
		background: transparent;
		cursor: pointer;
		line-height: 0;
	}

	.anim__ease:hover {
		border-color: var(--cv-color-neutral-300, #d0d0d0);
	}

	.anim__ease--on {
		border-color: var(--cv-color-primary, #333333);
	}

	/* Right-aligned tabular numerals; native spinners hidden — the ScrubUnit suffix
	   sits where they would, and the compact rows are too tight for them. */
	.inspector :global(input.num) {
		text-align: right;
		font-variant-numeric: tabular-nums;
		-moz-appearance: textfield;
		appearance: textfield;
	}

	.inspector :global(input.num::-webkit-outer-spin-button),
	.inspector :global(input.num::-webkit-inner-spin-button) {
		-webkit-appearance: none;
		margin: 0;
	}

	/* CONTENT textarea — grows with the text up to ~4 rows, then scrolls. */
	.inspector :global(.insp-grow) {
		min-height: 0;
		max-height: 7.5rem;
		overflow-y: auto;
	}

	@supports (field-sizing: content) {
		.inspector :global(.insp-grow) {
			field-sizing: content;
			resize: none;
		}
	}

	/* Enum alias editor — stacked rows (code chip + text input) inside the Format block.
	   Aligns with the existing frow label-gutter; the frow__control column holds the table. */
	.enum-aliases__col {
		display: flex;
		flex-direction: column;
		gap: 0.1875rem;
		width: 100%;
	}
	.enum-alias {
		display: grid;
		grid-template-columns: 5rem 1fr;
		align-items: center;
		gap: 0.25rem;
	}
	.enum-alias__code {
		font-size: 0.6875rem;
		color: var(--cv-color-neutral-400, #a0a0a0);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
