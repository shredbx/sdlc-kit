/**
 * FixtureLoader - Bridge between YAML fixtures and TypeScript
 *
 * Provides utilities for loading fixture data from YAML files for:
 * - Component previews in /components page
 * - E2E tests with Playwright
 * - Development and debugging
 *
 * BUILD-ONCE-REUSE pattern: Same fixture data for previews AND tests.
 *
 * @example
 * // In a Svelte component preview
 * import { loadFixture } from './FixtureLoader';
 *
 * const user = await loadFixture('user', 'current-user');
 * const guestUser = await loadFixture('user', 'current-user', 'guest');
 *
 * @example
 * // In Playwright e2e test
 * import { seedFixture, cleanupFixture } from './FixtureLoader';
 *
 * test.beforeEach(async () => {
 *   await seedFixture('notification', 'notification-list');
 * });
 *
 * test.afterEach(async () => {
 *   await cleanupFixture('notification', 'notification-list');
 * });
 */

import { fixtureRegistry, type IFixture, type IResolvedFixture } from './IFixture';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Raw YAML fixture structure (before conversion to IFixture).
 */
export interface YAMLFixture {
	type: 'fixture';
	name: string;
	entity: string;
	purpose: string;
	tags?: string[];
	data: Record<string, YAMLFixtureField>;
	variants?: Record<string, YAMLVariant>;
	usage?: {
		preview?: {
			components?: string[];
			transform?: string;
			additionalProps?: Record<string, unknown>;
		};
		e2e?: {
			tests?: string[];
			seed?: boolean;
			collection?: string;
			cleanup?: boolean;
		};
	};
}

/**
 * YAML fixture field structure.
 */
export interface YAMLFixtureField {
	type: string;
	value?: unknown;
	faker?: string;
	generator?: string;
	ref?: string;
	transform?: string;
	description?: string;
	options?: Record<string, unknown>;
	args?: unknown[];
	path?: string;
	source?: string | unknown;
}

/**
 * YAML variant structure.
 */
export interface YAMLVariant {
	extends?: string;
	description?: string;
	tags?: string[];
	overrides?: Record<string, YAMLFixtureField | unknown>;
}

/**
 * Fixture load options.
 */
export interface FixtureLoadOptions {
	/** Apply variant overrides */
	variant?: string;
	/** Merge with additional props */
	additionalProps?: Record<string, unknown>;
	/** JSONPath transform to extract subset */
	transform?: string;
}

/**
 * Cache entry for loaded fixtures.
 */
interface CacheEntry {
	fixture: IFixture;
	loadedAt: number;
}

// =============================================================================
// STATE
// =============================================================================

/** Fixture cache for performance */
const fixtureCache = new Map<string, CacheEntry>();

/** Cache TTL in ms (5 minutes) */
const CACHE_TTL = 5 * 60 * 1000;

/** Whether we're in browser environment */
const isBrowser = typeof window !== 'undefined';

// =============================================================================
// YAML TO IFIXTURE CONVERSION
// =============================================================================

/**
 * Convert YAML fixture field to IFixtureField.
 */
function convertField(field: YAMLFixtureField) {
	const base = {
		type: field.type,
		description: field.description
	};

	if ('value' in field && field.value !== undefined) {
		return { ...base, value: field.value };
	}

	if (field.faker) {
		return { ...base, faker: field.faker, options: field.options };
	}

	if (field.generator) {
		return { ...base, generator: field.generator, args: field.args };
	}

	if (field.ref) {
		return { ...base, ref: field.ref, path: field.path };
	}

	if (field.transform) {
		return { ...base, transform: field.transform, source: field.source };
	}

	return base;
}

/**
 * Convert YAML fixture to IFixture.
 */
export function convertYAMLToFixture(yaml: YAMLFixture): IFixture {
	const data: Record<string, unknown> = {};

	for (const [key, field] of Object.entries(yaml.data)) {
		data[key] = convertField(field);
	}

	const fixture: IFixture = {
		name: yaml.name,
		entity: yaml.entity,
		purpose: yaml.purpose,
		tags: yaml.tags,
		data: data as IFixture['data'],
		usage: yaml.usage
	};

	if (yaml.variants) {
		fixture.variants = {};
		for (const [name, variant] of Object.entries(yaml.variants)) {
			fixture.variants[name] = {
				extends: variant.extends,
				description: variant.description,
				tags: variant.tags,
				overrides: variant.overrides
			};
		}
	}

	return fixture;
}

// =============================================================================
// LOADING
// =============================================================================

/**
 * Get cache key for fixture.
 */
function getCacheKey(entity: string, name: string): string {
	return `${entity}:${name}`;
}

/**
 * Check if cache entry is valid.
 */
function isCacheValid(entry: CacheEntry): boolean {
	return Date.now() - entry.loadedAt < CACHE_TTL;
}

/**
 * Load fixture from filesystem (server-side) or cache.
 *
 * @param entity - Entity type (e.g., 'user', 'notification')
 * @param name - Fixture name (e.g., 'current-user')
 * @returns Loaded IFixture
 */
export async function loadFixtureDefinition(entity: string, name: string): Promise<IFixture> {
	const cacheKey = getCacheKey(entity, name);

	// Check cache
	const cached = fixtureCache.get(cacheKey);
	if (cached && isCacheValid(cached)) {
		return cached.fixture;
	}

	// Check registry first
	const registered = fixtureRegistry.get(entity, name);
	if (registered) {
		fixtureCache.set(cacheKey, { fixture: registered, loadedAt: Date.now() });
		return registered;
	}

	// In browser, fetch from API
	if (isBrowser) {
		const response = await fetch(`/api/fixtures/${entity}/${name}`);
		if (!response.ok) {
			throw new Error(`Fixture not found: ${entity}/${name}`);
		}

		const yaml = await response.json() as YAMLFixture;
		const fixture = convertYAMLToFixture(yaml);

		// Register and cache
		fixtureRegistry.register(fixture);
		fixtureCache.set(cacheKey, { fixture, loadedAt: Date.now() });

		return fixture;
	}

	// Server-side: Load from filesystem
	throw new Error(
		`Fixture ${entity}/${name} not registered. ` +
		'Use registerFixture() in server load function.'
	);
}

/**
 * Load and resolve fixture with optional variant.
 *
 * This is the main function for getting fixture data.
 *
 * @param entity - Entity type (e.g., 'user', 'notification')
 * @param name - Fixture name (e.g., 'current-user')
 * @param options - Load options (variant, additionalProps, transform)
 * @returns Resolved fixture data
 *
 * @example
 * // Load base fixture
 * const user = await loadFixture('user', 'current-user');
 * // user.data = { id: '123...', name: 'John Doe', ... }
 *
 * @example
 * // Load with variant
 * const guest = await loadFixture('user', 'current-user', { variant: 'guest' });
 * // guest.data = { id: null, name: 'Guest', ... }
 */
export async function loadFixture<T = Record<string, unknown>>(
	entity: string,
	name: string,
	options?: FixtureLoadOptions | string
): Promise<IResolvedFixture<T>> {
	// Support passing variant as string shorthand
	const opts: FixtureLoadOptions =
		typeof options === 'string' ? { variant: options } : options ?? {};

	// Load definition
	await loadFixtureDefinition(entity, name);

	// Resolve with registry
	const resolved = await fixtureRegistry.resolve<T>(entity, name, opts.variant);

	// Apply additional props
	if (opts.additionalProps) {
		resolved.data = { ...resolved.data, ...opts.additionalProps } as T;
	}

	// Apply transform (JSONPath) - simplified implementation
	if (opts.transform) {
		const path = opts.transform.split('.');
		let data: unknown = resolved.data;
		for (const key of path) {
			if (data && typeof data === 'object' && key in data) {
				data = (data as Record<string, unknown>)[key];
			}
		}
		resolved.data = data as T;
	}

	return resolved;
}

/**
 * Load fixture data directly (shorthand for loadFixture(...).data).
 *
 * @example
 * const messages = await loadFixtureData('chat', 'conversation', 'messages');
 */
export async function loadFixtureData<T = unknown>(
	entity: string,
	name: string,
	variant?: string
): Promise<T> {
	const resolved = await loadFixture(entity, name, { variant });
	return resolved.data as T;
}

// =============================================================================
// COMPONENT PREVIEW HELPERS
// =============================================================================

/**
 * Load fixture formatted for component preview.
 *
 * Looks up the fixture's usage.preview config and applies it.
 *
 * @example
 * // In presenter or preview component
 * const props = await loadPreviewFixture('user', 'current-user');
 * // Returns { ...user.data, size: 'md' } (with additionalProps merged)
 */
export async function loadPreviewFixture<T = Record<string, unknown>>(
	entity: string,
	name: string,
	variant?: string
): Promise<T> {
	const fixture = await loadFixtureDefinition(entity, name);
	const resolved = await loadFixture<T>(entity, name, { variant });

	// Apply preview-specific config
	if (fixture.usage?.preview?.additionalProps) {
		return { ...resolved.data, ...fixture.usage.preview.additionalProps } as T;
	}

	return resolved.data;
}

/**
 * Get list of variants available for a fixture.
 */
export async function getFixtureVariants(
	entity: string,
	name: string
): Promise<Array<{ name: string; description?: string }>> {
	const fixture = await loadFixtureDefinition(entity, name);

	if (!fixture.variants) {
		return [{ name: 'default', description: 'Base fixture' }];
	}

	const variants = [{ name: 'default', description: 'Base fixture' }];

	for (const [variantName, variant] of Object.entries(fixture.variants)) {
		variants.push({
			name: variantName,
			description: variant.description
		});
	}

	return variants;
}

/**
 * Get fixture usage info (which components/tests use it).
 */
export async function getFixtureUsage(
	entity: string,
	name: string
): Promise<{
	components: string[];
	tests: string[];
}> {
	const fixture = await loadFixtureDefinition(entity, name);

	return {
		components: fixture.usage?.preview?.components ?? [],
		tests: fixture.usage?.e2e?.tests ?? []
	};
}

// =============================================================================
// E2E TEST HELPERS
// =============================================================================

/**
 * Seed database with fixture data for e2e testing.
 *
 * @example
 * // In Playwright test setup
 * test.beforeEach(async ({ request }) => {
 *   await seedFixture('notification', 'notification-list', request);
 * });
 */
export async function seedFixture(
	entity: string,
	name: string,
	variant?: string,
	requestOrFetch?: typeof fetch
): Promise<boolean> {
	const fixture = await loadFixtureDefinition(entity, name);

	// Check if e2e seeding is enabled
	if (fixture.usage?.e2e?.seed === false) {
		console.warn(`Fixture ${entity}/${name} has seed disabled`);
		return false;
	}

	const resolved = await loadFixture(entity, name, { variant });

	// Call seed API
	const fetchFn = requestOrFetch ?? fetch;
	const response = await fetchFn('/api/fixtures/seed', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			entity,
			name,
			variant,
			collection: fixture.usage?.e2e?.collection ?? entity,
			data: resolved.data
		})
	});

	return response.ok;
}

/**
 * Cleanup seeded fixture data after e2e test.
 *
 * @example
 * // In Playwright test teardown
 * test.afterEach(async ({ request }) => {
 *   await cleanupFixture('notification', 'notification-list', request);
 * });
 */
export async function cleanupFixture(
	entity: string,
	name: string,
	requestOrFetch?: typeof fetch
): Promise<boolean> {
	const fixture = await loadFixtureDefinition(entity, name);

	// Check if cleanup is enabled (default true)
	if (fixture.usage?.e2e?.cleanup === false) {
		return true;
	}

	// Call cleanup API
	const fetchFn = requestOrFetch ?? fetch;
	const response = await fetchFn('/api/fixtures/cleanup', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			entity,
			name,
			collection: fixture.usage?.e2e?.collection ?? entity
		})
	});

	return response.ok;
}

// =============================================================================
// REGISTRATION
// =============================================================================

/**
 * Register a fixture directly (for server-side loading).
 *
 * @example
 * // In +page.server.ts
 * import { registerFixture } from './FixtureLoader';
 * import yaml from 'js-yaml';
 * import fs from 'fs/promises';
 *
 * export async function load() {
 *   const content = await fs.readFile('fixtures/user/current-user.yml', 'utf-8');
 *   const yamlData = yaml.load(content);
 *   registerFixture(yamlData);
 *
 *   return { ... };
 * }
 */
export function registerFixture(yaml: YAMLFixture): IFixture {
	const fixture = convertYAMLToFixture(yaml);
	fixtureRegistry.register(fixture);
	return fixture;
}

/**
 * Register multiple fixtures at once.
 */
export function registerFixtures(fixtures: YAMLFixture[]): IFixture[] {
	return fixtures.map(registerFixture);
}

// =============================================================================
// CACHE MANAGEMENT
// =============================================================================

/**
 * Clear fixture cache.
 */
export function clearFixtureCache(): void {
	fixtureCache.clear();
}

/**
 * Get cache stats for debugging.
 */
export function getFixtureCacheStats(): {
	size: number;
	keys: string[];
} {
	return {
		size: fixtureCache.size,
		keys: Array.from(fixtureCache.keys())
	};
}

// =============================================================================
// EXPORTS
// =============================================================================

export { fixtureRegistry } from './IFixture';
