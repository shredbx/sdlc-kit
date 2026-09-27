import type { RegisteredRenderer } from './types';

// The renderer registry (docs/proposals/bos-page-designer.md §5): each renderer lives in its own
// folder exporting a `registeredRenderer`, discovered here via Vite's `import.meta.glob` — the
// build-time equivalent of agent_framework's `core/registry.py` filesystem walk (Python can
// importlib an arbitrary path at runtime; a bundled browser app can't, but Vite's glob resolves the
// same "add a folder, restart" shape at build/dev time, re-scanning automatically). No hand-written
// switch/if-chain and no generated registry file to keep in sync — the glob *is* the registry.
const modules = import.meta.glob<{ registeredRenderer: RegisteredRenderer }>('./*/index.ts', {
	eager: true
});

export const renderers: RegisteredRenderer[] = Object.values(modules).map((m) => m.registeredRenderer);

const byId = new Map(renderers.map((r) => [r.id, r]));

export function getRenderer(id: string): RegisteredRenderer | undefined {
	return byId.get(id);
}
