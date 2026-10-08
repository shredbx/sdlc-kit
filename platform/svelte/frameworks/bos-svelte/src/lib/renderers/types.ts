import type { Component } from 'svelte';

/** One field of a renderer's inline-content schema — enough for the admin's "Add section" form to
 * build a generic form without knowing the renderer ahead of time. Mirrors agent_framework's
 * RegisteredAgent shape in spirit (an id, the real thing, and a bit of metadata), adapted to what a
 * Svelte content renderer actually needs (see agent_framework/core/types.py, RegisteredAgent). */
export interface SectionField {
	key: string;
	label: string;
	type: 'text' | 'markdown';
}

/** One entry in the renderer registry (docs/proposals/bos-page-designer.md §5). `shape` is the
 * bos-constructor.md §4.3 compatibility concept — what kind of content this renderer accepts — kept
 * to 'single' for now (a renderer for a whole list-shaped source is roadmap M3 territory). */
export interface RegisteredRenderer {
	id: string;
	label: string;
	shape: 'single';
	fields: SectionField[];
	component: Component<{ content: Record<string, unknown> }>;
}

/** A section as stored: a `kind` (the renderer id) plus its typed payload nested under a key
 * EQUAL to kind — {"kind":"hero","hero":{...}} — matching the Go-side wire shape exactly
 * (platform/go/packages/cms/section.go: "the payload key EQUALS the discriminator, so the
 * Svelte renderer reads section[section.kind] directly"). `hidden` (optional, default false)
 * marks a section authored-but-not-rendered — the public loader skips it but the admin still
 * lists it for editing. */
export interface SectionData {
	kind: string;
	hidden?: boolean;
	[key: string]: unknown;
}
