import type { MapProvider, MapSurface } from './types.js';

const WORKSPACE_DEFAULT: Record<MapSurface, MapProvider> = {
	admin: 'google',
	public: 'mapbox'
};

interface ResolveProviderInput {
	provider?: MapProvider;
	surface?: MapSurface;
	envProvider?: string;
}

// Resolution order:
//   1. explicit provider prop
//   2. env PUBLIC_MAP_PROVIDER (only when surface is unset — env is per-surface)
//   3. workspace default per surface
//   4. fall back to 'google'
export function resolveProvider(input: ResolveProviderInput): MapProvider {
	if (input.provider) return input.provider;
	if (!input.surface && isProvider(input.envProvider)) return input.envProvider as MapProvider;
	if (input.surface) return WORKSPACE_DEFAULT[input.surface];
	return 'google';
}

function isProvider(v: unknown): v is MapProvider {
	return v === 'google' || v === 'mapbox';
}

interface ResolveKeyInput {
	provider: MapProvider;
	googleKey?: string;
	mapboxToken?: string;
}

export function resolveApiKey(input: ResolveKeyInput): string | undefined {
	if (input.provider === 'google') return input.googleKey || undefined;
	if (input.provider === 'mapbox') return input.mapboxToken || undefined;
	return undefined;
}

// SSR-safe env read. Returns undefined during SSR; called from onMount only.
export function readBrowserEnv(): {
	provider: string | undefined;
	googleKey: string | undefined;
	mapboxToken: string | undefined;
} {
	const env = (import.meta as ImportMeta & { env?: Record<string, string | undefined> }).env ?? {};
	return {
		provider: env.PUBLIC_MAP_PROVIDER,
		googleKey: env.PUBLIC_GOOGLE_MAPS_API_KEY,
		mapboxToken: env.PUBLIC_MAPBOX_TOKEN
	};
}
