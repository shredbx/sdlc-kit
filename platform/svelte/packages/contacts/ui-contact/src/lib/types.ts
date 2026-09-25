// @sbx/ui-contact — public contract. Contact-native projection of the picker, modeled
// on @sbx/canvas-kit's Source contract discipline (minimal record + provider adapter)
// but DECOUPLED from canvas/media: this package never imports canvas-kit. Consumers
// (BR transactions/inquiries, calendar EventEditor) map their domain record to
// ContactPickerRecord and implement ContactProvider over their own API.

import type { Snippet } from 'svelte';

/**
 * A pickable contact row — the package's minimal projection. Consumers build this from
 * their domain record (e.g. BR's ContactRecord via buildContactDisplayName / contactInitials).
 */
export interface ContactPickerRecord {
	id: string;
	/** Full display name (consumer-built). */
	name: string;
	/** Secondary line — e.g. "+66 81 234 5678". */
	phone?: string;
	/** Category codes (≥1, ordered; [0] is primary) — drives the trailing badge + color dot. */
	categoryCodes: string[];
	/** Resolved photo URL; absent → initials avatar. */
	photoUrl?: string | null;
	/** 1–2 char initials fallback when there is no photo. */
	initials?: string;
}

/** A category group for the rail + the trailing badge label. */
export interface ContactCategory {
	code: string;
	label: string;
	/** Optional record count for the rail header. */
	count?: number;
}

/**
 * One page of rows. `nextCursor` is an opaque, provider-defined token (e.g. an offset);
 * null/undefined → no more pages. The picker re-calls list/listByCategory with it for load-more.
 */
export interface ContactPage {
	records: ContactPickerRecord[];
	nextCursor?: string | null;
}

/**
 * Data adapter — the package depends on THIS, never on a concrete API. Consumers implement
 * one over their contacts endpoint and pass it to the picker (modular-architecture-first).
 */
export interface ContactProvider {
	/** Flat search across all categories. `cursor` paginates (load-more). */
	list(query: string, cursor?: string): Promise<ContactPage>;
	/** Optional. When present the rail renders category groups; absence → flat list only. */
	categories?(): Promise<ContactCategory[]>;
	/** Optional. Paginated rows within one category; required iff `categories` is implemented. */
	listByCategory?(code: string, cursor?: string): Promise<ContactPage>;
	/** Optional. Resolve a category code → label for the trailing badge (falls back to the code). */
	categoryLabel?(code: string): string;
}

/** 'single' = assign (radio + confirm footer) · 'multi' = checkbox-select + "Add (N)" footer. */
export type ContactPickerMode = 'single' | 'multi';

/** Public prop contract for ContactPickerModal. */
export interface ContactPickerProps {
	/** The data-source adapter — the picker drives ONLY this contract. */
	provider: ContactProvider;
	/** Modal visibility (bindable). */
	open?: boolean;
	/** 'single' (default) = assign one (radio + confirm) · 'multi' = add many (checkbox + count). */
	mode?: ContactPickerMode;
	/** Scope to one category: the rail hides, and inline-create injects this code. */
	category?: string;
	/**
	 * Soft default category — opens the rail ON this category but keeps it BROWSABLE (the rail
	 * stays visible; the user can click "All" or any other group to broaden). A default filter,
	 * NOT a lock — unlike `category`, search and create are not hard-scoped to it. Ignored when
	 * `category` is set (a hard lock wins). Use when a manager should default to a role but still
	 * be able to assign anyone (e.g. a deal-party picker).
	 */
	defaultCategory?: string;
	/** Footer label in single mode (default 'Assign'). */
	confirmLabel?: string;
	/** Pre-selected record ids — multi pre-checks each; single pre-highlights the first. */
	selectedIds?: string[];
	/** A contact was chosen (single mode). */
	onpick?: (record: ContactPickerRecord) => void;
	/** Selection confirmed (multi mode) — ordered as picked. */
	onconfirm?: (records: ContactPickerRecord[]) => void;
	/** Modal dismissed without a choice. */
	onclose?: () => void;
	/**
	 * Consumer-supplied create form — keeps domain form deps (dictionaries, social, address)
	 * OUT of this package. The picker renders a "+ Create new" affordance that REPLACES the
	 * picker overlay with this snippet (the snippet owns its own surface, e.g. a full-screen
	 * card — so the rich form gets real width instead of the narrow picker panel). The snippet
	 * POSTs and calls `onCreated(record)` to auto-select, or `cancel()` to return to the list.
	 */
	createForm?: Snippet<
		[{ onCreated: (record: ContactPickerRecord) => void; cancel: () => void; category?: string }]
	>;
}
