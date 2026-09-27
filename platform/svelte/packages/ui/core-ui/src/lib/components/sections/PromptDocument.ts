/**
 * Prompt document model + renderer (the lean v1 contract).
 *
 * A prompt "document" is a FLAT, ordered list of sections — each a title + body of
 * plain text. It renders to a single string in one of two markup formats (XML
 * elements or Markdown headings). This is the shared, presets-free, nesting-free atom
 * the `PromptDocumentEditor` section edits.
 *
 * The renderer mirrors the authoritative python prompt renderer
 * (`sbx_assistant.prompt.renderer`): escape happens only at the format boundary; the
 * element form `<name>…</name>` carries no attributes (no attribute-injection class).
 * The python renderer assumes clean section names; this editor accepts FREE-FORM
 * titles, so the xml tag is slugified to a safe element name (markdown keeps the title
 * verbatim — it is prose, not structure). When the engine renders the same Document the
 * canonical name→tag rule must be unified with python (flagged for the config-contract
 * work). That python model is the richer, recursive superset — v1 keeps only
 * `{ name, text }` and drops `children` / `link` / `based_on`.
 *
 * Pure module (no Svelte, no DOM) so it unit-tests in plain node and any product can
 * call `renderPromptDocument` server-side (e.g. to materialise a system prompt).
 */

/**
 * One section of a prompt document: a title (`name`) and a body (`text`).
 *
 * `name` is the element/heading label; `text` is the free-form body. A `hidden`
 * section is kept in the list (so toggling it back loses nothing) but excluded from
 * the rendered output — it never reaches the wire. The field is optional so a plain
 * `{ name, text }` literal is a valid block.
 */
export interface PromptBlock {
	/** The section title — the Markdown heading verbatim; slugified to a safe XML element name. */
	name: string;
	/** The section body. Rendered verbatim (XML-escaped for the xml format). */
	text: string;
	/** When true the section is excluded from the rendered document (still editable). */
	hidden?: boolean;
}

/** The two markup formats a prompt document renders to. */
export type PromptDocumentFormat = 'xml' | 'markdown';

/**
 * Escape the three XML-structural characters so a section's body cannot forge markup
 * structure. This matches the authoritative renderer exactly (`& < >` only — a
 * reversible, structurally-safe set; quotes are not escaped because the element form
 * has no attributes). Order matters: `&` is replaced first so the entities the later
 * replacements introduce are not re-escaped.
 */
export function escapeXml(value: string): string {
	return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

/**
 * Normalise a free-form section title into a safe XML element name. The python renderer
 * assumes clean (identifier-like) names; this editor accepts ANY title, so the body's
 * structural safety would be undone by a title like `a><b` or `role onload="x"`. We
 * slugify to `[A-Za-z0-9_-]`, collapse runs to `-`, trim edge `-`, and fall back to
 * `section` when nothing survives (e.g. a body-only section). A clean name is unchanged,
 * so this is a no-op for the python-parity case and only diverges where the verbatim form
 * would be invalid XML anyway.
 */
function xmlTagName(name: string): string {
	return (
		name
			.trim()
			.replace(/[^A-Za-z0-9_-]+/g, '-')
			.replace(/^-+|-+$/g, '') || 'section'
	);
}

/**
 * Render a flat prompt document to a single string.
 *
 * - `xml`      → `<name>text</name>` per section; the title is slugified to a safe
 *                element name, the body is XML-escaped (`& < >`). Blank-line separated.
 * - `markdown` → `## name` then a blank line then the verbatim body. Sections
 *                separated by a blank line. (Markdown is fidelity-preserving prose —
 *                the body passes through unescaped so the model sees literal text.)
 *
 * Hidden sections, and sections whose title AND body are both empty, are skipped — a
 * seeded-but-untouched section never bloats the output. Only emptiness is judged on a
 * trim; the body itself is never trimmed (author whitespace is intentional).
 */
export function renderPromptDocument(blocks: PromptBlock[], fmt: PromptDocumentFormat): string {
	const parts: string[] = [];

	for (const block of blocks) {
		if (block.hidden) continue;
		const hasTitle = block.name.trim().length > 0;
		const hasBody = block.text.trim().length > 0;
		if (!hasTitle && !hasBody) continue;

		if (fmt === 'xml') {
			const tag = xmlTagName(block.name);
			parts.push(`<${tag}>${escapeXml(block.text)}</${tag}>`);
		} else {
			parts.push(hasBody ? `## ${block.name}\n\n${block.text}` : `## ${block.name}`);
		}
	}

	return parts.join('\n\n');
}
