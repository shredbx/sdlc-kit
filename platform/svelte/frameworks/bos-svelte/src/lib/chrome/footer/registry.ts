import type { RegisteredFooter } from './types';

// Same mechanism as header/registry.ts and renderers/registry.ts — see there for the rationale.
const modules = import.meta.glob<{ registeredChrome: RegisteredFooter }>('./*/index.ts', {
	eager: true
});

export const footerPresets: RegisteredFooter[] = Object.values(modules).map((m) => m.registeredChrome);

const byId = new Map(footerPresets.map((p) => [p.id, p]));

export function getFooterPreset(id: string): RegisteredFooter | undefined {
	return byId.get(id);
}
