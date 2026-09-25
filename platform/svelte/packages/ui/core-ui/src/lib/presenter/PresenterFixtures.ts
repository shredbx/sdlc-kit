/**
 * PresenterFixtures - Fixture integration for component previews
 *
 * Bridges the IFixture system with IPresenter for seamless fixture-driven
 * component previews. Enables the BUILD-ONCE-REUSE pattern.
 *
 * Usage Flow:
 * 1. Define fixture in YAML (projects/sbx/apps/sbx/fixtures/{entity}/{name}.yml)
 * 2. Create presenter with fixtureRefs
 * 3. Presenter loads fixture data automatically for previews
 * 4. Same fixtures usable in e2e tests
 *
 * @example
 * // In ChatPanel.presenter.ts
 * import { withFixtures } from './PresenterFixtures';
 *
 * const presenter = withFixtures({
 *   name: 'ChatPanel',
 *   slug: 'chat-panel',
 *   category: 'sections',
 *   presenterType: 'component',
 *   component: ChatPanel,
 *   fixtureRefs: [
 *     { entity: 'chat', name: 'conversation' },
 *     { entity: 'chat', name: 'conversation', variant: 'typing' }
 *   ]
 * });
 */

import type { IPresenter, IVariant } from './IPresenter';
import {
	loadFixture,
	loadPreviewFixture,
	getFixtureVariants,
	type FixtureLoadOptions
} from '../view/FixtureLoader';
import type { IFixtureRef, IResolvedFixture } from '../view/IFixture';

// =============================================================================
// TYPES
// =============================================================================

/**
 * Extended presenter with fixture support.
 */
export interface IPresenterWithFixtures extends IPresenter {
	/** References to fixtures that provide preview data */
	fixtureRefs?: IFixtureRef[];
}

/**
 * Fixture-loaded variant with resolved data.
 */
export interface IFixtureVariant extends IVariant {
	/** Fixture reference this variant came from */
	fixtureRef?: IFixtureRef;
	/** Resolved fixture data */
	fixtureData?: IResolvedFixture;
}

// =============================================================================
// VARIANT GENERATION
// =============================================================================

/**
 * Generate variants from fixture references.
 *
 * Each fixture variant becomes a presenter variant.
 */
export async function generateFixtureVariants(
	fixtureRefs: IFixtureRef[]
): Promise<IFixtureVariant[]> {
	const variants: IFixtureVariant[] = [];

	for (const ref of fixtureRefs) {
		const fixtureVariants = await getFixtureVariants(ref.entity, ref.fixture);

		for (const fv of fixtureVariants) {
			// Skip base fixture if we're asking for specific variant
			if (fv.name === 'default' && fixtureVariants.length > 1) {
				// Add base as first variant
				const baseData = await loadFixture(ref.entity, ref.fixture);
				variants.push({
					name: `${ref.fixture} (${fv.name})`,
					description: fv.description ?? `Base ${ref.fixture} fixture`,
					props: baseData.data as Record<string, unknown>,
					fixtureRef: ref,
					fixtureData: baseData
				});
			} else if (fv.name !== 'default') {
				// Add named variants
				const variantData = await loadFixture(ref.entity, ref.fixture, { variant: fv.name });
				variants.push({
					name: `${ref.fixture} (${fv.name})`,
					description: fv.description ?? `${fv.name} variant`,
					props: variantData.data as Record<string, unknown>,
					fixtureRef: { ...ref, use: ref.use },
					fixtureData: variantData
				});
			}
		}
	}

	return variants;
}

/**
 * Load single fixture as variant.
 */
export async function loadFixtureAsVariant(
	entity: string,
	name: string,
	variant?: string
): Promise<IFixtureVariant> {
	const data = await loadFixture(entity, name, { variant });

	return {
		name: variant ? `${name} (${variant})` : name,
		description: `Loaded from ${entity}/${name}` + (variant ? `#${variant}` : ''),
		props: data.data as Record<string, unknown>,
		fixtureRef: { entity, fixture: name },
		fixtureData: data
	};
}

// =============================================================================
// PRESENTER ENHANCEMENT
// =============================================================================

/**
 * Enhance presenter with fixture-driven variants.
 *
 * Adds fixtures as additional variants alongside manually defined ones.
 *
 * @example
 * export default await withFixtures({
 *   name: 'NotificationList',
 *   slug: 'notification-list',
 *   category: 'sections',
 *   presenterType: 'component',
 *   component: NotificationList,
 *   variants: [
 *     { name: 'Custom', props: { ... } }
 *   ],
 *   fixtureRefs: [
 *     { entity: 'notification', name: 'notification-list' }
 *   ]
 * });
 */
export async function withFixtures<T extends IPresenterWithFixtures>(
	presenter: T
): Promise<IPresenter> {
	if (!presenter.fixtureRefs || presenter.fixtureRefs.length === 0) {
		return presenter;
	}

	// Generate fixture variants
	const fixtureVariants = await generateFixtureVariants(presenter.fixtureRefs);

	// Combine with existing variants
	const existingVariants = presenter.variants ?? [];

	return {
		...presenter,
		variants: [...existingVariants, ...fixtureVariants]
	};
}

/**
 * Synchronous version for static presenter definitions.
 *
 * Returns presenter with fixtureRefs that will be resolved at runtime.
 * Use in conjunction with usePresenterFixtures() in preview component.
 */
export function definePresenterWithFixtures<T extends IPresenterWithFixtures>(presenter: T): T {
	return presenter;
}

// =============================================================================
// PREVIEW COMPONENT HELPERS
// =============================================================================

/**
 * Load fixture data for a presenter's preview.
 *
 * @example
 * // In preview component
 * <script>
 *   import { loadPresenterFixture } from './PresenterFixtures';
 *
 *   export let presenter;
 *   let fixtureData = null;
 *
 *   onMount(async () => {
 *     if (presenter.fixtureRefs?.[0]) {
 *       const ref = presenter.fixtureRefs[0];
 *       fixtureData = await loadPresenterFixture(ref);
 *     }
 *   });
 * </script>
 */
export async function loadPresenterFixture<T = Record<string, unknown>>(
	ref: IFixtureRef,
	options?: FixtureLoadOptions
): Promise<T> {
	const opts: FixtureLoadOptions = {
		...options,
		transform: ref.transform ?? options?.transform
	};

	return loadPreviewFixture<T>(ref.entity, ref.fixture, opts.variant);
}

/**
 * Load all fixture references for a presenter.
 */
export async function loadAllPresenterFixtures(
	presenter: IPresenterWithFixtures
): Promise<Map<string, IResolvedFixture>> {
	const results = new Map<string, IResolvedFixture>();

	if (!presenter.fixtureRefs) return results;

	for (const ref of presenter.fixtureRefs) {
		const key = `${ref.entity}:${ref.fixture}`;
		const data = await loadFixture(ref.entity, ref.fixture);
		results.set(key, data);
	}

	return results;
}

// =============================================================================
// VARIANT HELPERS
// =============================================================================

/**
 * Find variant by fixture name.
 */
export function findVariantByFixture(
	variants: IVariant[],
	entity: string,
	fixture: string
): IFixtureVariant | undefined {
	return variants.find((v) => {
		const fv = v as IFixtureVariant;
		return fv.fixtureRef?.entity === entity && fv.fixtureRef?.fixture === fixture;
	}) as IFixtureVariant | undefined;
}

/**
 * Filter variants to only fixture-based ones.
 */
export function getFixtureVariants(variants: IVariant[]): IFixtureVariant[] {
	return variants.filter((v) => (v as IFixtureVariant).fixtureRef !== undefined) as IFixtureVariant[];
}

/**
 * Filter variants to only manually defined ones.
 */
export function getManualVariants(variants: IVariant[]): IVariant[] {
	return variants.filter((v) => (v as IFixtureVariant).fixtureRef === undefined);
}

// =============================================================================
// FIXTURE DISCOVERY
// =============================================================================

/**
 * Get all fixtures that reference a component.
 *
 * Useful for finding all fixtures that can preview a component.
 */
export async function findFixturesForComponent(
	componentName: string
): Promise<Array<{ entity: string; name: string; variant?: string }>> {
	// This would query the fixture registry
	// For now, return empty - would need server endpoint
	console.warn('findFixturesForComponent requires server endpoint');
	return [];
}
