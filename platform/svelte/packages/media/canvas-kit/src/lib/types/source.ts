// Source contract — Media Canvas extension (D14). TRULY GENERIC: the kit defines
// the interface and the typed-field/formatter machinery; consumer projects (BR:
// property/guide/service) register concrete providers + formatters WITHOUT
// touching the kit (modular-architecture-first).

/** Open discriminator — projects define their own kinds ('property' | 'guide' | 'service' | …). */
export type SourceKind = string;

/**
 * Field value type — drives the formatter and which inspector property a chip targets.
 * 'longtext' is a read-only content field (see `category`), never a numeric/interpolated value.
 * 'location' resolves to a { lat, lng } point (from SourceProvider.location()).
 * 'region'   resolves to a GeoJSON Polygon ring (from SourceProvider.region()).
 */
export type FieldType =
	| 'text'
	| 'longtext'
	| 'number'
	| 'currency'
	| 'area'
	| 'image'
	| 'list'
	| 'date'
	| 'url'
	| 'location'
	| 'region';

/** A field a provider exposes — either a bindable chip or a read-only content row. */
export interface FieldDescriptor {
	/** Token path into the source snapshot: 'rooms.bedrooms' | 'price' | 'description'. */
	token: string;
	/** Display label: 'Beds' | 'Price' | 'Description'. */
	label: string;
	/** Value type — selects the formatter. */
	type: FieldType;
	/** 'field' = bindable drag-chip · 'content' = read-only, copyable long text (never draggable). */
	category: 'field' | 'content';
	/** Formatter hint: 'currency:THB' | 'area:rai' | 'date:medium'. */
	format?: string;
	/** Default display when the active record has no value for this field. */
	fallback?: string;
	/** Known raw codes for an enum/list field — used by the Inspector's alias editor to
	 *  render one editable row per value (e.g. ['sale','lease'] for transactionType).
	 *  Relevant only when `format` starts with 'enum:'. */
	enumValues?: string[];
}

/** A picker row returned by `SourceProvider.list`. */
export interface SourceRecord {
	id: string;
	title: string;
	subtitle?: string;
	thumb?: string;
}

/**
 * One page of picker rows (D21). `nextCursor` is an opaque, provider-defined token
 * (e.g. an offset or a keyset key); null/undefined → no more pages. The UI registers
 * a load-more that re-calls `list`/`listByCategory` with this cursor.
 */
export interface SourcePage {
	records: SourceRecord[];
	nextCursor?: string | null;
}

/** A grouping a provider exposes (D21): guide/service/news categories, property collections. */
export interface SourceCategory {
	id: string;
	label: string;
	/** Optional record count for the group header. */
	count?: number;
}

/**
 * A boolean filter toggle a provider exposes for its picker (D22). The picker renders
 * one checkbox per facet and passes the checked map as `list`'s `filters` argument —
 * so the generic picker stays project-agnostic (e.g. BR declares `{ key: 'drafts' }`
 * for the published/drafts toggle; the kit never hardcodes a domain filter).
 */
export interface SourceFacet {
	key: string;
	label: string;
	/** Default checked state (omitted → false). */
	default?: boolean;
}

/** Checked facet values passed to `list`/`listByCategory` (facet key → checked). */
export type SourceFilters = Record<string, boolean>;

/** Resolved record data: token → raw value (may be nested; bindings read dotted paths). */
export type SourceSnapshot = Record<string, unknown>;

/**
 * One resolved image in a snapshot's `images.<category>` array (E2 — gallery binding).
 * `url` is the bind target referenced by tokens `images.<category>.<index>.url`; `thumb`
 * is an optional smaller variant the Media panel grid renders; `alt`/`color` are
 * presentation hints. Indices are POSITIONAL and deliberate: a layer bound to
 * `images.gallery.2.url` shows the 3rd gallery image of whatever record is attached, and
 * the link persists (rendering an empty slot) when the record has fewer images — which is
 * what makes a document a template (duplicate + swap source → every slot re-fills).
 */
export interface ResolvedImage {
	url: string;
	thumb?: string;
	/**
	 * Render-role variants (additive, all optional). A provider MAY offer a
	 * right-sized URL per context so each surface loads only what it needs:
	 * grid/picker → `thumbnail`, on-screen artboard → `medium`, export → `original`
	 * (full resolution). A provider that omits them (maps, branding, a legacy
	 * document) keeps working — `pickImageVariant` falls back to `url`. `url`/`thumb`
	 * stay the bind-target identity + legacy alias, so existing token bindings
	 * (`images.<cat>.<idx>.url`) are unaffected.
	 */
	thumbnail?: string;
	medium?: string;
	original?: string;
	alt?: string;
	color?: string;
}

/** Which render context an image is being resolved for — the kit's image-role
 *  vocabulary, source-agnostic. Consumers map each role to a real URL via the
 *  provider's ResolvedImage variants (see `pickImageVariant`). */
export type ImageRenderRole = 'thumbnail' | 'medium' | 'original';

/**
 * Pick the URL for a render role from a ResolvedImage, with a GRACEFUL fallback so
 * a provider that offers no role variants (or an old document) still resolves —
 * the result is always a usable URL, never undefined:
 *   thumbnail → thumbnail ?? thumb (legacy) ?? medium ?? url
 *   medium    → medium ?? url
 *   original  → original ?? url
 */
export function pickImageVariant(img: ResolvedImage, role: ImageRenderRole): string {
	switch (role) {
		case 'thumbnail':
			return img.thumbnail ?? img.thumb ?? img.medium ?? img.url;
		case 'medium':
			return img.medium ?? img.url;
		case 'original':
			return img.original ?? img.url;
	}
}

/**
 * An image grouping a provider offers WITHIN a resolved record (E2 — Media panel).
 * DISTINCT from `categories()`, which groups records in the attach picker: these group
 * the images of ONE resolved record (BR property → 'cover' + 'gallery'). The resolved
 * snapshot carries a matching `images.<id>: ResolvedImage[]`; the panel renders one group
 * per category plus an 'All' view, and a thumbnail click binds `images.<id>.<index>.url`.
 */
export interface ImageCategory {
	id: string;
	label: string;
	/** Functional role of the group (M-2). 'map' marks derived map imagery the Map panel
	 *  lists per source — the UI selects on this role, NEVER on a literal group id, so
	 *  any provider can opt a group in. Absent → a plain photo group (Media panel only). */
	role?: 'map';
}

/**
 * One selectable display knob a {@link FormatterDescriptor} exposes to the Inspector
 * (e.g. 'display' | 'notation' | 'decimals' | 'unit'). The Inspector renders one control
 * per option; the chosen value is written into {@link Binding.formatOptions} under `key`
 * and read back by the descriptor's `format()`.
 */
export interface FormatOption {
	/** Key written into `Binding.formatOptions` and read by `format()` (e.g. 'notation'). */
	key: string;
	/** Human knob name for the Inspector (e.g. 'Notation'). */
	label: string;
	/** Allowed values for this knob (e.g. ['full', 'short']). */
	choices: string[];
	/** Default choice — the effective value when the binding sets no override for `key`. */
	default: string;
	/** Inspector preview value for THIS knob's choices (S-FORMAT INC-0b), defaulting to the
	 *  descriptor's {@link FormatterDescriptor.sample}. Set it when a knob's effect only shows at
	 *  a particular magnitude — e.g. a currency NOTATION knob needs a large value (฿7,000,000 →
	 *  "฿7M") while the descriptor's base sample is small (฿123) so the display/decimals previews
	 *  read cleanly. `number | string` mirrors {@link FormatterDescriptor.valueType}. */
	previewSample?: number | string;
}

/** Per-knob option values (key → chosen value). The descriptor defaults merged with the
 *  binding's `formatOptions` (override wins) are passed to `format()` as this map. */
export type FormatOpts = Record<string, string>;

/**
 * A named, options-aware value renderer. Consumers register one per format-hint name
 * ('currency' | 'area' | …); the kit's resolver selects it by the `format` hint's name,
 * merges the descriptor defaults with the binding's `formatOptions` (override wins), and
 * calls `format`. `arg` is the part after ':' in the hint ('currency:THB' → 'THB'); `value`
 * is the raw token coerced per {@link FormatterDescriptor.valueType} — `Number(raw)` for numeric
 * descriptors (default), or the raw `String(raw)` for `valueType: 'string'` (text/date/enum).
 */
export interface FormatterDescriptor {
	/** Format-hint name this descriptor answers to (matches the registry key). */
	name: string;
	/** The knobs the Inspector renders for a binding that uses this formatter. */
	options: FormatOption[];
	/** Representative value for the Inspector's preview labels. */
	sample: number | string;
	/** Raw-token coercion at resolve time (S-FORMAT INC-A): 'number' (default) → Number(raw);
	 *  'string' → the raw String(raw), unlocking text/date/enum formatters. Numeric descriptors
	 *  (currency/area) omit it and stay byte-identical. */
	valueType?: 'number' | 'string';
	format(value: number | string, opts: FormatOpts, arg?: string): string;
}

/** Pluggable registry keyed by format name: { currency, area, date, … }. Consumers populate it. */
export type FormatterRegistry = Record<string, FormatterDescriptor>;

/**
 * Data-source adapter. The kit depends on THIS, never on a concrete API.
 * Consumers implement one per kind and register it with the editor.
 */
export interface SourceProvider {
	kind: SourceKind;
	label: string;
	/** Bindable chips AND read-only content rows for this kind. */
	fields(): FieldDescriptor[];
	/** Token → raw value map for a chosen record. Re-called on document open (live re-resolve). */
	resolve(refId: string): Promise<SourceSnapshot>;
	/**
	 * One page of picker rows for the attach modal (D21). `cursor` is omitted for the first
	 * page and set to a prior page's `nextCursor` for load-more. `query` is the search string.
	 * `filters` carries the checked facet values (D22) — undefined when the provider has no facets.
	 */
	list(query: string, cursor?: string, filters?: SourceFilters): Promise<SourcePage>;
	/**
	 * Optional (D22). Boolean filter toggles the picker renders as checkboxes (e.g. drafts).
	 * Absence → no checkboxes. The checked map is passed back via `list`/`listByCategory` `filters`.
	 */
	facets?(): SourceFacet[];
	/**
	 * Optional (D21). When present, the rail renders category groups; absence → flat list only.
	 * BR: guide/service/news categories, property collections.
	 */
	categories?(): Promise<SourceCategory[]>;
	/** Optional (D21). Paginated rows within one category; required iff `categories` is implemented. */
	listByCategory?(categoryId: string, cursor?: string, filters?: SourceFilters): Promise<SourcePage>;
	/**
	 * Optional (E2). Image groupings this provider offers within a resolved record —
	 * BR property: [{ id: 'cover', label: 'Cover' }, { id: 'gallery', label: 'Gallery' }].
	 * Absence → the Media panel shows no source-image listing for this kind. The resolved
	 * snapshot carries the matching `images.<id>: ResolvedImage[]` arrays; a thumbnail click
	 * binds `images.<id>.<index>.url` onto the selected image layer.
	 */
	imageCategories?(): ImageCategory[];
	/**
	 * Optional (Decision #0297). Returns a { lat, lng } point for the active record,
	 * or null when the record has the capability but no location data is stored.
	 * Absence of the method → this provider never exposes location.
	 * The map editor queries this to decide whether the 'location' resolve token
	 * is available AND whether the current record has a value for it.
	 * FieldType 'location' is the corresponding chip type in fields().
	 */
	location?(refId: string): Promise<{ lat: number; lng: number } | null>;
	/**
	 * Optional (Decision #0297). Returns a GeoJSON polygon coordinate ring for the
	 * active record (e.g. a land parcel boundary), or null when the record has the
	 * capability but no region is stored.
	 * Absence of the method → this provider never exposes a region.
	 * The map editor queries this to decide whether the 'region' resolve token
	 * is available AND whether the current record has a value for it.
	 * FieldType 'region' is the corresponding chip type in fields().
	 * Coordinate ordering: [lng, lat][] (GeoJSON Polygon ring — matches MapPolygon
	 * in @sbx/ui-map and pkg/address.GeoJSONPolygon).
	 */
	region?(refId: string): Promise<Array<[number, number]> | null>;
}
