// Pure helpers for the prompt Edit surface — no store/DOM/API, unit-testable in isolation.

/** One segment of tokenized section content: literal prose or a `$variable` reference. */
export type ContentSegment =
	| { kind: 'text'; value: string }
	| { kind: 'arg'; name: string };

const ARG_RE = /\$([A-Za-z_][A-Za-z0-9_]*)/g;

/**
 * Split section content into prose + `$arg` segments so the host can render each `$name` as an
 * ArgToken chip while leaving prose untouched. Order-preserving; empty input → `[]`. A bare `$`
 * (no identifier after it) stays literal text.
 */
export function tokenizeArgs(text: string): ContentSegment[] {
	if (!text) return [];
	const out: ContentSegment[] = [];
	let last = 0;
	for (const m of text.matchAll(ARG_RE)) {
		const start = m.index ?? 0;
		if (start > last) out.push({ kind: 'text', value: text.slice(last, start) });
		out.push({ kind: 'arg', name: m[1] });
		last = start + m[0].length;
	}
	if (last < text.length) out.push({ kind: 'text', value: text.slice(last) });
	return out;
}

/** The distinct variable names a piece of content references, in first-seen order. */
export function argsInContent(text: string): string[] {
	const seen = new Set<string>();
	for (const seg of tokenizeArgs(text)) {
		if (seg.kind === 'arg') seen.add(seg.name);
	}
	return [...seen];
}

/** The provenance ref a preset pill points at (mirrors `PresetRef`). */
export interface PresetLike {
	section: string;
	preset: string;
}

/** A block node's provenance, as far as the pill needs it. */
export interface ProvenanceLike {
	link?: PresetLike | null;
	based_on?: PresetLike | null;
}

export type PresetStatus = 'linked' | 'edited' | 'custom';

/** A section as the Edit surface renders it — the host maps a real DocSection tree into these. */
export interface SectionNode {
	name: string;
	description?: string;
	content: string;
	status: PresetStatus;
	presetName?: string;
	/** Distinct `$vars` referenced by content — drives the "uses" caption. */
	uses?: string[];
	children?: SectionNode[];
}

/**
 * Derive the pill state from a block's link/based_on (types.ts semantics):
 *   link set            → linked (content mirrors the preset live)
 *   based_on only       → edited (detached from a preset, now diverged)
 *   neither             → custom (authored inline)
 */
export function presetStatus(node: ProvenanceLike): PresetStatus {
	if (node.link) return 'linked';
	if (node.based_on) return 'edited';
	return 'custom';
}

/** The preset NAME a pill shows for linked/edited (undefined for custom). */
export function presetName(node: ProvenanceLike): string | undefined {
	return node.link?.preset ?? node.based_on?.preset ?? undefined;
}

/** Whether a content string reads as a JSON literal (array/object) — renders as a code box. */
export function looksLikeJson(text: string): boolean {
	const t = text.trim();
	if (!(t.startsWith('{') || t.startsWith('['))) return false;
	return t.endsWith('}') || t.endsWith(']');
}
