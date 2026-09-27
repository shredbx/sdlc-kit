/**
 * Component Registry - Auto-Discovery of Presenters
 *
 * Uses Vite's import.meta.glob to automatically discover and register
 * all *.presenter.ts files in the components directory.
 *
 * This eliminates the need to manually import and register each presenter.
 */

import { presenterRegistry, type IPresenter } from './IPresenter';

/**
 * Auto-import all presenter files using Vite's glob pattern.
 * This runs at build time and includes all matching modules.
 */
const presenterModules = import.meta.glob<{ default: IPresenter }>(
	'../components/**/*.presenter.ts',
	{ eager: true }
);

/**
 * Register all discovered presenters.
 * This function is called once at module load time.
 */
function registerAllPresenters(): void {
	for (const path in presenterModules) {
		const module = presenterModules[path];
		if (module.default) {
			presenterRegistry.register(module.default);
		}
	}
}

// Auto-register on module load
registerAllPresenters();

/**
 * Get count of registered presenters by type.
 */
export function getPresenterCounts(): { components: number; packages: number } {
	const all = Array.from(presenterRegistry.presenters.values());
	return {
		components: all.filter((p) => p.presenterType === 'component').length,
		packages: all.filter((p) => p.presenterType === 'package').length
	};
}

/**
 * Get all registered component presenters grouped by category.
 */
export function getComponentsByCategory(): Map<string, IPresenter[]> {
	const grouped = new Map<string, IPresenter[]>();
	const components = presenterRegistry.getByType('component');

	for (const presenter of components) {
		const category = presenter.category;
		if (!grouped.has(category)) {
			grouped.set(category, []);
		}
		grouped.get(category)!.push(presenter);
	}

	return grouped;
}

/**
 * Get category counts for display.
 */
export function getCategoryCounts(): Record<string, number> {
	const counts: Record<string, number> = {};
	const components = presenterRegistry.getByType('component');

	for (const presenter of components) {
		const category = presenter.category;
		counts[category] = (counts[category] || 0) + 1;
	}

	return counts;
}

// Re-export the registry for convenience
export { presenterRegistry };
