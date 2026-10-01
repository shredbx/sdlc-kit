// Data-binding model — Media Canvas extension (D9).
// A binding maps one layer property to a token in an attached source's snapshot.
// `resolveBindings(layer, snapshot, formatters)` substitutes bound values
// (formatted by the binding's `format` hint, falling back to `fallback`).

export interface Binding {
	/** Layer property the bound value writes to: 'content' | 'src' | property path. */
	property: string;
	/** Which attached source this reads from (Document.sources[].alias, e.g. 'primary'). */
	sourceAlias: string;
	/** Field token into the source snapshot (dotted path): 'price' | 'rooms.bedrooms' | 'coverImage.url'. */
	token: string;
	/** Formatter hint: 'currency:THB' | 'area:rai' | 'date:medium'. Empty → raw String(value). */
	format?: string;
	/**
	 * Per-knob overrides for the binding's formatter (S-FORMAT). Keyed by a
	 * `FormatterDescriptor` option key (e.g. `{ notation: 'short', display: 'code' }`).
	 * Merged OVER the descriptor's defaults at resolve time (override wins); absent ⇒ the
	 * descriptor defaults apply, so the output is byte-identical to a binding without this field.
	 */
	formatOptions?: Record<string, string>;
	/**
	 * Manual fine-tune (D20). When set (non-null), this verbatim value WINS over the live
	 * source value — used as-is, NOT re-formatted (the user already typed the final string;
	 * for an image binding it is the chosen image URL). A **Restore** action clears it
	 * (set to undefined) to fall back to the live `token → format` value. Persisted with the
	 * document; the live snapshot is not. Resolve precedence: override → token→formatter → fallback.
	 */
	override?: string;
	/** Shown when the resolved value is missing/empty — never undefined, so layout never breaks. */
	fallback?: string;
	/** true → this binding re-resolves when a template spawns a new document (D14 / §6). */
	placeholder?: boolean;
}
