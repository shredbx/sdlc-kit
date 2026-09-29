import type { RegisteredHeader } from './types';

// Same mechanism as renderers/registry.ts (docs/proposals/bos-page-designer.md §5): each header
// preset lives in its own folder exporting a `registeredChrome`, discovered via `import.meta.glob` —
// no hand-written switch, no generated file.
const modules = import.meta.glob<{ registeredChrome: RegisteredHeader }>('./*/index.ts', {
	eager: true
});

export const headerPresets: RegisteredHeader[] = Object.values(modules).map((m) => m.registeredChrome);

const byId = new Map(headerPresets.map((p) => [p.id, p]));

export function getHeaderPreset(id: string): RegisteredHeader | undefined {
	return byId.get(id);
}
