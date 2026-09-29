import type { RegisteredLayout } from './types';

// The layout registry — same mechanism as the renderer and chrome registries
// (docs/proposals/bos-page-designer.md §5 / bos-site-chrome.md §4): one folder per
// preset, exporting a `registeredLayout`, discovered via Vite's `import.meta.glob`.
const modules = import.meta.glob<{ registeredLayout: RegisteredLayout }>('./*/index.ts', {
	eager: true
});

export const layouts: RegisteredLayout[] = Object.values(modules).map((m) => m.registeredLayout);

const byId = new Map(layouts.map((l) => [l.id, l]));

const DEFAULT_LAYOUT_ID = 'default';

/** Resolves a layout id, falling open to "default" for an unrecognized id — the same
 * fails-open rule the renderer registry uses for an unknown section kind. */
export function getLayout(id: string): RegisteredLayout {
	return byId.get(id) ?? byId.get(DEFAULT_LAYOUT_ID)!;
}
