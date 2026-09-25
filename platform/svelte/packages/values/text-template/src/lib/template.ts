// @sbx/text-template — the headless, framework-agnostic core for token-interpolating text
// templates. A template is an ordered list of literal-text and TOKEN nodes; a token names a
// dotted source path ('property.price') and optionally carries display formatting. The package
// owns parsing / serialising / resolving; the CONSUMER owns how a token resolves to a string
// (the TokenResolver protocol) and which tokens exist (the TokenCatalog protocol). So the same
// composer drives the Media Canvas today and post/content tools next — loose coupling via
// protocols, never a dependency on any one consumer (modular-architecture-first).

/** One node of a parsed template. */
export type TemplateNode =
	| { type: 'text'; text: string }
	| {
			type: 'token';
			/** Dotted source path, e.g. 'property.address.country'. */
			token: string;
			/** Optional display-format hint ('currency:THB' | 'enum:transactionType'). When absent,
			 *  the consumer's resolver falls back to the field's declared format. */
			format?: string;
			/** Optional per-knob format options (mirrors a canvas Binding's formatOptions). */
			formatOptions?: Record<string, string>;
	  };

/** A template = an ordered list of nodes (the canonical, persisted form). */
export type Template = TemplateNode[];

/** A token a {@link TokenCatalog} offers the composer's insert picker. */
export interface TemplateToken {
	token: string;
	/** Human label for the picker + the badge title ('Price', 'Country'). */
	label: string;
	/** The field's declared format hint, copied onto an inserted token node. */
	format?: string;
	/** OPTIONAL grouping key for the picker / @-menu (e.g. a source label like
	 *  'secondary · Wellness…'). Consumer-defined; the composer groups suggestions by it
	 *  and falls back to a flat list when absent. Pure metadata — never affects resolution. */
	group?: string;
}

/**
 * CONSUMER-OWNED resolution protocol. The package depends on THIS, never on a concrete data
 * source. Canvas implements it from the bound source snapshot + formatter registry; a posts
 * tool from its own record. `resolve` returns the FINAL display string for a token (already
 * formatted); '' for a missing value (so a template degrades gracefully, never shows undefined).
 */
export interface TokenResolver {
	resolve(token: string, format?: string, formatOptions?: Record<string, string>): string;
}

/** CONSUMER-OWNED catalog protocol — the insertable tokens (for the composer's picker + the
 *  badge titles). Optional: a composer can run resolve-only (no insert picker). */
export interface TokenCatalog {
	tokens(): TemplateToken[];
}

// Token grammar: [dotted.path]. A path is one or more word chars / dots. Anything that is not a
// well-formed [path] stays literal text — so ordinary prose with stray brackets never breaks.
const TOKEN_RE = /\[([A-Za-z0-9_.]+)\]/g;

/**
 * Parse a flat template string into nodes: "Hi [property.title]!" →
 * [text "Hi ", token property.title, text "!"]. Tokens carry no format (the consumer's resolver
 * supplies the field default); the structured/per-token format is set later via the composer.
 * Adjacent literal runs collapse into one text node; an empty source → [].
 */
export function parseTemplate(src: string): Template {
	const nodes: Template = [];
	let last = 0;
	let match: RegExpExecArray | null;
	// Fresh lastIndex each call — TOKEN_RE is module-level + global (stateful otherwise).
	TOKEN_RE.lastIndex = 0;
	while ((match = TOKEN_RE.exec(src)) !== null) {
		if (match.index > last) nodes.push({ type: 'text', text: src.slice(last, match.index) });
		nodes.push({ type: 'token', token: match[1] });
		last = match.index + match[0].length;
	}
	if (last < src.length) nodes.push({ type: 'text', text: src.slice(last) });
	return nodes;
}

/**
 * Serialize nodes back to a flat template string (tokens as [token]) — the inverse of
 * {@link parseTemplate} for the text+token structure. Per-token format lives on the nodes, not
 * in the flat string, so a serialize→parse round-trip preserves text + tokens (not format).
 */
export function serializeTemplate(template: Template): string {
	return template.map((node) => (node.type === 'text' ? node.text : `[${node.token}]`)).join('');
}

/**
 * Resolve a template to its final display string via the consumer's {@link TokenResolver}.
 * Each token node becomes `resolver.resolve(token, format, formatOptions)`; text nodes pass
 * through. A resolver returning '' contributes nothing — the template degrades gracefully.
 */
export function resolveTemplate(template: Template, resolver: TokenResolver): string {
	return template
		.map((node) =>
			node.type === 'text' ? node.text : resolver.resolve(node.token, node.format, node.formatOptions)
		)
		.join('');
}

/** The DISTINCT token paths a template references, in first-seen order — for dependency
 *  tracking (which source fields a template needs) + re-resolution on snapshot change. */
export function templateTokens(template: Template): string[] {
	const seen: string[] = [];
	for (const node of template) {
		if (node.type === 'token' && !seen.includes(node.token)) seen.push(node.token);
	}
	return seen;
}
